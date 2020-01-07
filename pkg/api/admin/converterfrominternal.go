package admin

import (
	"github.com/openshift/openshift-azure/pkg/api"
)

// FromInternal converts from a
// internal.OpenShiftManagedCluster to an admin.OpenShiftManagedCluster.
func FromInternal(cs *api.OpenShiftManagedCluster) *OpenShiftManagedCluster {
	oc := &OpenShiftManagedCluster{
		ID:       &cs.ID,
		Location: &cs.Location,
		Name:     &cs.Name,
		Type:     &cs.Type,
	}
	oc.Tags = make(map[string]*string, len(cs.Tags))
	for k := range cs.Tags {
		v := cs.Tags[k]
		oc.Tags[k] = &v
	}
	if cs.Plan != nil {
		oc.Plan = &ResourcePurchasePlan{
			Name:          cs.Plan.Name,
			Product:       cs.Plan.Product,
			PromotionCode: cs.Plan.PromotionCode,
			Publisher:     cs.Plan.Publisher,
		}
	}

	provisioningState := ProvisioningState(cs.Properties.ProvisioningState)
	oc.Properties = &Properties{
		ProvisioningState: &provisioningState,
		OpenShiftVersion:  &cs.Properties.OpenShiftVersion,
		ClusterVersion:    &cs.Properties.ClusterVersion,
		PublicHostname:    &cs.Properties.PublicHostname,
		FQDN:              &cs.Properties.FQDN,
		PrivateAPIServer:  cs.Properties.PrivateAPIServer,
	}
	// This is intentionally reversed as far as pointers go.
	if cs.Properties.RefreshCluster != nil {
		oc.Properties.RefreshCluster = *cs.Properties.RefreshCluster
	}

	oc.Properties.NetworkProfile = &NetworkProfile{
		VnetID:               &cs.Properties.NetworkProfile.VnetID,
		VnetCIDR:             &cs.Properties.NetworkProfile.VnetCIDR,
		ManagementSubnetCIDR: cs.Properties.NetworkProfile.ManagementSubnetCIDR,
		PeerVnetID:           cs.Properties.NetworkProfile.PeerVnetID,
	}
	oc.Properties.MonitorProfile = &MonitorProfile{
		Enabled:             &cs.Properties.MonitorProfile.Enabled,
		WorkspaceResourceID: &cs.Properties.MonitorProfile.WorkspaceResourceID,
	}

	oc.Properties.RouterProfiles = make([]RouterProfile, len(cs.Properties.RouterProfiles))
	for i := range cs.Properties.RouterProfiles {
		rp := cs.Properties.RouterProfiles[i]
		oc.Properties.RouterProfiles[i] = RouterProfile{
			Name:            &rp.Name,
			PublicSubdomain: &rp.PublicSubdomain,
			FQDN:            &rp.FQDN,
		}
	}

	oc.Properties.AgentPoolProfiles = make([]AgentPoolProfile, 0, len(cs.Properties.AgentPoolProfiles))
	for i := range cs.Properties.AgentPoolProfiles {
		app := cs.Properties.AgentPoolProfiles[i]
		vmSize := VMSize(app.VMSize)

		if app.Role == api.AgentPoolProfileRoleMaster {
			oc.Properties.MasterPoolProfile = &MasterPoolProfile{
				Count:      &app.Count,
				VMSize:     &vmSize,
				SubnetCIDR: &app.SubnetCIDR,
			}
		} else {
			osType := OSType(app.OSType)
			role := AgentPoolProfileRole(app.Role)

			oc.Properties.AgentPoolProfiles = append(oc.Properties.AgentPoolProfiles, AgentPoolProfile{
				Name:       &app.Name,
				Count:      &app.Count,
				VMSize:     &vmSize,
				SubnetCIDR: &app.SubnetCIDR,
				OSType:     &osType,
				Role:       &role,
			})
		}
	}

	oc.Properties.AuthProfile = &AuthProfile{}
	oc.Properties.AuthProfile.IdentityProviders = make([]IdentityProvider, len(cs.Properties.AuthProfile.IdentityProviders))
	for i := range cs.Properties.AuthProfile.IdentityProviders {
		ip := cs.Properties.AuthProfile.IdentityProviders[i]
		oc.Properties.AuthProfile.IdentityProviders[i].Name = &ip.Name
		switch provider := ip.Provider.(type) {
		case *api.AADIdentityProvider:
			oc.Properties.AuthProfile.IdentityProviders[i].Provider = &AADIdentityProvider{
				Kind:                 &provider.Kind,
				ClientID:             &provider.ClientID,
				TenantID:             &provider.TenantID,
				CustomerAdminGroupID: provider.CustomerAdminGroupID,
			}

		default:
			panic("authProfile.identityProviders conversion failed")
		}
	}

	oc.Config = convertConfigToAdmin(&cs.Config)

	return oc
}

