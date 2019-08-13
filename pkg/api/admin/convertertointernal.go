package admin

import (
	"errors"

	"github.com/openshift/openshift-azure/pkg/api"
)

// ToInternal converts from a
// admin.OpenShiftManagedCluster to an internal.OpenShiftManagedCluster.
// If old is non-nil, it is going to be used as the base for the internal
// output where the external request is merged on top of.
func ToInternal(oc *OpenShiftManagedCluster, old *api.OpenShiftManagedCluster) (*api.OpenShiftManagedCluster, error) {
	cs := &api.OpenShiftManagedCluster{}
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
	if cs.Tags == nil && len(oc.Tags) > 0 {
		cs.Tags = make(map[string]string, len(oc.Tags))
	}
	for k, v := range oc.Tags {
		if v != nil {
			cs.Tags[k] = *v
		}
	}

	if oc.Plan != nil {
		mergeFromResourcePurchasePlanAdmin(oc, cs)
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

// mergeFromResourcePurchasePlanAdmin merges filled out fields from the admin API to the internal representation, doesn't change fields which are nil in the input.
// This reflects the behaviour of the external API.
func mergeFromResourcePurchasePlanAdmin(oc *OpenShiftManagedCluster, cs *api.OpenShiftManagedCluster) {
	if cs.Plan == nil {
		cs.Plan = &api.ResourcePurchasePlan{}
	}
	if oc.Plan.Name != nil {
		cs.Plan.Name = oc.Plan.Name
	}
	if oc.Plan.Product != nil {
		cs.Plan.Product = oc.Plan.Product
	}
	if oc.Plan.PromotionCode != nil {
		cs.Plan.PromotionCode = oc.Plan.PromotionCode
	}
	if oc.Plan.Publisher != nil {
		cs.Plan.Publisher = oc.Plan.Publisher
	}
}

func mergePropertiesAdmin(oc *OpenShiftManagedCluster, cs *api.OpenShiftManagedCluster) error {
	if oc.Properties.ProvisioningState != nil {
		cs.Properties.ProvisioningState = api.ProvisioningState(*oc.Properties.ProvisioningState)
	}
	if oc.Properties.OpenShiftVersion != nil {
		cs.Properties.OpenShiftVersion = *oc.Properties.OpenShiftVersion
	}
	if oc.Properties.ClusterVersion != nil {
		cs.Properties.ClusterVersion = *oc.Properties.ClusterVersion
	}
	if oc.Properties.PublicHostname != nil {
		cs.Properties.PublicHostname = *oc.Properties.PublicHostname
	}
	if oc.Properties.FQDN != nil {
		cs.Properties.FQDN = *oc.Properties.FQDN
	}

	if oc.Properties.NetworkProfile != nil {
		if oc.Properties.NetworkProfile.VnetID != nil {
			cs.Properties.NetworkProfile.VnetID = *oc.Properties.NetworkProfile.VnetID
		}
		if oc.Properties.NetworkProfile.VnetCIDR != nil {
			cs.Properties.NetworkProfile.VnetCIDR = *oc.Properties.NetworkProfile.VnetCIDR
		}
		cs.Properties.NetworkProfile.PeerVnetID = oc.Properties.NetworkProfile.PeerVnetID
	}

	if oc.Properties.MonitorProfile != nil {
		if oc.Properties.MonitorProfile.WorkspaceResourceID != nil {
			cs.Properties.MonitorProfile.WorkspaceResourceID = *oc.Properties.MonitorProfile.WorkspaceResourceID
		}
	}

	if err := mergeRouterProfilesAdmin(oc, cs); err != nil {
		return err
	}

	if err := mergeAgentPoolProfiles(oc, cs); err != nil {
		return err
	}

	if err := mergeAuthProfile(oc, cs); err != nil {
		return err
	}

	return nil
}

func mergeRouterProfilesAdmin(oc *OpenShiftManagedCluster, cs *api.OpenShiftManagedCluster) error {
	if cs.Properties.RouterProfiles == nil && len(oc.Properties.RouterProfiles) > 0 {
		cs.Properties.RouterProfiles = make([]api.RouterProfile, 0, len(oc.Properties.RouterProfiles))
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
			cs.Properties.RouterProfiles = append(cs.Properties.RouterProfiles, convertRouterProfile(rp, nil))
		} else {
			head := append(cs.Properties.RouterProfiles[:index], convertRouterProfile(rp, &cs.Properties.RouterProfiles[index]))
			cs.Properties.RouterProfiles = append(head, cs.Properties.RouterProfiles[index+1:]...)
		}
	}
	return nil
}

func routerProfileIndex(name string, profiles []api.RouterProfile) int {
	for i, profile := range profiles {
		if profile.Name == name {
			return i
		}
	}
	return -1
}

func convertRouterProfile(in RouterProfile, old *api.RouterProfile) (out api.RouterProfile) {
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

func mergeAgentPoolProfiles(oc *OpenShiftManagedCluster, cs *api.OpenShiftManagedCluster) error {
	if cs.Properties.AgentPoolProfiles == nil && len(oc.Properties.AgentPoolProfiles) > 0 {
		cs.Properties.AgentPoolProfiles = make([]api.AgentPoolProfile, 0, len(oc.Properties.AgentPoolProfiles)+1)
	}

	if p := oc.Properties.MasterPoolProfile; p != nil {
		index := agentPoolProfileIndex(string(api.AgentPoolProfileRoleMaster), cs.Properties.AgentPoolProfiles)
		// the master profile does not exist, add it as is
		if index == -1 {
			cs.Properties.AgentPoolProfiles = append(cs.Properties.AgentPoolProfiles, convertMasterPoolProfileAdmin(*p, nil))
		} else {
			head := append(cs.Properties.AgentPoolProfiles[:index], convertMasterPoolProfileAdmin(*p, &cs.Properties.AgentPoolProfiles[index]))
			cs.Properties.AgentPoolProfiles = append(head, cs.Properties.AgentPoolProfiles[index+1:]...)
		}
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

func agentPoolProfileIndex(name string, profiles []api.AgentPoolProfile) int {
	for i, profile := range profiles {
		if profile.Name == name {
			return i
		}
	}
	return -1
}

func convertMasterPoolProfileAdmin(in MasterPoolProfile, old *api.AgentPoolProfile) (out api.AgentPoolProfile) {
	if old != nil {
		out = *old
	}
	out.Name = string(api.AgentPoolProfileRoleMaster)
	out.Role = api.AgentPoolProfileRoleMaster
	out.OSType = api.OSTypeLinux
	if in.Count != nil {
		out.Count = *in.Count
	}
	if in.VMSize != nil {
		out.VMSize = api.VMSize(*in.VMSize)
	}
	if in.SubnetCIDR != nil {
		out.SubnetCIDR = *in.SubnetCIDR
	}
	return
}

func convertAgentPoolProfileAdmin(in AgentPoolProfile, old *api.AgentPoolProfile) (out api.AgentPoolProfile) {
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
		out.VMSize = api.VMSize(*in.VMSize)
	}
	if in.SubnetCIDR != nil {
		out.SubnetCIDR = *in.SubnetCIDR
	}
	if in.OSType != nil {
		out.OSType = api.OSType(*in.OSType)
	}
	if in.Role != nil {
		out.Role = api.AgentPoolProfileRole(*in.Role)
	}
	return
}

func mergeAuthProfile(oc *OpenShiftManagedCluster, cs *api.OpenShiftManagedCluster) error {
	if oc.Properties.AuthProfile == nil {
		return nil
	}

	if cs.Properties.AuthProfile.IdentityProviders == nil && len(oc.Properties.AuthProfile.IdentityProviders) > 0 {
		cs.Properties.AuthProfile.IdentityProviders = make([]api.IdentityProvider, 0, len(oc.Properties.AuthProfile.IdentityProviders))
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
			case (*api.AADIdentityProvider):
				in := ip.Provider.(*AADIdentityProvider)
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

func identityProviderIndex(name string, providers []api.IdentityProvider) int {
	for i, provider := range providers {
		if provider.Name == name {
			return i
		}
	}
	return -1
}

func convertIdentityProviderAdmin(in IdentityProvider, old *api.IdentityProvider) (out api.IdentityProvider) {
	if old != nil {
		out = *old
	}
	if in.Name != nil {
		out.Name = *in.Name
	}
	if in.Provider != nil {
		switch provider := in.Provider.(type) {
		case *AADIdentityProvider:
			p := &api.AADIdentityProvider{}
			if out.Provider != nil {
				p = out.Provider.(*api.AADIdentityProvider)
			}
			if provider.Kind != nil {
				p.Kind = *provider.Kind
			}
			if provider.ClientID != nil {
				p.ClientID = *provider.ClientID
			}
			if provider.TenantID != nil {
				p.TenantID = *provider.TenantID
			}
			p.CustomerAdminGroupID = provider.CustomerAdminGroupID
			out.Provider = p

		default:
			panic("authProfile.identityProviders conversion failed")
		}
	}
	return
}

func mergeConfig(oc *OpenShiftManagedCluster, cs *api.OpenShiftManagedCluster) {
	in, out := oc.Config, &cs.Config

	if in.SecurityPatchPackages != nil {
		out.SecurityPatchPackages = *in.SecurityPatchPackages
	}
	if in.PluginVersion != nil {
		out.PluginVersion = *in.PluginVersion
	}
	if in.ComponentLogLevel != nil {
		mergeComponentLogLevel(in.ComponentLogLevel, &out.ComponentLogLevel)
	}
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
	if in.SSHSourceAddressPrefixes != nil {
		out.SSHSourceAddressPrefixes = *in.SSHSourceAddressPrefixes
	}
	if in.ConfigStorageAccount != nil {
		out.ConfigStorageAccount = *in.ConfigStorageAccount
	}
	if in.RegistryStorageAccount != nil {
		out.RegistryStorageAccount = *in.RegistryStorageAccount
	}
	if in.AzureFileStorageAccount != nil {
		out.AzureFileStorageAccount = *in.AzureFileStorageAccount
	}
	if in.Certificates != nil {
		mergeCertificateConfig(in.Certificates, &out.Certificates)
	}
	if in.Images != nil {
		mergeImageConfig(in.Images, &out.Images)
	}
	if in.ServiceCatalogClusterID != nil {
		out.ServiceCatalogClusterID = *in.ServiceCatalogClusterID
	}
	if in.GenevaLoggingSector != nil {
		out.GenevaLoggingSector = *in.GenevaLoggingSector
	}
	if in.GenevaLoggingNamespace != nil {
		out.GenevaLoggingNamespace = *in.GenevaLoggingNamespace
	}
	if in.GenevaLoggingAccount != nil {
		out.GenevaLoggingAccount = *in.GenevaLoggingAccount
	}
	if in.GenevaLoggingControlPlaneEnvironment != nil {
		out.GenevaLoggingControlPlaneEnvironment = *in.GenevaLoggingControlPlaneEnvironment
	}
	if in.GenevaLoggingControlPlaneRegion != nil {
		out.GenevaLoggingControlPlaneRegion = *in.GenevaLoggingControlPlaneRegion
	}
	if in.GenevaMetricsAccount != nil {
		out.GenevaMetricsAccount = *in.GenevaMetricsAccount
	}
	if in.GenevaMetricsEndpoint != nil {
		out.GenevaMetricsEndpoint = *in.GenevaMetricsEndpoint
	}
	if in.GenevaLoggingControlPlaneAccount != nil {
		out.GenevaLoggingControlPlaneAccount = *in.GenevaLoggingControlPlaneAccount
	}
	return
}

func mergeComponentLogLevel(in *ComponentLogLevel, out *api.ComponentLogLevel) {
	if in.APIServer != nil {
		out.APIServer = in.APIServer
	}
	if in.ControllerManager != nil {
		out.ControllerManager = in.ControllerManager
	}
	if in.Node != nil {
		out.Node = in.Node
	}
}

func mergeCertificateConfig(in *CertificateConfig, out *api.CertificateConfig) {
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
	if in.OpenShiftConsole != nil {
		mergeCertKeyPairChain(in.OpenShiftConsole, &out.OpenShiftConsole)
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
	if in.SDN != nil {
		mergeCertKeyPair(in.SDN, &out.SDN)
	}
	if in.Registry != nil {
		mergeCertKeyPair(in.Registry, &out.Registry)
	}
	if in.RegistryConsole != nil {
		mergeCertKeyPair(in.RegistryConsole, &out.RegistryConsole)
	}
	if in.Router != nil {
		mergeCertKeyPairChain(in.Router, &out.Router)
	}
	if in.ServiceCatalogServer != nil {
		mergeCertKeyPair(in.ServiceCatalogServer, &out.ServiceCatalogServer)
	}
	if in.BlackBoxMonitor != nil {
		mergeCertKeyPair(in.BlackBoxMonitor, &out.BlackBoxMonitor)
	}
	if in.GenevaLogging != nil {
		mergeCertKeyPair(in.GenevaLogging, &out.GenevaLogging)
	}
	if in.GenevaMetrics != nil {
		mergeCertKeyPair(in.GenevaMetrics, &out.GenevaMetrics)
	}
	if in.PackageRepository != nil {
		mergeCertKeyPair(in.PackageRepository, &out.PackageRepository)
	}
	return
}

func mergeCertKeyPair(in *Certificate, out *api.CertKeyPair) {
	if in.Cert != nil {
		out.Cert = in.Cert
	}
	return
}

// TODO: this is not a great semantic - revisit if this field is ever enabled for write
func mergeCertKeyPairChain(in *CertificateChain, out *api.CertKeyPairChain) {
	if in.Certs != nil {
		out.Certs = in.Certs
	}
}

func mergeImageConfig(in *ImageConfig, out *api.ImageConfig) {
	if in.Format != nil {
		out.Format = *in.Format
	}
	if in.ClusterMonitoringOperator != nil {
		out.ClusterMonitoringOperator = *in.ClusterMonitoringOperator
	}
	if in.AzureControllers != nil {
		out.AzureControllers = *in.AzureControllers
	}
	if in.PrometheusOperator != nil {
		out.PrometheusOperator = *in.PrometheusOperator
	}
	if in.Prometheus != nil {
		out.Prometheus = *in.Prometheus
	}
	if in.PrometheusConfigReloader != nil {
		out.PrometheusConfigReloader = *in.PrometheusConfigReloader
	}
	if in.ConfigReloader != nil {
		out.ConfigReloader = *in.ConfigReloader
	}
	if in.AlertManager != nil {
		out.AlertManager = *in.AlertManager
	}
	if in.NodeExporter != nil {
		out.NodeExporter = *in.NodeExporter
	}
	if in.Grafana != nil {
		out.Grafana = *in.Grafana
	}
	if in.KubeStateMetrics != nil {
		out.KubeStateMetrics = *in.KubeStateMetrics
	}
	if in.KubeRbacProxy != nil {
		out.KubeRbacProxy = *in.KubeRbacProxy
	}
	if in.OAuthProxy != nil {
		out.OAuthProxy = *in.OAuthProxy
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
	if in.TLSProxy != nil {
		out.TLSProxy = *in.TLSProxy
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
	if in.Httpd != nil {
		out.Httpd = *in.Httpd
	}
	if in.Canary != nil {
		out.Canary = *in.Canary
	}
	if in.Startup != nil {
		out.Startup = *in.Startup
	}
	if in.GenevaLogging != nil {
		out.GenevaLogging = *in.GenevaLogging
	}
	if in.GenevaTDAgent != nil {
		out.GenevaTDAgent = *in.GenevaTDAgent
	}
	if in.GenevaStatsd != nil {
		out.GenevaStatsd = *in.GenevaStatsd
	}
	if in.MetricsBridge != nil {
		out.MetricsBridge = *in.MetricsBridge
	}
	if in.LogAnalyticsAgent != nil {
		out.LogAnalyticsAgent = *in.LogAnalyticsAgent
	}
	return
}
