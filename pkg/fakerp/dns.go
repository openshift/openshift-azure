package fakerp

import (
	"context"
	"fmt"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/openshift/openshift-azure/pkg/api"
	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2017-10-01/dns"
)

func CreateOCPDNS(ctx context.Context, subscriptionID, resourceGroup, dnsResourceGroup, dnsDomain string, config *api.PluginConfig, oc *v20180930preview.OpenShiftManagedCluster) error {
	zoneName := fmt.Sprintf("%s.%s", resourceGroup, dnsDomain)
	zoneLocation := "global"
	routerCName := fmt.Sprintf("%s-router.%s.%s", resourceGroup, oc.Location, "cloudapp.azure.com")
	soaEmail := "azuredns-hostmaster.microsoft.com"
	defaultRecordName := "@"

	authorizer, err := azureclient.NewAuthorizerFromContext(ctx)
	if err != nil {
		return err
	}

	// create clients
	zc := azureclient.NewZonesClient(subscriptionID, authorizer, config.AcceptLanguages)
	rsc := azureclient.NewRecordSetsClient(subscriptionID, authorizer, config.AcceptLanguages)

	// dns zone object
	z := dns.Zone{
		Location: &zoneLocation,
	}
	zone, err := zc.CreateOrUpdate(ctx, dnsResourceGroup, zoneName, z, "", "")
	if err != nil {
		return err
	}

	// construct namesever list for NS update
	nsServer := *zone.NameServers
	nsServerList := []dns.NsRecord{}
	for _, r := range nsServer {
		t := r
		rec := dns.NsRecord{
			Nsdname: &t,
		}
		nsServerList = append(nsServerList, rec)
	}

	// update default SOA record
	soa := dns.RecordSet{
		Etag: zone.Etag,
		RecordSetProperties: &dns.RecordSetProperties{
			TTL: to.Int64Ptr(3600),
			SoaRecord: &dns.SoaRecord{
				Host:         &nsServer[0],
				Email:        &soaEmail,
				RefreshTime:  to.Int64Ptr(60),
				RetryTime:    to.Int64Ptr(60),
				ExpireTime:   to.Int64Ptr(60),
				MinimumTTL:   to.Int64Ptr(60),
				SerialNumber: to.Int64Ptr(1),
			},
		},
	}
	_, err = rsc.CreateOrUpdate(ctx, dnsResourceGroup, zoneName, defaultRecordName, dns.SOA, soa, "", "")
	if err != nil {
		return err
	}

	// update default NS record
	ns := dns.RecordSet{
		Etag: zone.Etag,
		RecordSetProperties: &dns.RecordSetProperties{
			TTL:       to.Int64Ptr(60),
			NsRecords: &nsServerList,
		},
	}
	_, err = rsc.CreateOrUpdate(ctx, dnsResourceGroup, zoneName, defaultRecordName, dns.NS, ns, "", "")
	if err != nil {
		return err
	}

	// create router wildcard DNS CName record
	cn := dns.RecordSet{
		Etag: zone.Etag,
		RecordSetProperties: &dns.RecordSetProperties{
			CnameRecord: &dns.CnameRecord{
				Cname: &routerCName,
			},
			TTL: to.Int64Ptr(3600),
		},
	}
	_, err = rsc.CreateOrUpdate(ctx, dnsResourceGroup, zoneName, "*", dns.CNAME, cn, "", "")
	if err != nil {
		return err
	}

	// update main NS records
	ns = dns.RecordSet{
		RecordSetProperties: &dns.RecordSetProperties{
			TTL:       to.Int64Ptr(60),
			NsRecords: &nsServerList,
		},
	}
	_, err = rsc.CreateOrUpdate(ctx, dnsResourceGroup, dnsDomain, resourceGroup, dns.NS, ns, "", "")
	if err != nil {
		return err
	}

	return nil
}

func DeleteOCPDNS(ctx context.Context, subscriptionID, resourceGroup, dnsResourceGroup, dnsDomain string, config *api.PluginConfig) error {
	zoneName := fmt.Sprintf("%s.%s", resourceGroup, dnsDomain)
	authorizer, err := azureclient.NewAuthorizerFromContext(ctx)
	if err != nil {
		return err
	}

	// delete zone
	zc := azureclient.NewZonesClient(subscriptionID, authorizer, config.AcceptLanguages)
	rsc := azureclient.NewRecordSetsClient(subscriptionID, authorizer, config.AcceptLanguages)

	future, err := zc.Delete(ctx, dnsResourceGroup, zoneName, "")
	if err != nil {
		return err
	}

	if err := future.WaitForCompletionRef(ctx, zc.Client()); err != nil {
		return err
	}

	// delete main zone NS record
	_, err = rsc.Delete(ctx, dnsResourceGroup, dnsDomain+"-test", resourceGroup, dns.NS, "")
	if err != nil {
		return err
	}

	return nil
}
