package api

// TODO: needs unit tests

// Notes: shortcomings of the Azure-style API / its integration with Golang
// include:
// 1. There is no mechanism to delete an item from a slice or map without
// implementing a separate CRUD endpoint.
// 2. There is no way to set a plain value to its zero value (e.g. bool to
// false, int to 0, string to "") without storing it as a pointer in the API.

// At the moment Config isn't merged.  Perhaps what we actually need is a
// merging converter f(*ext, *int) *int.  Perhaps this would mean we could get
// rid of some pointers in our internal type.

func Merge(complete, partial *OpenShiftManagedCluster) *OpenShiftManagedCluster {
	if complete != nil {
		complete = complete.DeepCopy() // required because Config isn't merged
	}

	return mergeOpenShiftManagedCluster(complete, partial)
}

func mergeOpenShiftManagedCluster(complete, partial *OpenShiftManagedCluster) *OpenShiftManagedCluster {
	switch {
	case complete == nil && partial == nil:
		return nil
	case complete == nil:
		complete = &OpenShiftManagedCluster{}
	case partial == nil:
		partial = &OpenShiftManagedCluster{}
	}

	return &OpenShiftManagedCluster{
		ID:         mergeString(complete.ID, partial.ID),
		Location:   mergeString(complete.Location, partial.Location),
		Name:       mergeString(complete.Name, partial.Name),
		Plan:       mergeResourcePurchasePlan(complete.Plan, partial.Plan),
		Tags:       mergeTags(complete.Tags, partial.Tags),
		Type:       mergeString(complete.Type, partial.Type),
		Properties: mergeProperties(complete.Properties, partial.Properties),
		Config:     complete.Config, // NOTE: Config isn't merged
	}
}

func mergeString(complete, partial string) string {
	if partial == "" {
		return complete
	}
	return partial
}

func mergeResourcePurchasePlan(complete, partial *ResourcePurchasePlan) *ResourcePurchasePlan {
	switch {
	case complete == nil && partial == nil:
		return nil
	case complete == nil:
		complete = &ResourcePurchasePlan{}
	case partial == nil:
		partial = &ResourcePurchasePlan{}
	}

	return &ResourcePurchasePlan{
		Name:          mergeString(complete.Name, partial.Name),
		Product:       mergeString(complete.Product, partial.Product),
		PromotionCode: mergeString(complete.PromotionCode, partial.PromotionCode),
		Publisher:     mergeString(complete.Publisher, partial.Publisher),
	}
}

func mergeTags(complete, partial map[string]string) map[string]string {
	if complete == nil && partial == nil {
		return nil
	}

	// TODO: is this the semantic that we want for mergeTags?

	rv := make(map[string]string, len(complete))
	for k, v := range complete {
		rv[k] = v
	}
	for k, v := range partial {
		rv[k] = v
	}
	return rv
}

func mergeProperties(complete, partial *Properties) *Properties {
	switch {
	case complete == nil && partial == nil:
		return nil
	case complete == nil:
		complete = &Properties{}
	case partial == nil:
		partial = &Properties{}
	}

	return &Properties{
		ProvisioningState:       mergeProvisioningState(complete.ProvisioningState, partial.ProvisioningState),
		OpenShiftVersion:        mergeString(complete.OpenShiftVersion, partial.OpenShiftVersion),
		PublicHostname:          mergeString(complete.PublicHostname, partial.PublicHostname),
		FQDN:                    mergeString(complete.FQDN, partial.FQDN),
		RouterProfiles:          mergeRouterProfiles(complete.RouterProfiles, partial.RouterProfiles),
		AgentPoolProfiles:       mergeAgentPoolProfiles(complete.AgentPoolProfiles, partial.AgentPoolProfiles),
		AuthProfile:             mergeAuthProfile(complete.AuthProfile, partial.AuthProfile),
		ServicePrincipalProfile: mergeServicePrincipalProfile(complete.ServicePrincipalProfile, partial.ServicePrincipalProfile),
		AzProfile:               mergeAzProfile(complete.AzProfile, partial.AzProfile),
	}
}

func mergeProvisioningState(complete, partial ProvisioningState) ProvisioningState {
	if partial == "" {
		return complete
	}
	return partial
}

func mergeRouterProfiles(complete, partial []RouterProfile) []RouterProfile {
	if complete == nil && partial == nil {
		return nil
	}

	// preconditions: each RouterProfile in partial is named uniquely; each
	// RouterProfile in complete is named uniquely

	partialMap := make(map[string]RouterProfile, len(partial))
	for _, p := range partial {
		partialMap[p.Name] = p
	}

	rv := make([]RouterProfile, 0, len(complete))

	for _, c := range complete {
		if p, found := partialMap[c.Name]; found {
			rv = append(rv, *mergeRouterProfile(&c, &p))
			delete(partialMap, c.Name)
		} else {
			rv = append(rv, c)
		}
	}

	for _, p := range partial {
		if _, found := partialMap[p.Name]; found {
			rv = append(rv, p)
		}
	}

	return rv
}

func mergeRouterProfile(complete, partial *RouterProfile) *RouterProfile {
	switch {
	case complete == nil && partial == nil:
		return nil
	case complete == nil:
		complete = &RouterProfile{}
	case partial == nil:
		partial = &RouterProfile{}
	}

	return &RouterProfile{
		Name:            mergeString(complete.Name, partial.Name),
		PublicSubdomain: mergeString(complete.PublicSubdomain, partial.PublicSubdomain),
		FQDN:            mergeString(complete.FQDN, partial.FQDN),
	}
}

