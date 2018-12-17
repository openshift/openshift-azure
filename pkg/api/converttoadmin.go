package api

import (
	admin "github.com/openshift/openshift-azure/pkg/api/admin/api"
)

func nilIfAdminProvisioningStateEmpty(s *admin.ProvisioningState) *admin.ProvisioningState {
	if s == nil || len(*s) == 0 {
		return nil
	}
	return s
}

func nilIfAdminAgentPoolProfileRoleEmpty(s *admin.AgentPoolProfileRole) *admin.AgentPoolProfileRole {
	if s == nil || len(*s) == 0 {
		return nil
	}
	return s
}

func nilIfAdminVMSizeEmpty(s *admin.VMSize) *admin.VMSize {
	if s == nil || len(*s) == 0 {
		return nil
	}
	return s
}

func nilIfAdminOSTypeEmpty(s *admin.OSType) *admin.OSType {
	if s == nil || len(*s) == 0 {
		return nil
	}
	return s
}

func ConvertToAdmin(cs *OpenShiftManagedCluster) *admin.OpenShiftManagedCluster {
	oc := &admin.OpenShiftManagedCluster{
		ID:       nilIfStringEmpty(&cs.ID),
		Location: nilIfStringEmpty(&cs.Location),
		Name:     nilIfStringEmpty(&cs.Name),
		Type:     nilIfStringEmpty(&cs.Type),
	}
	oc.Tags = make(map[string]*string, len(cs.Tags))
	for k := range cs.Tags {
		v := cs.Tags[k]
		oc.Tags[k] = nilIfStringEmpty(&v)
	}

	oc.Plan = &admin.ResourcePurchasePlan{
		Name:          nilIfStringEmpty(&cs.Plan.Name),
		Product:       nilIfStringEmpty(&cs.Plan.Product),
		PromotionCode: nilIfStringEmpty(&cs.Plan.PromotionCode),
		Publisher:     nilIfStringEmpty(&cs.Plan.Publisher),
	}

	provisioningState := admin.ProvisioningState(cs.Properties.ProvisioningState)
	oc.Properties = &admin.Properties{
		ProvisioningState: nilIfAdminProvisioningStateEmpty(&provisioningState),
		OpenShiftVersion:  nilIfStringEmpty(&cs.Properties.OpenShiftVersion),
		PublicHostname:    nilIfStringEmpty(&cs.Properties.PublicHostname),
		FQDN:              nilIfStringEmpty(&cs.Properties.FQDN),
	}

	oc.Properties.NetworkProfile = &admin.NetworkProfile{
		VnetCIDR:   nilIfStringEmpty(&cs.Properties.NetworkProfile.VnetCIDR),
		PeerVnetID: nilIfStringEmpty(&cs.Properties.NetworkProfile.PeerVnetID),
	}

	oc.Properties.RouterProfiles = make([]admin.RouterProfile, len(cs.Properties.RouterProfiles))
	for i := range cs.Properties.RouterProfiles {
		rp := cs.Properties.RouterProfiles[i]
		oc.Properties.RouterProfiles[i] = admin.RouterProfile{
			Name:            nilIfStringEmpty(&rp.Name),
			PublicSubdomain: nilIfStringEmpty(&rp.PublicSubdomain),
			FQDN:            nilIfStringEmpty(&rp.FQDN),
		}
	}

	oc.Properties.AgentPoolProfiles = make([]admin.AgentPoolProfile, 0, len(cs.Properties.AgentPoolProfiles))
	for i := range cs.Properties.AgentPoolProfiles {
		app := cs.Properties.AgentPoolProfiles[i]
		vmSize := admin.VMSize(app.VMSize)
		osType := admin.OSType(app.OSType)
		role := admin.AgentPoolProfileRole(app.Role)

		oc.Properties.AgentPoolProfiles = append(oc.Properties.AgentPoolProfiles, admin.AgentPoolProfile{
			Name:       nilIfStringEmpty(&app.Name),
			Count:      &app.Count,
			VMSize:     nilIfAdminVMSizeEmpty(&vmSize),
			SubnetCIDR: nilIfStringEmpty(&app.SubnetCIDR),
			OSType:     nilIfAdminOSTypeEmpty(&osType),
			Role:       nilIfAdminAgentPoolProfileRoleEmpty(&role),
		})
	}

	oc.Properties.AuthProfile = &admin.AuthProfile{}
	oc.Properties.AuthProfile.IdentityProviders = make([]admin.IdentityProvider, len(cs.Properties.AuthProfile.IdentityProviders))
	for i := range cs.Properties.AuthProfile.IdentityProviders {
		ip := cs.Properties.AuthProfile.IdentityProviders[i]
		oc.Properties.AuthProfile.IdentityProviders[i].Name = &ip.Name
		switch provider := ip.Provider.(type) {
		case *AADIdentityProvider:
			oc.Properties.AuthProfile.IdentityProviders[i].Provider = &admin.AADIdentityProvider{
				Kind:     nilIfStringEmpty(&provider.Kind),
				ClientID: nilIfStringEmpty(&provider.ClientID),
				TenantID: nilIfStringEmpty(&provider.TenantID),
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
		ImageOffer:                       nilIfStringEmpty(&cs.ImageOffer),
		ImagePublisher:                   nilIfStringEmpty(&cs.ImagePublisher),
		ImageSKU:                         nilIfStringEmpty(&cs.ImageSKU),
		ImageVersion:                     nilIfStringEmpty(&cs.ImageVersion),
		ConfigStorageAccount:             nilIfStringEmpty(&cs.ConfigStorageAccount),
		RegistryStorageAccount:           nilIfStringEmpty(&cs.RegistryStorageAccount),
		Certificates:                     convertCertificateConfigToAdmin(cs.Certificates),
		Images:                           convertImageConfigToAdmin(cs.Images),
		ServiceCatalogClusterID:          &cs.ServiceCatalogClusterID,
		GenevaLoggingSector:              nilIfStringEmpty(&cs.GenevaLoggingSector),
		GenevaLoggingAccount:             nilIfStringEmpty(&cs.GenevaLoggingAccount),
		GenevaLoggingNamespace:           nilIfStringEmpty(&cs.GenevaLoggingNamespace),
		GenevaLoggingControlPlaneAccount: nilIfStringEmpty(&cs.GenevaLoggingControlPlaneAccount),
		GenevaMetricsAccount:             nilIfStringEmpty(&cs.GenevaMetricsAccount),
		GenevaMetricsEndpoint:            nilIfStringEmpty(&cs.GenevaMetricsEndpoint),
	}
}

func convertCertificateConfigToAdmin(in CertificateConfig) *admin.CertificateConfig {
	return &admin.CertificateConfig{
		EtcdCa:                  convertCertKeyPairToAdmin(in.EtcdCa),
		Ca:                      convertCertKeyPairToAdmin(in.Ca),
		FrontProxyCa:            convertCertKeyPairToAdmin(in.FrontProxyCa),
		ServiceSigningCa:        convertCertKeyPairToAdmin(in.ServiceSigningCa),
		ServiceCatalogCa:        convertCertKeyPairToAdmin(in.ServiceCatalogCa),
		EtcdServer:              convertCertKeyPairToAdmin(in.EtcdServer),
		EtcdPeer:                convertCertKeyPairToAdmin(in.EtcdPeer),
		EtcdClient:              convertCertKeyPairToAdmin(in.EtcdClient),
		MasterServer:            convertCertKeyPairToAdmin(in.MasterServer),
		OpenshiftConsole:        convertCertKeyPairToAdmin(in.OpenshiftConsole),
		Admin:                   convertCertKeyPairToAdmin(in.Admin),
		AggregatorFrontProxy:    convertCertKeyPairToAdmin(in.AggregatorFrontProxy),
		MasterKubeletClient:     convertCertKeyPairToAdmin(in.MasterKubeletClient),
		MasterProxyClient:       convertCertKeyPairToAdmin(in.MasterProxyClient),
		OpenShiftMaster:         convertCertKeyPairToAdmin(in.OpenShiftMaster),
		NodeBootstrap:           convertCertKeyPairToAdmin(in.NodeBootstrap),
		Registry:                convertCertKeyPairToAdmin(in.Registry),
		Router:                  convertCertKeyPairToAdmin(in.Router),
		ServiceCatalogServer:    convertCertKeyPairToAdmin(in.ServiceCatalogServer),
		ServiceCatalogAPIClient: convertCertKeyPairToAdmin(in.ServiceCatalogAPIClient),
		AzureClusterReader:      convertCertKeyPairToAdmin(in.AzureClusterReader),
		GenevaLogging:           convertCertKeyPairToAdmin(in.GenevaLogging),
		GenevaMetrics:           convertCertKeyPairToAdmin(in.GenevaMetrics),
	}
}

func convertCertKeyPairToAdmin(in CertKeyPair) *admin.Certificate {
	return &admin.Certificate{
		Cert: in.Cert,
	}
}

func convertImageConfigToAdmin(in ImageConfig) *admin.ImageConfig {
	return &admin.ImageConfig{
		Format:                       nilIfStringEmpty(&in.Format),
		ClusterMonitoringOperator:    nilIfStringEmpty(&in.ClusterMonitoringOperator),
		AzureControllers:             nilIfStringEmpty(&in.AzureControllers),
		PrometheusOperatorBase:       nilIfStringEmpty(&in.PrometheusOperatorBase),
		PrometheusBase:               nilIfStringEmpty(&in.PrometheusBase),
		PrometheusConfigReloaderBase: nilIfStringEmpty(&in.PrometheusConfigReloaderBase),
		ConfigReloaderBase:           nilIfStringEmpty(&in.ConfigReloaderBase),
		AlertManagerBase:             nilIfStringEmpty(&in.AlertManagerBase),
		NodeExporterBase:             nilIfStringEmpty(&in.NodeExporterBase),
		GrafanaBase:                  nilIfStringEmpty(&in.GrafanaBase),
		KubeStateMetricsBase:         nilIfStringEmpty(&in.KubeStateMetricsBase),
		KubeRbacProxyBase:            nilIfStringEmpty(&in.KubeRbacProxyBase),
		OAuthProxyBase:               nilIfStringEmpty(&in.OAuthProxyBase),
		MasterEtcd:                   nilIfStringEmpty(&in.MasterEtcd),
		ControlPlane:                 nilIfStringEmpty(&in.ControlPlane),
		Node:                         nilIfStringEmpty(&in.Node),
		ServiceCatalog:               nilIfStringEmpty(&in.ServiceCatalog),
		Sync:                         nilIfStringEmpty(&in.Sync),
		TemplateServiceBroker:        nilIfStringEmpty(&in.TemplateServiceBroker),
		Registry:                     nilIfStringEmpty(&in.Registry),
		Router:                       nilIfStringEmpty(&in.Router),
		RegistryConsole:              nilIfStringEmpty(&in.RegistryConsole),
		AnsibleServiceBroker:         nilIfStringEmpty(&in.AnsibleServiceBroker),
		WebConsole:                   nilIfStringEmpty(&in.WebConsole),
		Console:                      nilIfStringEmpty(&in.Console),
		EtcdBackup:                   nilIfStringEmpty(&in.EtcdBackup),
		GenevaLogging:                nilIfStringEmpty(&in.GenevaLogging),
		GenevaTDAgent:                nilIfStringEmpty(&in.GenevaTDAgent),
		GenevaStatsd:                 nilIfStringEmpty(&in.GenevaStatsd),
		MetricsBridge:                nilIfStringEmpty(&in.MetricsBridge),
	}
}
