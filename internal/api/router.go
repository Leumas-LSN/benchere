package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/Leumas-LSN/benchere/internal/benchmark"
	"github.com/Leumas-LSN/benchere/internal/db"
	"github.com/Leumas-LSN/benchere/internal/report"
	"github.com/Leumas-LSN/benchere/internal/ws"
	bweb "github.com/Leumas-LSN/benchere/web"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Server struct {
	DB           *db.DB
	Hub          *ws.Hub
	Orchestrator *benchmark.Orchestrator
	Reporter     *report.Generator
	Version      string

	// JobsDir is the per-job artifact root that the debug bundle endpoint
	// reads from. Mirrors Orchestrator.JobsDir so the API does not need to
	// reach into the orchestrator just for that path.
	JobsDir string
}

func (s *Server) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			if req.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, req)
		})
	})

	r.Get("/ws", s.handleWS)
	r.Route("/api", func(r chi.Router) {
		r.Get("/settings", s.getSettings)
		r.Post("/settings", s.postSettings)
		r.Post("/proxmox/test", s.testProxmox)
		r.Get("/proxmox/storages", s.getStorages)
		r.Get("/proxmox/bridges", s.getBridges)
		r.Get("/jobs", s.listJobs)
		r.Post("/jobs", s.createJob)
		r.Get("/jobs/{id}", s.getJob)
		r.Get("/jobs/{id}/results", s.getJobResults)
		r.Get("/jobs/{id}/summaries", s.listSummaries)
		r.Get("/jobs/{id}/results.csv", s.exportJobCSV)
		r.Get("/jobs/{id}/debug", s.downloadDebug)
		r.Post("/jobs/{id}/cancel", s.cancelJob)
		r.Delete("/jobs", s.clearHistory)
		r.Get("/jobs/{id}/workers", s.listWorkers)
		r.Get("/jobs/{id}/report.pdf", s.reportPDF)
		r.Get("/jobs/{id}/report.html", s.reportHTML)
		r.Get("/overview", s.getOverview)
		r.Get("/version", s.getVersion)
		r.Get("/profiles", s.listProfiles)
		r.Post("/profiles", s.createProfile)
		r.Put("/profiles/{id}", s.updateProfile)
		r.Delete("/profiles/{id}", s.deleteProfile)
	})

	// Serve Vue3 SPA - API and WS routes already matched above.
	dist, err := fs.Sub(bweb.DistFS, "dist")
	if err != nil {
		panic("web: could not sub into dist: " + err.Error())
	}
	r.Handle("/*", spaHandler(dist))

	return r
}

// spaHandler serves static files and falls back to index.html for SPA
// routing (HTML5 history mode).
func spaHandler(fsys fs.FS) http.Handler {
	fileServer := http.FileServerFS(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		f, err := fsys.Open(path)
		if err != nil {
			// Unknown path: serve index.html so Vue Router handles it.
			http.ServeFileFS(w, r, fsys, "index.html")
			return
		}
		f.Close()
		fileServer.ServeHTTP(w, r)
	})
}
