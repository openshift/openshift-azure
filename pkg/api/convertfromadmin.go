package api

import (
	"errors"

	admin "github.com/openshift/openshift-azure/pkg/api/admin/api"
)

func ConvertFromAdmin(oc *admin.OpenShiftManagedCluster, old *OpenShiftManagedCluster) (*OpenShiftManagedCluster, error) {
	cs := &OpenShiftManagedCluster{}
	if old != nil {
		cs = old.DeepCopy()
	}
	if oc.ID != nil {
		cs.ID = *oc.ID
	}
	if oc.Name != nil {
		cs.Name = *oc.Name
	}
	if oc.Type != nil {
		cs.Type = *oc.Type
	}
	if oc.Location != nil {
		cs.Location = *oc.Location
	}
	if cs.Tags == nil {
		cs.Tags = make(map[string]string, len(oc.Tags))
	}
	for k, v := range oc.Tags {
		if v != nil {
			cs.Tags[k] = *v
		}
	}

	if oc.Plan != nil {
		mergeResourcePurchasePlanAdmin(oc, cs)
	}

	if oc.Properties != nil {
		if err := mergePropertiesAdmin(oc, cs); err != nil {
			return nil, err
		}
	}

	if oc.Config != nil {
		mergeConfig(oc, cs)
	}

	return cs, nil
}

func mergeResourcePurchasePlanAdmin(oc *admin.OpenShiftManagedCluster, cs *OpenShiftManagedCluster) {
	if oc.Plan.Name != nil {
		cs.Plan.Name = *oc.Plan.Name
	}
	if oc.Plan.Product != nil {
		cs.Plan.Product = *oc.Plan.Product
	}
	if oc.Plan.PromotionCode != nil {
		cs.Plan.PromotionCode = *oc.Plan.PromotionCode
	}
	if oc.Plan.Publisher != nil {
		cs.Plan.Publisher = *oc.Plan.Publisher
	}
}

func mergePropertiesAdmin(oc *admin.OpenShiftManagedCluster, cs *OpenShiftManagedCluster) error {
	if oc.Properties.ProvisioningState != nil {
		cs.Properties.ProvisioningState = ProvisioningState(*oc.Properties.ProvisioningState)
	}
	if oc.Properties.OpenShiftVersion != nil {
		cs.Properties.OpenShiftVersion = *oc.Properties.OpenShiftVersion
	}
	if oc.Properties.PublicHostname != nil {
		cs.Properties.PublicHostname = *oc.Properties.PublicHostname
	}
	if oc.Properties.FQDN != nil {
		cs.Properties.FQDN = *oc.Properties.FQDN
	}

	if oc.Properties.NetworkProfile != nil {
		if oc.Properties.NetworkProfile.VnetCIDR != nil {
			cs.Properties.NetworkProfile.VnetCIDR = *oc.Properties.NetworkProfile.VnetCIDR
		}
		if oc.Properties.NetworkProfile.PeerVnetID != nil {
			cs.Properties.NetworkProfile.PeerVnetID = *oc.Properties.NetworkProfile.PeerVnetID
		}
	}

	if err := mergeRouterProfilesAdmin(oc, cs); err != nil {
		return err
	}

	if err := mergeAgentPoolProfilesAdmin(oc, cs); err != nil {
		return err
	}

	if err := mergeAuthProfileAdmin(oc, cs); err != nil {
		return err
	}

	if oc.Properties.ServicePrincipalProfile != nil {
		if oc.Properties.ServicePrincipalProfile.ClientID != nil {
			cs.Properties.ServicePrincipalProfile.ClientID = *oc.Properties.ServicePrincipalProfile.ClientID
		}
		if oc.Properties.ServicePrincipalProfile.Secret != nil {
			cs.Properties.ServicePrincipalProfile.Secret = *oc.Properties.ServicePrincipalProfile.Secret
		}
	}

	if oc.Properties.AzProfile != nil {
		if oc.Properties.AzProfile.TenantID != nil {
			cs.Properties.AzProfile.TenantID = *oc.Properties.AzProfile.TenantID
		}
		if oc.Properties.AzProfile.SubscriptionID != nil {
			cs.Properties.AzProfile.SubscriptionID = *oc.Properties.AzProfile.SubscriptionID
		}
		if oc.Properties.AzProfile.ResourceGroup != nil {
			cs.Properties.AzProfile.ResourceGroup = *oc.Properties.AzProfile.ResourceGroup
		}
	}

	return nil
}

