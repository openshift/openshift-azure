package fakerp

import (
	"context"
	"net/http"

	"github.com/Azure/go-autorest/autorest/azure"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/fakerp/client"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
)

func (s *Server) logger(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.log.Debugf("starting: %s %s", r.Method, r.URL)
		handler.ServeHTTP(w, r)
		s.log.Debugf("ending:   %s %s", r.Method, r.URL)
	})
}

func (s *Server) validator(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			select {
			case s.inProgress <- struct{}{}:
				// continue
			default:
				// did not get the lock
				resp := "423 Locked: Processing another in-flight request"
				s.log.Debug(resp)
				http.Error(w, resp, http.StatusLocked)
				return
			}
			defer func() {
				// drain once we are done processing this request
				<-s.inProgress
			}()
		}
		handler.ServeHTTP(w, r)
	})
}

func (s *Server) context(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		authorizer, err := azureclient.NewAuthorizerFromEnvironment("")
		if err != nil {
			s.badRequest(w, err.Error())
			return
		}
		ctx = context.WithValue(ctx, api.ContextKeyClientAuthorizer, authorizer)

		graphauthorizer, err := azureclient.NewAuthorizerFromEnvironment(azure.PublicCloud.GraphEndpoint)
		if err != nil {
			s.badRequest(w, err.Error())
			return
		}
		ctx = context.WithValue(ctx, contextKeyGraphClientAuthorizer, graphauthorizer)

		vaultauthorizer, err := azureclient.NewAuthorizerFromEnvironment(azureclient.KeyVaultEndpoint)
		if err != nil {
			s.badRequest(w, err.Error())
			return
		}
		ctx = context.WithValue(ctx, api.ContextKeyVaultClientAuthorizer, vaultauthorizer)

		// we ignore errors, as those are handled by code using the object
		cs, _ := s.store.Get()
		ctx = context.WithValue(ctx, contextKeyContainerService, cs)

		// we use context object config for all methods, except CREATE PUT.
		// at the creation time cs does not exist in the store/context
		var conf *client.Config
		if cs != nil {
			conf, err = client.NewServerConfig(s.log, cs)
			if err != nil {
				return
			}
			ctx = context.WithValue(ctx, contextKeyConfig, conf)

			peIP, err := getPrivateEndpointIP(ctx, s.log, cs.Properties.AzProfile.SubscriptionID, conf.ManagementResourceGroup, cs.Properties.AzProfile.ResourceGroup)
			if err != nil {
				s.adminreply(w, err, nil)
			}
			ctx = context.WithValue(ctx, contextKeyPrivateEndpointIP, peIP)
		}

		handler.ServeHTTP(w, r.WithContext(ctx))
	})
}
