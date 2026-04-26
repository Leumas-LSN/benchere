package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/Leumas-LSN/benchere/internal/proxmox"
)

type overviewResponse struct {
	ActiveJobs []overviewJob `json:"active_jobs"`
	RecentJobs []overviewJob `json:"recent_jobs"`
	Cluster    []clusterNode `json:"cluster"`
}

type overviewJob struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	ClientName string  `json:"client_name"`
	Status     string  `json:"status"`
	Mode       string  `json:"mode"`
	CreatedAt  string  `json:"created_at"`
	FinishedAt *string `json:"finished_at,omitempty"`
}

type clusterNode struct {
	Name    string  `json:"name"`
	CPUPct  float64 `json:"cpu_pct"`
	RAMPct  float64 `json:"ram_pct"`
	LoadAvg float64 `json:"load_avg"`
	Online  bool    `json:"online"`
}

func (srv *Server) getOverview(w http.ResponseWriter, r *http.Request) {
	resp := overviewResponse{
		ActiveJobs: []overviewJob{},
		RecentJobs: []overviewJob{},
		Cluster:    []clusterNode{},
	}

	active, err := srv.DB.ListActiveJobs()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, j := range active {
		oj := overviewJob{
			ID: j.ID, Name: j.Name, ClientName: j.ClientName,
			Status: j.Status, Mode: j.Mode,
			CreatedAt: j.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if j.FinishedAt != nil {
			ts := j.FinishedAt.Format("2006-01-02T15:04:05Z")
			oj.FinishedAt = &ts
		}
		resp.ActiveJobs = append(resp.ActiveJobs, oj)
	}

	recent, err := srv.DB.ListRecentJobs(5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, j := range recent {
		oj := overviewJob{
			ID: j.ID, Name: j.Name, ClientName: j.ClientName,
			Status: j.Status, Mode: j.Mode,
			CreatedAt: j.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if j.FinishedAt != nil {
			ts := j.FinishedAt.Format("2006-01-02T15:04:05Z")
			oj.FinishedAt = &ts
		}
		resp.RecentJobs = append(resp.RecentJobs, oj)
	}

	proxmoxURL, err := srv.DB.GetSetting("proxmox_url")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("overview: GetSetting proxmox_url: %v", err)
	}
	proxmoxToken, err := srv.DB.GetSetting("proxmox_token")
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("overview: GetSetting proxmox_token: %v", err)
	}
	if proxmoxURL != "" && proxmoxToken != "" {
		pxCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		pxClient := proxmox.NewClient(proxmoxURL, proxmoxToken)
		if nodes, err := pxClient.GetNodes(pxCtx); err == nil {
			for _, node := range nodes {
				cn := clusterNode{Name: node, Online: true}
				if st, err := pxClient.GetNodeStatus(pxCtx, node); err == nil {
					cn.CPUPct = st.CPUPct
					cn.RAMPct = st.RAMPct
					cn.LoadAvg = st.LoadAvg
				} else {
					cn.Online = false
				}
				resp.Cluster = append(resp.Cluster, cn)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
