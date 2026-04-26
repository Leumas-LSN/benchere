package api

import (
	"encoding/json"
	"net/http"
)

func (s *Server) getVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	v := s.Version
	if v == "" {
		v = "dev"
	}
	json.NewEncoder(w).Encode(map[string]string{"version": v})
}
