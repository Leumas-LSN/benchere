package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/Leumas-LSN/benchere/internal/ansible"
	"github.com/Leumas-LSN/benchere/internal/api"
	"github.com/Leumas-LSN/benchere/internal/benchmark"
	"github.com/Leumas-LSN/benchere/internal/config"
	"github.com/Leumas-LSN/benchere/internal/db"
	debugpkg "github.com/Leumas-LSN/benchere/internal/debug"
	"github.com/Leumas-LSN/benchere/internal/proxmox"
	"github.com/Leumas-LSN/benchere/internal/report"
	"github.com/Leumas-LSN/benchere/internal/stress"
	"github.com/Leumas-LSN/benchere/internal/ws"
)

// Version is set at build time via -ldflags '-X main.Version=v1.2.3'.
// Defaults to 'dev' for local/unstamped builds.
var Version = "dev"

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer database.Close()

	sshKeyPath := envOr("BENCHERE_SSH_KEY", "/opt/benchere/id_rsa")
	log.Printf("[startup] sshKeyPath=%q", sshKeyPath)
	playbookDir := envOr("BENCHERE_PLAYBOOK_DIR", "/opt/benchere/ansible/playbooks")
	elbenchoDeb := envOr("BENCHERE_ELBENCHO_DEB", "/opt/benchere/assets/elbencho_amd64.deb")
	profilesDir := envOr("BENCHERE_PROFILES_DIR", "/opt/benchere/profiles")
	outputDir := envOr("BENCHERE_OUTPUT_DIR", "/opt/benchere/output")
	jobsDir := envOr("BENCHERE_JOBS_DIR", "/var/lib/benchere/jobs")

	os.MkdirAll(profilesDir, 0755)
	os.MkdirAll(outputDir, 0755)
	os.MkdirAll(jobsDir, 0755)
	os.MkdirAll(filepath.Dir(cfg.DBPath), 0755)

	// Retain per-job artifact dirs for 7 days. One log line, fire and forget.
	if cleaned := debugpkg.CleanOldJobDirs(jobsDir, 7*24*time.Hour); cleaned > 0 {
		log.Printf("[startup] cleaned %d job dirs older than 7d in %s", cleaned, jobsDir)
	}

	proxmoxURL, _ := database.GetSetting("proxmox_url")
	proxmoxToken, _ := database.GetSetting("proxmox_token")

	pxClient := proxmox.NewClient(proxmoxURL, proxmoxToken)
	ansibleRunner := ansible.NewRunner(playbookDir, sshKeyPath, elbenchoDeb)
	stressRunner := stress.NewRunner(sshKeyPath)

	hub := ws.NewHub()
	go hub.Run()

	orch := &benchmark.Orchestrator{
		DB:          database,
		Proxmox:     pxClient,
		Ansible:     ansibleRunner,
		Stress:      stressRunner,
		Hub:         hub,
		SSHKey:      sshKeyPath,
		ProfilesDir: profilesDir,
		OutputDir:   outputDir,
		JobsDir:     jobsDir,
	}

	go orch.RecoverOrphanedJobs(context.Background())

	reporter := report.NewGenerator(database, profilesDir, Version)

	srv := &api.Server{
		DB:           database,
		Hub:          hub,
		Orchestrator: orch,
		Reporter:     reporter,
		Version:      Version,
		JobsDir:      jobsDir,
	}

	log.Printf("benchere %s listening on :%s", Version, cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, srv.Router()); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
