package fakerp

import (
	"path/filepath"

	"github.com/go-chi/chi/middleware"
)

func (s *Server) setupRoutes() {
	s.router.Use(middleware.DefaultCompress)
	s.router.Use(middleware.RedirectSlashes)
	s.router.Use(middleware.Recoverer)
	s.router.Use(s.logger)
	s.router.Use(s.validator)
	s.router.Use(s.context)

	s.router.Delete(s.basePath, s.handleDelete)
	s.router.Get(s.basePath, s.handleGet)
	s.router.Put(s.basePath, s.handlePut)
	s.router.Get(filepath.Join("/admin", s.basePath), s.handleGet)
	s.router.Put(filepath.Join("/admin", s.basePath), s.handlePut)
	s.router.Get(filepath.Join("/admin", s.basePath, "/listBackups"), s.handleListBackups)
	s.router.Put(filepath.Join("/admin", s.basePath, "/restore/{backupName}"), s.handleRestore)
	s.router.Put(filepath.Join("/admin", s.basePath, "/rotate/secrets"), s.handleRotateSecrets)
	s.router.Put(filepath.Join("/admin", s.basePath, "/rotate/certificates"), s.handleRotateCertificates)
	s.router.Put(filepath.Join("/admin", s.basePath, "/rotate/certificatesAndSecrets"), s.handleRotateCertificatesAndSecrets)
	s.router.Get(filepath.Join("/admin", s.basePath, "/status"), s.handleGetControlPlanePods)
	s.router.Put(filepath.Join("/admin", s.basePath, "/forceUpdate"), s.handleForceUpdate)
	s.router.Get(filepath.Join("/admin", s.basePath, "/listClusterVMs"), s.handleListClusterVMs)
	s.router.Put(filepath.Join("/admin", s.basePath, "/reimage/{hostname}"), s.handleReimage)
	s.router.Put(filepath.Join("/admin", s.basePath, "/backup/{backupName}"), s.handleBackup)
	s.router.Put(filepath.Join("/admin", s.basePath, "/runCommand/{hostname}/{command}"), s.handleRunCommand)
	s.router.Get(filepath.Join("/admin", s.basePath, "/pluginVersion"), s.handleGetPluginVersion)
	s.router.Put(filepath.Join("/admin", s.basePath, "/restart/{hostname}"), s.handleRestart)
}
