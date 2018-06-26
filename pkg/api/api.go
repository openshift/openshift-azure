package api

type Manifest struct {
	TenantID               string
	SubscriptionID         string
	ClientID               string
	ClientSecret           string
	Location               string
	ResourceGroup          string
	VMSize                 string
	ComputeCount           int
	InfraCount             int
	ImageResourceGroup     string
	ImageResourceName      string
	RoutingConfigSubdomain string
	PublicHostname         string
}
