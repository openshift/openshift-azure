package fakerp

import (
	"path/filepath"

	"github.com/go-chi/chi/middleware"
)

func (s *Server) SetupRoutes() {
	s.router.Use(middleware.DefaultCompress)
	s.router.Use(middleware.RedirectSlashes)
	s.router.Use(middleware.Recoverer)
	s.router.Use(s.logger)
	s.router.Use(s.validator)

	s.router.Delete(s.basePath, s.handleDelete)
	s.router.Get(s.basePath, s.handleGet)
	s.router.Put(s.basePath, s.handlePut)
	s.router.Delete(filepath.Join("/admin", s.basePath), s.handleDelete)
	s.router.Get(filepath.Join("/admin", s.basePath), s.handleGet)
	s.router.Put(filepath.Join("/admin", s.basePath), s.handlePut)
	s.router.Put(filepath.Join("/admin", s.basePath, "/restore"), s.handleRestore)
	s.router.Put(filepath.Join("/admin", s.basePath, "/rotate/secrets"), s.handleRotateSecrets)
	s.router.Get(filepath.Join("/admin", s.basePath, "/status"), s.handleGetControlPlanePods)
	s.router.Put(filepath.Join("/admin", s.basePath, "/forceUpdate"), s.handleForceUpdate)
	s.router.Put(filepath.Join("/admin", s.basePath, "/reimage/{hostname}"), s.handleReimage)
}
