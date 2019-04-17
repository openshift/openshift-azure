package fakerp

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2017-10-01/dns"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

type dnsManager struct {
	zc               azureclient.ZonesClient
	rsc              azureclient.RecordSetsClient
	dnsResourceGroup string
	dnsDomain        string
}

func newDNSManager(ctx context.Context, log *logrus.Entry, subscriptionID, dnsResourceGroup, dnsDomain string) (*dnsManager, error) {
	authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
	if err != nil {
		return nil, err
	}

	return &dnsManager{
		zc:               azureclient.NewZonesClient(ctx, log, subscriptionID, authorizer),
		rsc:              azureclient.NewRecordSetsClient(ctx, log, subscriptionID, authorizer),
		dnsResourceGroup: dnsResourceGroup,
		dnsDomain:        dnsDomain,
	}, nil
}

func (dm *dnsManager) createOrUpdateZone(ctx context.Context, resourceGroup, zoneName, parentResourceGroup, parentZoneName string) error {
	zone, err := dm.zc.CreateOrUpdate(ctx, resourceGroup, zoneName, dns.Zone{
		Location: to.StringPtr("global"),
	}, "", "")
	if err != nil {
		return err
	}

	// update TTLs
	rs, err := dm.rsc.Get(ctx, resourceGroup, zoneName, "@", dns.SOA)
	if err != nil {
		return err
	}

	rs.RecordSetProperties.TTL = to.Int64Ptr(60)
	rs.RecordSetProperties.SoaRecord.RefreshTime = to.Int64Ptr(60)
	rs.RecordSetProperties.SoaRecord.RetryTime = to.Int64Ptr(60)
	rs.RecordSetProperties.SoaRecord.ExpireTime = to.Int64Ptr(60)
	rs.RecordSetProperties.SoaRecord.MinimumTTL = to.Int64Ptr(60)

	_, err = dm.rsc.CreateOrUpdate(ctx, resourceGroup, zoneName, "@", dns.SOA, rs, "", "")
	if err != nil {
		return err
	}

	rs, err = dm.rsc.Get(ctx, resourceGroup, zoneName, "@", dns.NS)
	if err != nil {
		return err
	}

	rs.RecordSetProperties.TTL = to.Int64Ptr(60)

	_, err = dm.rsc.CreateOrUpdate(ctx, resourceGroup, zoneName, "@", dns.NS, rs, "", "")
	if err != nil {
		return err
	}

	nsRecords := make([]dns.NsRecord, len(*zone.NameServers))
	for i := range *zone.NameServers {
		nsRecords[i] = dns.NsRecord{
			Nsdname: &(*zone.NameServers)[i],
		}
	}

	// create glue record in parent zone
	_, err = dm.rsc.CreateOrUpdate(ctx, parentResourceGroup, parentZoneName, strings.Split(zoneName, ".")[0], dns.NS, dns.RecordSet{
		RecordSetProperties: &dns.RecordSetProperties{
			TTL:       to.Int64Ptr(60),
			NsRecords: &nsRecords,
		},
	}, "", "")

	return err
}

func (dm *dnsManager) deleteZone(ctx context.Context, resourceGroup, zoneName, parentResourceGroup, parentZoneName string) error {
	// delete glue record in parent zone
	_, err := dm.rsc.Delete(ctx, parentResourceGroup, parentZoneName, strings.Split(zoneName, ".")[0], dns.NS, "")
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
	_, err = dm.rsc.CreateOrUpdate(ctx, cs.Properties.AzProfile.ResourceGroup, parentZone, "openshift", dns.CNAME, dns.RecordSet{
		RecordSetProperties: &dns.RecordSetProperties{
			CnameRecord: &dns.CnameRecord{
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
	_, err = dm.rsc.CreateOrUpdate(ctx, cs.Properties.AzProfile.ResourceGroup, cs.Properties.RouterProfiles[0].PublicSubdomain, "*", dns.CNAME, dns.RecordSet{
		RecordSetProperties: &dns.RecordSetProperties{
			CnameRecord: &dns.CnameRecord{
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