func mergeRouterProfilesAdmin(oc *admin.OpenShiftManagedCluster, cs *OpenShiftManagedCluster) error {
	if cs.Properties.RouterProfiles == nil && len(oc.Properties.RouterProfiles) > 0 {
		cs.Properties.RouterProfiles = make([]RouterProfile, 0, len(oc.Properties.RouterProfiles))
	}
	for _, rp := range oc.Properties.RouterProfiles {
		if rp.Name == nil || *rp.Name == "" {
			return errors.New("invalid router profile - name is missing")
		}

		index := routerProfileIndex(*rp.Name, cs.Properties.RouterProfiles)
		// If the requested profile does not exist, add it
		// in cs as is, otherwise merge it in the existing
		// profile.
		if index == -1 {
			cs.Properties.RouterProfiles = append(cs.Properties.RouterProfiles, convertRouterProfileAdmin(rp, nil))
		} else {
			head := append(cs.Properties.RouterProfiles[:index], convertRouterProfileAdmin(rp, &cs.Properties.RouterProfiles[index]))
			cs.Properties.RouterProfiles = append(head, cs.Properties.RouterProfiles[index+1:]...)
		}
	}
	return nil
}

func convertRouterProfileAdmin(in admin.RouterProfile, old *RouterProfile) (out RouterProfile) {
	if old != nil {
		out = *old
	}
	if in.Name != nil {
		out.Name = *in.Name
	}
	if in.PublicSubdomain != nil {
		out.PublicSubdomain = *in.PublicSubdomain
	}
	if in.FQDN != nil {
		out.FQDN = *in.FQDN
	}
	return
}

func mergeAgentPoolProfilesAdmin(oc *admin.OpenShiftManagedCluster, cs *OpenShiftManagedCluster) error {
	if cs.Properties.AgentPoolProfiles == nil && len(oc.Properties.AgentPoolProfiles) > 0 {
		cs.Properties.AgentPoolProfiles = make([]AgentPoolProfile, 0, len(oc.Properties.AgentPoolProfiles)+1)
	}

	for _, in := range oc.Properties.AgentPoolProfiles {
		if in.Name == nil || *in.Name == "" {
			return errors.New("invalid agent pool profile - name is missing")
		}
		index := agentPoolProfileIndex(*in.Name, cs.Properties.AgentPoolProfiles)
		// If the requested profile does not exist, add it
		// in cs as is, otherwise merge it in the existing
		// profile.
		if index == -1 {
			cs.Properties.AgentPoolProfiles = append(cs.Properties.AgentPoolProfiles, convertAgentPoolProfileAdmin(in, nil))
		} else {
			head := append(cs.Properties.AgentPoolProfiles[:index], convertAgentPoolProfileAdmin(in, &cs.Properties.AgentPoolProfiles[index]))
			cs.Properties.AgentPoolProfiles = append(head, cs.Properties.AgentPoolProfiles[index+1:]...)
		}
	}
	return nil
}

func convertAgentPoolProfileAdmin(in admin.AgentPoolProfile, old *AgentPoolProfile) (out AgentPoolProfile) {
	if old != nil {
		out = *old
	}
	if in.Name != nil {
		out.Name = *in.Name
	}
	if in.Count != nil {
		out.Count = *in.Count
	}
	if in.VMSize != nil {
		out.VMSize = VMSize(*in.VMSize)
	}
	if in.SubnetCIDR != nil {
		out.SubnetCIDR = *in.SubnetCIDR
	}
	if in.OSType != nil {
		out.OSType = OSType(*in.OSType)
	}
	if in.Role != nil {
		out.Role = AgentPoolProfileRole(*in.Role)
	}
	return
}