func mergeAgentPoolProfiles(complete, partial []AgentPoolProfile) []AgentPoolProfile {
	if complete == nil && partial == nil {
		return nil
	}

	// preconditions: each AgentPoolProfile in partial is named uniquely; each
	// AgentPoolProfile in complete is named uniquely

	partialMap := make(map[string]AgentPoolProfile, len(partial))
	for _, p := range partial {
		partialMap[p.Name] = p
	}

	rv := make([]AgentPoolProfile, 0, len(complete))

	for _, c := range complete {
		if p, found := partialMap[c.Name]; found {
			rv = append(rv, *mergeAgentPoolProfile(&c, &p))
			delete(partialMap, c.Name)
		} else {
			rv = append(rv, c)
		}
	}

	for _, p := range partial {
		if _, found := partialMap[p.Name]; found {
			rv = append(rv, p)
		}
	}

	return rv
}

func mergeAgentPoolProfile(complete, partial *AgentPoolProfile) *AgentPoolProfile {
	switch {
	case complete == nil && partial == nil:
		return nil
	case complete == nil:
		complete = &AgentPoolProfile{}
	case partial == nil:
		partial = &AgentPoolProfile{}
	}

	return &AgentPoolProfile{
		Name:         mergeString(complete.Name, partial.Name),
		Count:        mergeIntPtr(complete.Count, partial.Count),
		VMSize:       mergeVMSize(complete.VMSize, partial.VMSize),
		VnetSubnetID: mergeString(complete.VnetSubnetID, partial.VnetSubnetID),
		OSType:       mergeOSType(complete.OSType, partial.OSType),
		Role:         mergeAgentPoolProfileRole(complete.Role, partial.Role),
	}
}

func mergeIntPtr(complete, partial *int) *int {
	if complete == nil && partial == nil {
		return nil
	}

	if partial == nil {
		rv := *complete
		return &rv
	}

	rv := *partial
	return &rv
}

func mergeVMSize(complete, partial VMSize) VMSize {
	if partial == "" {
		return complete
	}
	return partial
}

func mergeOSType(complete, partial OSType) OSType {
	if partial == "" {
		return complete
	}
	return partial
}

func mergeAgentPoolProfileRole(complete, partial AgentPoolProfileRole) AgentPoolProfileRole {
	if partial == "" {
		return complete
	}
	return partial
}

func mergeAuthProfile(complete, partial *AuthProfile) *AuthProfile {
	switch {
	case complete == nil && partial == nil:
		return nil
	case complete == nil:
		complete = &AuthProfile{}
	case partial == nil:
		partial = &AuthProfile{}
	}

	return &AuthProfile{
		IdentityProviders: mergeIdentityProviders(complete.IdentityProviders, partial.IdentityProviders),
	}
}

func mergeIdentityProviders(complete, partial []IdentityProvider) []IdentityProvider {
	if complete == nil && partial == nil {
		return nil
	}

	// preconditions: each IdentityProvider in partial is named uniquely; each
	// IdentityProvider in complete is named uniquely

	partialMap := make(map[string]IdentityProvider, len(partial))
	for _, p := range partial {
		partialMap[p.Name] = p
	}

	rv := make([]IdentityProvider, 0, len(complete))

	for _, c := range complete {
		if p, found := partialMap[c.Name]; found {
			rv = append(rv, *mergeIdentityProvider(&c, &p))
			delete(partialMap, c.Name)
		} else {
			rv = append(rv, c)
		}
	}

	for _, p := range partial {
		if _, found := partialMap[p.Name]; found {
			rv = append(rv, p)
		}
	}

	return rv
}

func mergeIdentityProvider(complete, partial *IdentityProvider) *IdentityProvider {
	rv := &IdentityProvider{
		Name: mergeString(complete.Name, partial.Name),
	}
	switch c := complete.Provider.(type) {
	case *AADIdentityProvider:
		if p, ok := partial.Provider.(*AADIdentityProvider); ok {
			rv.Provider = mergeAADIdentityProvider(c, p)
			return rv
		}
	}

	panic("unmergable IdentityProviders")
}

func mergeAADIdentityProvider(complete, partial *AADIdentityProvider) *AADIdentityProvider {
	switch {
	case complete == nil && partial == nil:
		return nil
	case complete == nil:
		complete = &AADIdentityProvider{}
	case partial == nil:
		partial = &AADIdentityProvider{}
	}

	return &AADIdentityProvider{
		Kind:     mergeString(complete.Kind, partial.Kind),
		ClientID: mergeString(complete.ClientID, partial.ClientID),
		Secret:   mergeString(complete.Secret, partial.Secret),
		TenantID: mergeString(complete.TenantID, partial.TenantID),
	}
}

func mergeServicePrincipalProfile(complete, partial *ServicePrincipalProfile) *ServicePrincipalProfile {
	switch {
	case complete == nil && partial == nil:
		return nil
	case complete == nil:
		complete = &ServicePrincipalProfile{}
	case partial == nil:
		partial = &ServicePrincipalProfile{}
	}

	return &ServicePrincipalProfile{
		ClientID: mergeString(complete.ClientID, partial.ClientID),
		Secret:   mergeString(complete.Secret, partial.Secret),
	}
}

func mergeAzProfile(complete, partial *AzProfile) *AzProfile {
	switch {
	case complete == nil && partial == nil:
		return nil
	case complete == nil:
		complete = &AzProfile{}
	case partial == nil:
		partial = &AzProfile{}
	}

	return &AzProfile{
		TenantID:       mergeString(complete.TenantID, partial.TenantID),
		SubscriptionID: mergeString(complete.SubscriptionID, partial.SubscriptionID),
		ResourceGroup:  mergeString(complete.ResourceGroup, partial.ResourceGroup),
	}
}
