package fakerp

import (
	"context"
	"strings"

	azdns "github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2017-10-01/dns"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/dns"
)

type dnsManager struct {
	zc               dns.ZonesClient
	rsc              dns.RecordSetsClient
	dnsResourceGroup string
	dnsDomain        string
}

func newDNSManager(ctx context.Context, log *logrus.Entry, subscriptionID, dnsResourceGroup, dnsDomain string) (*dnsManager, error) {
	authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
	if err != nil {
		return nil, err
	}

	return &dnsManager{
		zc:               dns.NewZonesClient(ctx, log, subscriptionID, authorizer),
		rsc:              dns.NewRecordSetsClient(ctx, log, subscriptionID, authorizer),
		dnsResourceGroup: dnsResourceGroup,
		dnsDomain:        dnsDomain,
	}, nil
}

func (dm *dnsManager) createOrUpdateZone(ctx context.Context, resourceGroup, zoneName, parentResourceGroup, parentZoneName string) error {
	zone, err := dm.zc.CreateOrUpdate(ctx, resourceGroup, zoneName, azdns.Zone{
		Location: to.StringPtr("global"),
	}, "", "")
	if err != nil {
		return err
	}

	// update TTLs
	rs, err := dm.rsc.Get(ctx, resourceGroup, zoneName, "@", azdns.SOA)
	if err != nil {
		return err
	}

	rs.RecordSetProperties.TTL = to.Int64Ptr(60)
	rs.RecordSetProperties.SoaRecord.RefreshTime = to.Int64Ptr(60)
	rs.RecordSetProperties.SoaRecord.RetryTime = to.Int64Ptr(60)
	rs.RecordSetProperties.SoaRecord.ExpireTime = to.Int64Ptr(60)
	rs.RecordSetProperties.SoaRecord.MinimumTTL = to.Int64Ptr(60)

	_, err = dm.rsc.CreateOrUpdate(ctx, resourceGroup, zoneName, "@", azdns.SOA, rs, "", "")
	if err != nil {
		return err
	}

	rs, err = dm.rsc.Get(ctx, resourceGroup, zoneName, "@", azdns.NS)
	if err != nil {
		return err
	}

	rs.RecordSetProperties.TTL = to.Int64Ptr(60)

	_, err = dm.rsc.CreateOrUpdate(ctx, resourceGroup, zoneName, "@", azdns.NS, rs, "", "")
	if err != nil {
		return err
	}

	nsRecords := make([]azdns.NsRecord, len(*zone.NameServers))
	for i := range *zone.NameServers {
		nsRecords[i] = azdns.NsRecord{
			Nsdname: &(*zone.NameServers)[i],
		}
	}

	// create glue record in parent zone
	_, err = dm.rsc.CreateOrUpdate(ctx, parentResourceGroup, parentZoneName, strings.Split(zoneName, ".")[0], azdns.NS, azdns.RecordSet{
		RecordSetProperties: &azdns.RecordSetProperties{
			TTL:       to.Int64Ptr(60),
			NsRecords: &nsRecords,
		},
	}, "", "")

	return err
}

func (dm *dnsManager) deleteZone(ctx context.Context, resourceGroup, zoneName, parentResourceGroup, parentZoneName string) error {
	// delete glue record in parent zone
	_, err := dm.rsc.Delete(ctx, parentResourceGroup, parentZoneName, strings.Split(zoneName, ".")[0], azdns.NS, "")
	if err != nil {
		return err
	}

	return dm.zc.Delete(ctx, resourceGroup, zoneName, "")
}

func (dm *dnsManager) createOrUpdateOCPDNS(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	parentZone := strings.SplitN(cs.Properties.RouterProfiles[0].PublicSubdomain, ".", 2)[1]

	// <random>.osacloud.dev zone
	err := dm.createOrUpdateZone(ctx, cs.Properties.AzProfile.ResourceGroup, parentZone, dm.dnsResourceGroup, dm.dnsDomain)
	if err != nil {
		return err
	}

	// openshift.<random>.osacloud.dev cname
	_, err = dm.rsc.CreateOrUpdate(ctx, cs.Properties.AzProfile.ResourceGroup, parentZone, "openshift", azdns.CNAME, azdns.RecordSet{
		RecordSetProperties: &azdns.RecordSetProperties{
			CnameRecord: &azdns.CnameRecord{
				Cname: &cs.Properties.FQDN,
			},
			TTL: to.Int64Ptr(60),
		},
	}, "", "")

	// apps.<random>.osacloud.dev zone
	err = dm.createOrUpdateZone(ctx, cs.Properties.AzProfile.ResourceGroup, cs.Properties.RouterProfiles[0].PublicSubdomain, cs.Properties.AzProfile.ResourceGroup, parentZone)
	if err != nil {
		return err
	}

	// *.apps.<random>.osacloud.dev cname
	_, err = dm.rsc.CreateOrUpdate(ctx, cs.Properties.AzProfile.ResourceGroup, cs.Properties.RouterProfiles[0].PublicSubdomain, "*", azdns.CNAME, azdns.RecordSet{
		RecordSetProperties: &azdns.RecordSetProperties{
			CnameRecord: &azdns.CnameRecord{
				Cname: &cs.Properties.RouterProfiles[0].FQDN,
			},
			TTL: to.Int64Ptr(60),
		},
	}, "", "")

	return err
}

func (dm *dnsManager) deleteOCPDNS(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	parentZone := strings.SplitN(cs.Properties.RouterProfiles[0].PublicSubdomain, ".", 2)[1]

	// apps.<random>.osacloud.dev zone
	err := dm.deleteZone(ctx, cs.Properties.AzProfile.ResourceGroup, cs.Properties.RouterProfiles[0].PublicSubdomain, cs.Properties.AzProfile.ResourceGroup, parentZone)
	if err != nil {
		return err
	}

	// <random>.osacloud.dev zone
	return dm.deleteZone(ctx, cs.Properties.AzProfile.ResourceGroup, parentZone, dm.dnsResourceGroup, dm.dnsDomain)
}