func convertConfigToAdmin(cs *api.Config) *Config {
	return &Config{
		SecurityPatchPackages:                &cs.SecurityPatchPackages,
		PluginVersion:                        &cs.PluginVersion,
		ComponentLogLevel:                    convertComponentLogLevelToAdmin(cs.ComponentLogLevel),
		SSHSourceAddressPrefixes:             &cs.SSHSourceAddressPrefixes,
		ImageOffer:                           &cs.ImageOffer,
		ImagePublisher:                       &cs.ImagePublisher,
		ImageSKU:                             &cs.ImageSKU,
		ImageVersion:                         &cs.ImageVersion,
		ConfigStorageAccount:                 &cs.ConfigStorageAccount,
		RegistryStorageAccount:               &cs.RegistryStorageAccount,
		AzureFileStorageAccount:              &cs.AzureFileStorageAccount,
		Certificates:                         convertCertificateConfigToAdmin(cs.Certificates),
		Images:                               convertImageConfigToAdmin(cs.Images),
		ServiceCatalogClusterID:              &cs.ServiceCatalogClusterID,
		GenevaLoggingSector:                  &cs.GenevaLoggingSector,
		GenevaLoggingAccount:                 &cs.GenevaLoggingAccount,
		GenevaLoggingNamespace:               &cs.GenevaLoggingNamespace,
		GenevaLoggingControlPlaneAccount:     &cs.GenevaLoggingControlPlaneAccount,
		GenevaLoggingControlPlaneEnvironment: &cs.GenevaLoggingControlPlaneEnvironment,
		GenevaLoggingControlPlaneRegion:      &cs.GenevaLoggingControlPlaneRegion,
		GenevaMetricsAccount:                 &cs.GenevaMetricsAccount,
		GenevaMetricsEndpoint:                &cs.GenevaMetricsEndpoint,
	}
}

func convertComponentLogLevelToAdmin(in api.ComponentLogLevel) *ComponentLogLevel {
	return &ComponentLogLevel{
		APIServer:         in.APIServer,
		ControllerManager: in.ControllerManager,
		Node:              in.Node,
	}
}

