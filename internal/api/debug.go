package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/Leumas-LSN/benchere/internal/debug"
	"github.com/Leumas-LSN/benchere/internal/proxmox"
	"github.com/go-chi/chi/v5"
)

// downloadDebug streams a tar.gz bundle for the given job. Returns:
//
//	404 when the job id is unknown
//	409 when the job is still in flight (status not done|failed|cancelled)
//	200 application/gzip otherwise, never buffered in memory
func (s *Server) downloadDebug(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	job, err := s.DB.GetJob(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "job not found")
		} else {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	switch job.Status {
	case "done", "failed", "cancelled":
		// allowed
	default:
		writeJSONError(w, http.StatusConflict,
			"job is still in flight (status="+job.Status+"); wait for it to finish before downloading the debug bundle")
		return
	}

	short := id
	if len(short) >= 8 {
		short = short[:8]
	}
	stamp := time.Now().UTC().Format("20060102-1504")
	filename := fmt.Sprintf("benchere-debug-%s-%s.tar.gz", short, stamp)

	// Headers must be set before any write.
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")

	src := debug.BundleSources{
		DB:             s.DB,
		Proxmox:        s.proxmoxClient(),
		JobsDir:        s.JobsDir,
		Version:        s.Version,
		JobID:          id,
		IncludeJournal: true,
		IncludeProxmox: true,
		IncludeCeph:    true,
		IncludeDBCopy:  true,
	}

	// Use the request context so a client disconnect cancels in-flight
	// collector calls (notably journalctl and Proxmox http).
	if err := debug.Build(r.Context(), w, src); err != nil {
		// Status code is already 200 by the time we get here: a header has
		// been sent. We log and let the truncated tar reach the client; an
		// error sentinel inside errors.log would have captured collector
		// failures. Critical assembler errors are vanishingly rare (most
		// collector failures are caught before).
		log.Printf("[debug] bundle build error for job %s: %v", id, err)
	}
}

// proxmoxClient returns a fresh Proxmox client built from the current DB
// settings. nil when proxmox is not configured. Used by the debug bundle so
// snapshots reflect live cluster state, not a stale client cached at boot.
func (s *Server) proxmoxClient() *proxmox.Client {
	if s == nil {
		return nil
	}
	url, _ := s.DB.GetSetting("proxmox_url")
	token, _ := s.DB.GetSetting("proxmox_token")
	if strings.TrimSpace(url) == "" || strings.TrimSpace(token) == "" {
		return nil
	}
	return proxmox.NewClient(url, token)
}

// writeJSONError emits {"error": "..."} with the given status, used by the
// debug endpoint so the frontend can surface the failure reason without
// parsing free-form text.
func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
