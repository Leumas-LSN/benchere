package api

import (
	"encoding/json"
	"net/http"

	"github.com/Leumas-LSN/benchere/internal/benchmark"
	"github.com/Leumas-LSN/benchere/internal/db"
)

// estimateJob backs POST /api/jobs/estimate. The wizard calls this after
// each step change to refresh the estimated cost panel without hitting
// the heavyweight createJob flow. The handler is read-only: it does not
// touch the DB beyond loading profile metadata for runtime overrides
// and rw= detection.
func (s *Server) estimateJob(w http.ResponseWriter, r *http.Request) {
	var in benchmark.EstimateInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Build a name -> Profile lookup so the estimator can pick up
	// profile-level runtime/ramp overrides and detect writes via rw=.
	// Failure to load profiles is non-fatal: the estimator falls back
	// to the wizard-provided runtime and assumes writes when unknown.
	var lookup map[string]*db.Profile
	if list, err := s.DB.ListProfiles(); err == nil {
		lookup = make(map[string]*db.Profile, len(list))
		for i := range list {
			p := list[i]
			lookup[p.Name] = &p
		}
	}

	out, err := benchmark.Estimate(in, lookup)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}
