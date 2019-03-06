package fakerp

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2017-10-01/dns"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

// CreateOCPDNS creates the dns zone for the cluster, updates the main zone in dnsResourceGroup and returns
// the generated publicSubdomain, routerPrefix
func CreateOCPDNS(ctx context.Context, subscriptionID, resourceGroup, location, dnsResourceGroup, dnsDomain, zoneName, routerCName string) error {
	authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
	if err != nil {
		return err
	}

	// create clients
	zc := azureclient.NewZonesClient(ctx, subscriptionID, authorizer)
	rsc := azureclient.NewRecordSetsClient(ctx, subscriptionID, authorizer)

	// dns zone object
	z := dns.Zone{
		Location: to.StringPtr("global"),
	}

	// This creates creates the new zone
	zone, err := zc.CreateOrUpdate(ctx, resourceGroup, zoneName, z, "", "")
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
	_, err = rsc.CreateOrUpdate(ctx, resourceGroup, zoneName, defaultRecordName, dns.SOA, soa, "", "")
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
	_, err = rsc.CreateOrUpdate(ctx, resourceGroup, zoneName, defaultRecordName, dns.NS, ns, "", "")
	if err != nil {
		return err
	}

	cn := dns.RecordSet{
		Etag: zone.Etag,
		RecordSetProperties: &dns.RecordSetProperties{
			CnameRecord: &dns.CnameRecord{
				Cname: &routerCName,
			},
			TTL: to.Int64Ptr(3600),
		},
	}
	_, err = rsc.CreateOrUpdate(ctx, resourceGroup, zoneName, "*", dns.CNAME, cn, "", "")
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
	_, err = rsc.CreateOrUpdate(ctx, dnsResourceGroup, dnsDomain, strings.Split(zoneName, ".")[0], dns.NS, ns, "", "")
	return err
}

func DeleteOCPDNS(ctx context.Context, subscriptionID, resourceGroup, dnsResourceGroup, dnsDomain string) error {
	authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
	if err != nil {
		return err
	}

	// delete zone
	zc := azureclient.NewZonesClient(ctx, subscriptionID, authorizer)
	rsc := azureclient.NewRecordSetsClient(ctx, subscriptionID, authorizer)
	zones, err := zc.ListByResourceGroup(ctx, resourceGroup, to.Int32Ptr(100))
	for _, z := range zones {
		zoneName := to.String(z.Name)
		// delete main zone NS record
		rscDelete, err := rsc.Delete(ctx, dnsResourceGroup, dnsDomain, zoneName, dns.NS, "")
		if err != nil {
			return err
		}

		if rscDelete.StatusCode != 204 {
			return fmt.Errorf("error(%d) removing %v from %v in %v", rscDelete.StatusCode, strings.Split(zoneName, ".")[0], dnsDomain, dnsResourceGroup)
		}

		err = zc.Delete(ctx, dnsResourceGroup, zoneName, "")
		if err != nil {
			return err
		}
	}

	return nil
}
