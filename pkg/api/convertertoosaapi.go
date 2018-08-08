package api

import (
	"github.com/openshift/openshift-azure/pkg/api/v1"
)

// ConvertContainerServiceToVLabsOpenShiftCluster converts from a
// ContainerService to a v1.OpenShiftCluster.
func ConvertContainerServiceToVLabsOpenShiftCluster(cs *ContainerService) *v1.OpenShiftCluster {
	oc := &v1.OpenShiftCluster{
		ID:       cs.ID,
		Location: cs.Location,
		Name:     cs.Name,
		Tags:     cs.Tags,
		Type:     cs.Type,
	}

	if cs.Plan != nil {
		oc.Plan = &v1.ResourcePurchasePlan{
			Name:          cs.Plan.Name,
			Product:       cs.Plan.Product,
			PromotionCode: cs.Plan.PromotionCode,
			Publisher:     cs.Plan.Publisher,
		}
	}

	if cs.Properties != nil {
		oc.Properties = &v1.Properties{
			ProvisioningState: v1.ProvisioningState(cs.Properties.ProvisioningState),
		}

		if cs.Properties.OrchestratorProfile != nil {
			oc.Properties.OpenShiftVersion = cs.Properties.OrchestratorProfile.OrchestratorVersion

			if cs.Properties.OrchestratorProfile.OpenShiftConfig != nil {
				oc.Properties.PublicHostname = cs.Properties.OrchestratorProfile.OpenShiftConfig.PublicHostname

				oc.Properties.RouterProfiles = make([]v1.RouterProfile, len(cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles))
				for i, rp := range cs.Properties.OrchestratorProfile.OpenShiftConfig.RouterProfiles {
					oc.Properties.RouterProfiles[i] = v1.RouterProfile{
						Name:            rp.Name,
						PublicSubdomain: rp.PublicSubdomain,
						FQDN:            rp.FQDN,
					}
				}
			}
		}

		if cs.Properties.MasterProfile != nil {
			oc.Properties.FQDN = cs.Properties.MasterProfile.FQDN
		}

		if cs.Properties.ServicePrincipalProfile != nil {
			oc.Properties.ServicePrincipalProfile = v1.ServicePrincipalProfile{
				ClientID: cs.Properties.ServicePrincipalProfile.ClientID,
				Secret:   cs.Properties.ServicePrincipalProfile.Secret,
			}
		}

		// -1 because master profile moves to its own field
		oc.Properties.AgentPoolProfiles = make([]v1.AgentPoolProfile, len(cs.Properties.AgentPoolProfiles)-1)
		for i, app := range cs.Properties.AgentPoolProfiles {
			if app.Role == AgentPoolProfileRoleMaster {
				oc.Properties.MasterPoolProfile = v1.MasterPoolProfile{
					ProfileSpec: v1.ProfileSpec{
						Name:         app.Name,
						Count:        app.Count,
						VMSize:       app.VMSize,
						OSType:       v1.OSType(app.OSType),
						VnetSubnetID: app.VnetSubnetID,
					},
				}
			} else {
				oc.Properties.AgentPoolProfiles[i] = v1.AgentPoolProfile{
					ProfileSpec: v1.ProfileSpec{
						Name:         app.Name,
						Count:        app.Count,
						VMSize:       app.VMSize,
						OSType:       v1.OSType(app.OSType),
						VnetSubnetID: app.VnetSubnetID,
					},
					Role: v1.AgentPoolProfileRole(app.Role),
				}
			}
		}
	}

	return oc
}