func mergeAuthProfileAdmin(oc *admin.OpenShiftManagedCluster, cs *OpenShiftManagedCluster) error {
	if oc.Properties.AuthProfile == nil {
		return nil
	}

	if cs.Properties.AuthProfile.IdentityProviders == nil {
		cs.Properties.AuthProfile.IdentityProviders = make([]IdentityProvider, 0, len(oc.Properties.AuthProfile.IdentityProviders))
	}

	for _, ip := range oc.Properties.AuthProfile.IdentityProviders {
		if ip.Name == nil || *ip.Name == "" {
			return errors.New("invalid identity provider - name is missing")
		}
		index := identityProviderIndex(*ip.Name, cs.Properties.AuthProfile.IdentityProviders)
		// If the requested provider does not exist, add it
		// in cs as is, otherwise merge it in the existing
		// provider.
		if index == -1 {
			cs.Properties.AuthProfile.IdentityProviders = append(cs.Properties.AuthProfile.IdentityProviders, convertIdentityProviderAdmin(ip, nil))
		} else {
			provider := cs.Properties.AuthProfile.IdentityProviders[index].Provider
			switch out := provider.(type) {
			case (*AADIdentityProvider):
				in := ip.Provider.(*admin.AADIdentityProvider)
				if in.Kind != nil {
					if out.Kind != "" && out.Kind != *in.Kind {
						return errors.New("cannot update the kind of the identity provider")
					}
				}
			default:
				return errors.New("authProfile.identityProviders conversion failed")
			}
			head := append(cs.Properties.AuthProfile.IdentityProviders[:index], convertIdentityProviderAdmin(ip, &cs.Properties.AuthProfile.IdentityProviders[index]))
			cs.Properties.AuthProfile.IdentityProviders = append(head, cs.Properties.AuthProfile.IdentityProviders[index+1:]...)
		}
	}
	return nil
}

func convertIdentityProviderAdmin(in admin.IdentityProvider, old *IdentityProvider) (out IdentityProvider) {
	if old != nil {
		out = *old
	}
	if in.Name != nil {
		out.Name = *in.Name
	}
	if in.Provider != nil {
		switch provider := in.Provider.(type) {
		case *admin.AADIdentityProvider:
			p := &AADIdentityProvider{}
			if out.Provider != nil {
				p = out.Provider.(*AADIdentityProvider)
			}
			if provider.Kind != nil {
				p.Kind = *provider.Kind
			}
			if provider.ClientID != nil {
				p.ClientID = *provider.ClientID
			}
			if provider.Secret != nil {
				p.Secret = *provider.Secret
			}
			if provider.TenantID != nil {
				p.TenantID = *provider.TenantID
			}
			out.Provider = p

		default:
			panic("authProfile.identityProviders conversion failed")
		}
	}
	return
}

