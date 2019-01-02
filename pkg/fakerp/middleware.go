package fakerp

import (
	"net/http"
)

func (s *Server) logger(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.log.Debugf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
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
		}
		handler.ServeHTTP(w, r)
	})
}
