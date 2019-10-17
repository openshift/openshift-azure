package fakerp

import (
	"context"
	"strings"

	azuredns "github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2017-10-01/dns"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/dns"
)

type dnsManager struct {
	zonesClient      dns.ZonesClient
	recordSetsClient dns.RecordSetsClient
	dnsResourceGroup string
	dnsDomain        string
	log              *logrus.Entry
}

func newDNSManager(ctx context.Context, log *logrus.Entry, subscriptionID, dnsResourceGroup, dnsDomain string) (*dnsManager, error) {
	authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
	if err != nil {
		return nil, err
	}

	return &dnsManager{
		zonesClient:      dns.NewZonesClient(ctx, log, subscriptionID, authorizer),
		recordSetsClient: dns.NewRecordSetsClient(ctx, log, subscriptionID, authorizer),
		dnsResourceGroup: dnsResourceGroup,
		dnsDomain:        dnsDomain,
		log:              log,
	}, nil
}

func (dm *dnsManager) createOrUpdateDns(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	clusterRootDnsZone := strings.SplitN(cs.Properties.RouterProfiles[0].PublicSubdomain, ".", 2)[1]
	appsDnsZone := cs.Properties.RouterProfiles[0].PublicSubdomain
	consoleDnsRecordSet := "openshift"
	wildcardRecordSet := "*"
	clusterRg := cs.Properties.AzProfile.ResourceGroup
	dm.log.Debugf("creating cluster root dns zone %q", clusterRootDnsZone)
	err := dm.createOrUpdateZone(ctx, clusterRg, clusterRootDnsZone, dm.dnsResourceGroup, dm.dnsDomain)
	if err != nil {
		return err
	}

	if !cs.Properties.PrivateAPIServer {

		dm.log.Debugf("creating CNAME record set %q in dns zone %q", consoleDnsRecordSet, clusterRootDnsZone)
		_, err = dm.recordSetsClient.CreateOrUpdate(ctx, clusterRg, clusterRootDnsZone, "openshift", azuredns.CNAME, azuredns.RecordSet{
			RecordSetProperties: &azuredns.RecordSetProperties{
				CnameRecord: &azuredns.CnameRecord{
					Cname: &cs.Properties.FQDN,
				},
				TTL: to.Int64Ptr(60),
			},
		}, "", "")
	}

	dm.log.Debugf("creating apps dns zone %q", appsDnsZone)
	err = dm.createOrUpdateZone(ctx, clusterRg, appsDnsZone, clusterRg, clusterRootDnsZone)
	if err != nil {
		return err
	}

	dm.log.Debugf("creating CNAME record set %q in dns zone %q", wildcardRecordSet, appsDnsZone)
	_, err = dm.recordSetsClient.CreateOrUpdate(ctx, cs.Properties.AzProfile.ResourceGroup, cs.Properties.RouterProfiles[0].PublicSubdomain, wildcardRecordSet, azuredns.CNAME, azuredns.RecordSet{
		RecordSetProperties: &azuredns.RecordSetProperties{
			CnameRecord: &azuredns.CnameRecord{
				Cname: &cs.Properties.RouterProfiles[0].FQDN,
			},
			TTL: to.Int64Ptr(60),
		},
	}, "", "")

	return err
}

func (dm *dnsManager) createOrUpdateZone(ctx context.Context, dnsResourceGroup, zoneName, parentDnsZoneResourceGroup, parentDnsZoneName string) error {
	zone, err := dm.zonesClient.CreateOrUpdate(ctx, dnsResourceGroup, zoneName, azuredns.Zone{
		Location: to.StringPtr("global"),
	}, "", "")
	if err != nil {
		return err
	}

	// update TTLs
	rs, err := dm.recordSetsClient.Get(ctx, dnsResourceGroup, zoneName, "@", azuredns.SOA)
	if err != nil {
		return err
	}

	rs.RecordSetProperties.TTL = to.Int64Ptr(60)
	rs.RecordSetProperties.SoaRecord.RefreshTime = to.Int64Ptr(60)
	rs.RecordSetProperties.SoaRecord.RetryTime = to.Int64Ptr(60)
	rs.RecordSetProperties.SoaRecord.ExpireTime = to.Int64Ptr(60)
	rs.RecordSetProperties.SoaRecord.MinimumTTL = to.Int64Ptr(60)

	_, err = dm.recordSetsClient.CreateOrUpdate(ctx, dnsResourceGroup, zoneName, "@", azuredns.SOA, rs, "", "")
	if err != nil {
		return err
	}

	rs, err = dm.recordSetsClient.Get(ctx, dnsResourceGroup, zoneName, "@", azuredns.NS)
	if err != nil {
		return err
	}

	rs.RecordSetProperties.TTL = to.Int64Ptr(60)

	_, err = dm.recordSetsClient.CreateOrUpdate(ctx, dnsResourceGroup, zoneName, "@", azuredns.NS, rs, "", "")
	if err != nil {
		return err
	}

	nsRecords := make([]azuredns.NsRecord, len(*zone.NameServers))
	for i := range *zone.NameServers {
		nsRecords[i] = azuredns.NsRecord{
			Nsdname: &(*zone.NameServers)[i],
		}
	}

	relativeRecordSetName := strings.Split(zoneName, ".")[0]
	dm.log.Debugf("creating NS record set %q in dns zone %q", relativeRecordSetName, parentDnsZoneName)
	_, err = dm.recordSetsClient.CreateOrUpdate(ctx, parentDnsZoneResourceGroup, parentDnsZoneName, strings.Split(zoneName, ".")[0], azuredns.NS, azuredns.RecordSet{
		RecordSetProperties: &azuredns.RecordSetProperties{
			TTL:       to.Int64Ptr(60),
			NsRecords: &nsRecords,
		},
	}, "", "")

	return err
}

func (dm *dnsManager) deleteDns(ctx context.Context, cs *api.OpenShiftManagedCluster) error {
	appsDnsZone := cs.Properties.RouterProfiles[0].PublicSubdomain
	clusterRootDnsZone := strings.SplitN(appsDnsZone, ".", 2)[1]
	clusterRg := cs.Properties.AzProfile.ResourceGroup

	dm.log.Debugf("deleting apps dns zone %q", appsDnsZone)
	err := dm.deleteZone(ctx, clusterRg, appsDnsZone, clusterRg, clusterRootDnsZone)
	if err != nil {
		return err
	}
	dm.log.Debugf("deleting cluster root dns zone %q", clusterRootDnsZone)
	err = dm.deleteZone(ctx, clusterRg, clusterRootDnsZone, dm.dnsResourceGroup, dm.dnsDomain)
	if err != nil {
		return err
	}
	dm.log.Debugf("deleting NS record set %q in global root dns zone %q", clusterRg, dm.dnsDomain)
	_, err = dm.recordSetsClient.Delete(ctx, dm.dnsResourceGroup, dm.dnsDomain, clusterRg, azuredns.NS, "")
	return err
}

func (dm *dnsManager) deleteZone(ctx context.Context, dnsResourceGroup, zoneName, parentDnsZoneResourceGroup, parentDnsZoneName string) error {
	relativeRecordSetName := strings.Split(zoneName, ".")[0]
	dm.log.Debugf("deleting NS record set %q in parent dns zone %q", relativeRecordSetName, parentDnsZoneName)
	_, err := dm.recordSetsClient.Delete(ctx, parentDnsZoneResourceGroup, parentDnsZoneName, relativeRecordSetName, azuredns.NS, "")
	if err != nil {
		return err
	}

	return dm.zonesClient.Delete(ctx, dnsResourceGroup, zoneName, "")
}