func mergeConfig(oc *admin.OpenShiftManagedCluster, cs *OpenShiftManagedCluster) {
	in, out := oc.Config, cs.Config

	if in.ImageOffer != nil {
		out.ImageOffer = *in.ImageOffer
	}
	if in.ImagePublisher != nil {
		out.ImagePublisher = *in.ImagePublisher
	}
	if in.ImageSKU != nil {
		out.ImageSKU = *in.ImageSKU
	}
	if in.ImageVersion != nil {
		out.ImageVersion = *in.ImageVersion
	}
	if in.SSHKey != nil {
		out.SSHKey = in.SSHKey
	}
	if in.ConfigStorageAccount != nil {
		out.ConfigStorageAccount = *in.ConfigStorageAccount
	}
	if in.RegistryStorageAccount != nil {
		out.RegistryStorageAccount = *in.RegistryStorageAccount
	}
	if in.Certificates != nil {
		mergeCertificateConfig(in.Certificates, &out.Certificates)
	}
	if in.Images != nil {
		mergeImageConfig(in.Images, &out.Images)
	}
	if in.AdminKubeconfig != nil {
		out.AdminKubeconfig = in.AdminKubeconfig.DeepCopy()
	}
	if in.MasterKubeconfig != nil {
		out.MasterKubeconfig = in.MasterKubeconfig.DeepCopy()
	}
	if in.NodeBootstrapKubeconfig != nil {
		out.NodeBootstrapKubeconfig = in.NodeBootstrapKubeconfig.DeepCopy()
	}
	if in.AzureClusterReaderKubeconfig != nil {
		out.AzureClusterReaderKubeconfig = in.AzureClusterReaderKubeconfig.DeepCopy()
	}
	if in.ServiceAccountKey != nil {
		out.ServiceAccountKey = in.ServiceAccountKey
	}
	if len(in.SessionSecretAuth) > 0 {
		out.SessionSecretAuth = in.SessionSecretAuth
	}
	if len(in.SessionSecretEnc) > 0 {
		out.SessionSecretEnc = in.SessionSecretEnc
	}
	if in.RunningUnderTest != nil {
		out.RunningUnderTest = *in.RunningUnderTest
	}
	if len(in.HtPasswd) > 0 {
		out.HtPasswd = in.HtPasswd
	}
	if in.CustomerAdminPasswd != nil {
		out.CustomerAdminPasswd = *in.CustomerAdminPasswd
	}
	if in.CustomerReaderPasswd != nil {
		out.CustomerReaderPasswd = *in.CustomerReaderPasswd
	}
	if in.EndUserPasswd != nil {
		out.EndUserPasswd = *in.EndUserPasswd
	}
	if len(in.RegistryHTTPSecret) > 0 {
		out.RegistryHTTPSecret = in.RegistryHTTPSecret
	}
	if len(in.PrometheusProxySessionSecret) > 0 {
		out.PrometheusProxySessionSecret = in.PrometheusProxySessionSecret
	}
	if len(in.AlertManagerProxySessionSecret) > 0 {
		out.AlertManagerProxySessionSecret = in.AlertManagerProxySessionSecret
	}
	if len(in.AlertsProxySessionSecret) > 0 {
		out.AlertsProxySessionSecret = in.AlertsProxySessionSecret
	}
	if in.RegistryConsoleOAuthSecret != nil {
		out.RegistryConsoleOAuthSecret = *in.RegistryConsoleOAuthSecret
	}
	if in.ConsoleOAuthSecret != nil {
		out.ConsoleOAuthSecret = *in.ConsoleOAuthSecret
	}
	if in.RouterStatsPassword != nil {
		out.RouterStatsPassword = *in.RouterStatsPassword
	}
	if in.ServiceCatalogClusterID != nil {
		out.ServiceCatalogClusterID = *in.ServiceCatalogClusterID
	}
	if in.GenevaLoggingSector != nil {
		out.GenevaLoggingSector = *in.GenevaLoggingSector
	}
	return
}

func mergeCertificateConfig(in *admin.CertificateConfig, out *CertificateConfig) {
	if in.EtcdCa != nil {
		mergeCertKeyPair(in.EtcdCa, &out.EtcdCa)
	}
	if in.Ca != nil {
		mergeCertKeyPair(in.Ca, &out.Ca)
	}
	if in.FrontProxyCa != nil {
		mergeCertKeyPair(in.FrontProxyCa, &out.FrontProxyCa)
	}
	if in.ServiceSigningCa != nil {
		mergeCertKeyPair(in.ServiceSigningCa, &out.ServiceSigningCa)
	}
	if in.ServiceCatalogCa != nil {
		mergeCertKeyPair(in.ServiceCatalogCa, &out.ServiceCatalogCa)
	}
	if in.EtcdServer != nil {
		mergeCertKeyPair(in.EtcdServer, &out.EtcdServer)
	}
	if in.EtcdPeer != nil {
		mergeCertKeyPair(in.EtcdPeer, &out.EtcdPeer)
	}
	if in.EtcdClient != nil {
		mergeCertKeyPair(in.EtcdClient, &out.EtcdClient)
	}
	if in.MasterServer != nil {
		mergeCertKeyPair(in.MasterServer, &out.MasterServer)
	}
	if in.OpenshiftConsole != nil {
		mergeCertKeyPair(in.OpenshiftConsole, &out.OpenshiftConsole)
	}
	if in.Admin != nil {
		mergeCertKeyPair(in.Admin, &out.Admin)
	}
	if in.AggregatorFrontProxy != nil {
		mergeCertKeyPair(in.AggregatorFrontProxy, &out.AggregatorFrontProxy)
	}
	if in.MasterKubeletClient != nil {
		mergeCertKeyPair(in.MasterKubeletClient, &out.MasterKubeletClient)
	}
	if in.MasterProxyClient != nil {
		mergeCertKeyPair(in.MasterProxyClient, &out.MasterProxyClient)
	}
	if in.OpenShiftMaster != nil {
		mergeCertKeyPair(in.OpenShiftMaster, &out.OpenShiftMaster)
	}
	if in.NodeBootstrap != nil {
		mergeCertKeyPair(in.NodeBootstrap, &out.NodeBootstrap)
	}
	if in.Registry != nil {
		mergeCertKeyPair(in.Registry, &out.Registry)
	}
	if in.Router != nil {
		mergeCertKeyPair(in.Router, &out.Router)
	}
	if in.ServiceCatalogServer != nil {
		mergeCertKeyPair(in.ServiceCatalogServer, &out.ServiceCatalogServer)
	}
	if in.ServiceCatalogAPIClient != nil {
		mergeCertKeyPair(in.ServiceCatalogAPIClient, &out.ServiceCatalogAPIClient)
	}
	if in.AzureClusterReader != nil {
		mergeCertKeyPair(in.AzureClusterReader, &out.AzureClusterReader)
	}
	if in.GenevaLogging != nil {
		mergeCertKeyPair(in.GenevaLogging, &out.GenevaLogging)
	}
	return
}