func convertCertificateConfigToAdmin(in api.CertificateConfig) *CertificateConfig {
	return &CertificateConfig{
		EtcdCa:                       convertCertKeyPairToAdmin(in.EtcdCa),
		Ca:                           convertCertKeyPairToAdmin(in.Ca),
		FrontProxyCa:                 convertCertKeyPairToAdmin(in.FrontProxyCa),
		ServiceSigningCa:             convertCertKeyPairToAdmin(in.ServiceSigningCa),
		ServiceCatalogCa:             convertCertKeyPairToAdmin(in.ServiceCatalogCa),
		EtcdServer:                   convertCertKeyPairToAdmin(in.EtcdServer),
		EtcdPeer:                     convertCertKeyPairToAdmin(in.EtcdPeer),
		EtcdClient:                   convertCertKeyPairToAdmin(in.EtcdClient),
		MasterServer:                 convertCertKeyPairToAdmin(in.MasterServer),
		OpenShiftConsole:             convertCertKeyPairChainToAdmin(in.OpenShiftConsole),
		Admin:                        convertCertKeyPairToAdmin(in.Admin),
		AggregatorFrontProxy:         convertCertKeyPairToAdmin(in.AggregatorFrontProxy),
		MasterKubeletClient:          convertCertKeyPairToAdmin(in.MasterKubeletClient),
		MasterProxyClient:            convertCertKeyPairToAdmin(in.MasterProxyClient),
		OpenShiftMaster:              convertCertKeyPairToAdmin(in.OpenShiftMaster),
		NodeBootstrap:                convertCertKeyPairToAdmin(in.NodeBootstrap),
		SDN:                          convertCertKeyPairToAdmin(in.SDN),
		Registry:                     convertCertKeyPairToAdmin(in.Registry),
		RegistryConsole:              convertCertKeyPairToAdmin(in.RegistryConsole),
		Router:                       convertCertKeyPairChainToAdmin(in.Router),
		ServiceCatalogServer:         convertCertKeyPairToAdmin(in.ServiceCatalogServer),
		AroAdmissionController:       convertCertKeyPairToAdmin(in.AroAdmissionController),
		AroAdmissionControllerClient: convertCertKeyPairToAdmin(in.AroAdmissionControllerClient),
		BlackBoxMonitor:              convertCertKeyPairToAdmin(in.BlackBoxMonitor),
		GenevaLogging:                convertCertKeyPairToAdmin(in.GenevaLogging),
		GenevaMetrics:                convertCertKeyPairToAdmin(in.GenevaMetrics),
		PackageRepository:            convertCertKeyPairToAdmin(in.PackageRepository),
		MetricsServer:                convertCertKeyPairToAdmin(in.MetricsServer),
	}
}

func convertCertKeyPairToAdmin(in api.CertKeyPair) *Certificate {
	return &Certificate{
		Cert: in.Cert,
	}
}

func convertCertKeyPairChainToAdmin(in api.CertKeyPairChain) *CertificateChain {
	return &CertificateChain{
		Certs: in.Certs,
	}
}

func convertImageConfigToAdmin(in api.ImageConfig) *ImageConfig {
	return &ImageConfig{
		Format:                    &in.Format,
		ClusterMonitoringOperator: &in.ClusterMonitoringOperator,
		AzureControllers:          &in.AzureControllers,
		PrometheusOperator:        &in.PrometheusOperator,
		Prometheus:                &in.Prometheus,
		PrometheusConfigReloader:  &in.PrometheusConfigReloader,
		ConfigReloader:            &in.ConfigReloader,
		AlertManager:              &in.AlertManager,
		NodeExporter:              &in.NodeExporter,
		Grafana:                   &in.Grafana,
		KubeStateMetrics:          &in.KubeStateMetrics,
		KubeRbacProxy:             &in.KubeRbacProxy,
		OAuthProxy:                &in.OAuthProxy,
		MasterEtcd:                &in.MasterEtcd,
		ControlPlane:              &in.ControlPlane,
		Node:                      &in.Node,
		ServiceCatalog:            &in.ServiceCatalog,
		Sync:                      &in.Sync,
		Startup:                   &in.Startup,
		TemplateServiceBroker:     &in.TemplateServiceBroker,
		TLSProxy:                  &in.TLSProxy,
		Registry:                  &in.Registry,
		Router:                    &in.Router,
		RegistryConsole:           &in.RegistryConsole,
		AnsibleServiceBroker:      &in.AnsibleServiceBroker,
		WebConsole:                &in.WebConsole,
		Console:                   &in.Console,
		EtcdBackup:                &in.EtcdBackup,
		Httpd:                     &in.Httpd,
		Canary:                    &in.Canary,
		GenevaLogging:             &in.GenevaLogging,
		GenevaTDAgent:             &in.GenevaTDAgent,
		GenevaStatsd:              &in.GenevaStatsd,
		MetricsBridge:             &in.MetricsBridge,
		LogAnalyticsAgent:         &in.LogAnalyticsAgent,
		AroAdmissionController:    &in.AroAdmissionController,
		MetricsServer:             &in.MetricsServer,
	}
}
