package api

import (
	admin "github.com/openshift/openshift-azure/pkg/api/admin/api"
)

// ConvertToAdmin converts the config representation for the admin API
func ConvertToAdmin(cs *OpenShiftManagedCluster) *admin.OpenShiftManagedCluster {
	oc := &admin.OpenShiftManagedCluster{
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
		oc.Plan = &admin.ResourcePurchasePlan{
			Name:          cs.Plan.Name,
			Product:       cs.Plan.Product,
			PromotionCode: cs.Plan.PromotionCode,
			Publisher:     cs.Plan.Publisher,
		}
	}

	provisioningState := admin.ProvisioningState(cs.Properties.ProvisioningState)
	oc.Properties = &admin.Properties{
		ProvisioningState: &provisioningState,
		OpenShiftVersion:  &cs.Properties.OpenShiftVersion,
		ClusterVersion:    &cs.Properties.ClusterVersion,
		PublicHostname:    &cs.Properties.PublicHostname,
		FQDN:              &cs.Properties.FQDN,
	}

	oc.Properties.NetworkProfile = &admin.NetworkProfile{
		VnetID:     &cs.Properties.NetworkProfile.VnetID,
		VnetCIDR:   &cs.Properties.NetworkProfile.VnetCIDR,
		PeerVnetID: cs.Properties.NetworkProfile.PeerVnetID,
	}

	oc.Properties.RouterProfiles = make([]admin.RouterProfile, len(cs.Properties.RouterProfiles))
	for i := range cs.Properties.RouterProfiles {
		rp := cs.Properties.RouterProfiles[i]
		oc.Properties.RouterProfiles[i] = admin.RouterProfile{
			Name:            &rp.Name,
			PublicSubdomain: &rp.PublicSubdomain,
			FQDN:            &rp.FQDN,
		}
	}

	oc.Properties.AgentPoolProfiles = make([]admin.AgentPoolProfile, 0, len(cs.Properties.AgentPoolProfiles))
	for i := range cs.Properties.AgentPoolProfiles {
		app := cs.Properties.AgentPoolProfiles[i]
		vmSize := admin.VMSize(app.VMSize)

		if app.Role == AgentPoolProfileRoleMaster {
			oc.Properties.MasterPoolProfile = &admin.MasterPoolProfile{
				Count:      &app.Count,
				VMSize:     &vmSize,
				SubnetCIDR: &app.SubnetCIDR,
			}
		} else {
			osType := admin.OSType(app.OSType)
			role := admin.AgentPoolProfileRole(app.Role)

			oc.Properties.AgentPoolProfiles = append(oc.Properties.AgentPoolProfiles, admin.AgentPoolProfile{
				Name:       &app.Name,
				Count:      &app.Count,
				VMSize:     &vmSize,
				SubnetCIDR: &app.SubnetCIDR,
				OSType:     &osType,
				Role:       &role,
			})
		}
	}

	oc.Properties.AuthProfile = &admin.AuthProfile{}
	oc.Properties.AuthProfile.IdentityProviders = make([]admin.IdentityProvider, len(cs.Properties.AuthProfile.IdentityProviders))
	for i := range cs.Properties.AuthProfile.IdentityProviders {
		ip := cs.Properties.AuthProfile.IdentityProviders[i]
		oc.Properties.AuthProfile.IdentityProviders[i].Name = &ip.Name
		switch provider := ip.Provider.(type) {
		case *AADIdentityProvider:
			oc.Properties.AuthProfile.IdentityProviders[i].Provider = &admin.AADIdentityProvider{
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

func convertConfigToAdmin(cs *Config) *admin.Config {
	return &admin.Config{
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

func convertComponentLogLevelToAdmin(in ComponentLogLevel) *admin.ComponentLogLevel {
	return &admin.ComponentLogLevel{
		APIServer:         &in.APIServer,
		ControllerManager: &in.ControllerManager,
		Node:              &in.Node,
	}
}

func convertCertificateConfigToAdmin(in CertificateConfig) *admin.CertificateConfig {
	return &admin.CertificateConfig{
		EtcdCa:               convertCertKeyPairToAdmin(in.EtcdCa),
		Ca:                   convertCertKeyPairToAdmin(in.Ca),
		FrontProxyCa:         convertCertKeyPairToAdmin(in.FrontProxyCa),
		ServiceSigningCa:     convertCertKeyPairToAdmin(in.ServiceSigningCa),
		ServiceCatalogCa:     convertCertKeyPairToAdmin(in.ServiceCatalogCa),
		EtcdServer:           convertCertKeyPairToAdmin(in.EtcdServer),
		EtcdPeer:             convertCertKeyPairToAdmin(in.EtcdPeer),
		EtcdClient:           convertCertKeyPairToAdmin(in.EtcdClient),
		MasterServer:         convertCertKeyPairToAdmin(in.MasterServer),
		OpenShiftConsole:     convertCertKeyPairToAdmin(in.OpenShiftConsole),
		Admin:                convertCertKeyPairToAdmin(in.Admin),
		AggregatorFrontProxy: convertCertKeyPairToAdmin(in.AggregatorFrontProxy),
		MasterKubeletClient:  convertCertKeyPairToAdmin(in.MasterKubeletClient),
		MasterProxyClient:    convertCertKeyPairToAdmin(in.MasterProxyClient),
		OpenShiftMaster:      convertCertKeyPairToAdmin(in.OpenShiftMaster),
		NodeBootstrap:        convertCertKeyPairToAdmin(in.NodeBootstrap),
		Registry:             convertCertKeyPairToAdmin(in.Registry),
		RegistryConsole:      convertCertKeyPairToAdmin(in.RegistryConsole),
		Router:               convertCertKeyPairToAdmin(in.Router),
		ServiceCatalogServer: convertCertKeyPairToAdmin(in.ServiceCatalogServer),
		BlackBoxMonitor:      convertCertKeyPairToAdmin(in.BlackBoxMonitor),
		GenevaLogging:        convertCertKeyPairToAdmin(in.GenevaLogging),
		GenevaMetrics:        convertCertKeyPairToAdmin(in.GenevaMetrics),
	}
}

func convertCertKeyPairToAdmin(in CertKeyPair) *admin.Certificate {
	return &admin.Certificate{
		Cert: in.Cert,
	}
}

func convertImageConfigToAdmin(in ImageConfig) *admin.ImageConfig {
	return &admin.ImageConfig{
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
		Registry:                  &in.Registry,
		Router:                    &in.Router,
		RegistryConsole:           &in.RegistryConsole,
		AnsibleServiceBroker:      &in.AnsibleServiceBroker,
		WebConsole:                &in.WebConsole,
		Console:                   &in.Console,
		EtcdBackup:                &in.EtcdBackup,
		Httpd:                     &in.Httpd,
		GenevaLogging:             &in.GenevaLogging,
		GenevaTDAgent:             &in.GenevaTDAgent,
		GenevaStatsd:              &in.GenevaStatsd,
		MetricsBridge:             &in.MetricsBridge,
	}
}