func mergeCertKeyPair(in *admin.CertKeyPair, out *CertKeyPair) {
	if in.Key != nil {
		out.Key = in.Key
	}
	if in.Cert != nil {
		out.Cert = in.Cert
	}
	return
}

func mergeImageConfig(in *admin.ImageConfig, out *ImageConfig) {
	if in.Format != nil {
		out.Format = *in.Format
	}
	if in.ClusterMonitoringOperator != nil {
		out.ClusterMonitoringOperator = *in.ClusterMonitoringOperator
	}
	if in.AzureControllers != nil {
		out.AzureControllers = *in.AzureControllers
	}
	if in.PrometheusOperatorBase != nil {
		out.PrometheusOperatorBase = *in.PrometheusOperatorBase
	}
	if in.PrometheusBase != nil {
		out.PrometheusBase = *in.PrometheusBase
	}
	if in.PrometheusConfigReloaderBase != nil {
		out.PrometheusConfigReloaderBase = *in.PrometheusConfigReloaderBase
	}
	if in.ConfigReloaderBase != nil {
		out.ConfigReloaderBase = *in.ConfigReloaderBase
	}
	if in.AlertManagerBase != nil {
		out.AlertManagerBase = *in.AlertManagerBase
	}
	if in.NodeExporterBase != nil {
		out.NodeExporterBase = *in.NodeExporterBase
	}
	if in.GrafanaBase != nil {
		out.GrafanaBase = *in.GrafanaBase
	}
	if in.KubeStateMetricsBase != nil {
		out.KubeStateMetricsBase = *in.KubeStateMetricsBase
	}
	if in.KubeRbacProxyBase != nil {
		out.KubeRbacProxyBase = *in.KubeRbacProxyBase
	}
	if in.OAuthProxyBase != nil {
		out.OAuthProxyBase = *in.OAuthProxyBase
	}
	if in.MasterEtcd != nil {
		out.MasterEtcd = *in.MasterEtcd
	}
	if in.ControlPlane != nil {
		out.ControlPlane = *in.ControlPlane
	}
	if in.Node != nil {
		out.Node = *in.Node
	}
	if in.ServiceCatalog != nil {
		out.ServiceCatalog = *in.ServiceCatalog
	}
	if in.Sync != nil {
		out.Sync = *in.Sync
	}
	if in.TemplateServiceBroker != nil {
		out.TemplateServiceBroker = *in.TemplateServiceBroker
	}
	if in.Registry != nil {
		out.Registry = *in.Registry
	}
	if in.Router != nil {
		out.Router = *in.Router
	}
	if in.RegistryConsole != nil {
		out.RegistryConsole = *in.RegistryConsole
	}
	if in.AnsibleServiceBroker != nil {
		out.AnsibleServiceBroker = *in.AnsibleServiceBroker
	}
	if in.WebConsole != nil {
		out.WebConsole = *in.WebConsole
	}
	if in.Console != nil {
		out.Console = *in.Console
	}
	if in.EtcdBackup != nil {
		out.EtcdBackup = *in.EtcdBackup
	}
	if len(in.GenevaImagePullSecret) > 0 {
		out.GenevaImagePullSecret = in.GenevaImagePullSecret
	}
	if in.GenevaLogging != nil {
		out.GenevaLogging = *in.GenevaLogging
	}
	if in.GenevaTDAgent != nil {
		out.GenevaTDAgent = *in.GenevaTDAgent
	}
	return
}
