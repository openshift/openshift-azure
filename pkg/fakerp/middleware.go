package fakerp

import (
	"context"
	"net/http"

	internalapi "github.com/openshift/openshift-azure/pkg/api"
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
		if r.Method == http.MethodPut {
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
		ctx = context.WithValue(ctx, internalapi.ContextKeyClientAuthorizer, authorizer)

		vaultauthorizer, err := azureclient.NewAuthorizerFromEnvironment(azureclient.KeyVaultEndpoint)
		if err != nil {
			s.badRequest(w, err.Error())
			return
		}
		ctx = context.WithValue(ctx, internalapi.ContextKeyVaultClientAuthorizer, vaultauthorizer)

		handler.ServeHTTP(w, r.WithContext(ctx))
	})
}
