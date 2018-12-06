package fakerp

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2017-10-01/dns"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/sirupsen/logrus"

	v20180930preview "github.com/openshift/openshift-azure/pkg/api/2018-09-30-preview/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

func CreateOCPDNS(ctx context.Context, log *logrus.Entry, subscriptionID, resourceGroup, dnsResourceGroup, dnsDomain string, oc *v20180930preview.OpenShiftManagedCluster) error {
	authorizer, err := azureclient.NewAuthorizerFromContext(ctx)
	if err != nil {
		return err
	}

	// create clients
	zc := dns.NewZonesClient(subscriptionID)
	zc.Authorizer = authorizer
	rsc := dns.NewRecordSetsClient(subscriptionID)
	rsc.Authorizer = authorizer

	// dns zone object
	z := dns.Zone{
		Location: to.StringPtr("global"),
	}
	zoneName := fmt.Sprintf("%s.%s", resourceGroup, dnsDomain)
	log.Infof("create dns zone %s", zoneName)
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
				Email:        to.StringPtr("azuredns-hostmaster.microsoft.com"),
				RefreshTime:  to.Int64Ptr(60),
				RetryTime:    to.Int64Ptr(60),
				ExpireTime:   to.Int64Ptr(60),
				MinimumTTL:   to.Int64Ptr(60),
				SerialNumber: to.Int64Ptr(1),
			},
		},
	}
	defaultRecordName := "@"
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
	routerCName := fmt.Sprintf("%s-router.%s.%s", resourceGroup, *oc.Location, "cloudapp.azure.com")
	log.Infof("create dns router cname %s", routerCName)
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
	return err
}

func DeleteOCPDNS(ctx context.Context, subscriptionID, resourceGroup, dnsResourceGroup, dnsDomain string) error {
	authorizer, err := azureclient.NewAuthorizerFromContext(ctx)
	if err != nil {
		return err
	}

	// delete zone
	zc := dns.NewZonesClient(subscriptionID)
	zc.Authorizer = authorizer
	rsc := dns.NewRecordSetsClient(subscriptionID)
	rsc.Authorizer = authorizer

	zoneName := fmt.Sprintf("%s.%s", resourceGroup, dnsDomain)
	future, err := zc.Delete(ctx, dnsResourceGroup, zoneName, "")
	if err != nil {
		return err
	}

	if err := future.WaitForCompletionRef(ctx, zc.Client); err != nil {
		return err
	}

	// delete main zone NS record
	_, err = rsc.Delete(ctx, dnsResourceGroup, dnsDomain+"-test", resourceGroup, dns.NS, "")
	return err
}
