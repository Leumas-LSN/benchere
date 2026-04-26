package db_test

import (
	"testing"
	"time"

	"github.com/Leumas-LSN/benchere/internal/db"
)

func TestOpen_RunsMigrations(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	_, err = database.Exec("INSERT INTO jobs(id,name,client_name,status,mode,created_at) VALUES('x','n','c','pending','storage',datetime('now'))")
	if err != nil {
		t.Fatalf("jobs table not created: %v", err)
	}
}

func TestJobCRUD(t *testing.T) {
	database, _ := db.Open(":memory:")
	defer database.Close()

	j := db.Job{ID: "j1", Name: "test", ClientName: "client", Status: "pending", Mode: "storage", CreatedAt: time.Now()}
	if err := database.CreateJob(j); err != nil {
		t.Fatal(err)
	}
	got, err := database.GetJob("j1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "test" {
		t.Errorf("got name %q, want %q", got.Name, "test")
	}
	if err := database.UpdateJobStatus("j1", "running"); err != nil {
		t.Fatal(err)
	}
	got, _ = database.GetJob("j1")
	if got.Status != "running" {
		t.Errorf("status not updated, got %q", got.Status)
	}
}

func TestSettingCRUD(t *testing.T) {
	database, _ := db.Open(":memory:")
	defer database.Close()

	if err := database.SetSetting("proxmox_url", "https://10.0.0.1:8006"); err != nil {
		t.Fatal(err)
	}
	val, err := database.GetSetting("proxmox_url")
	if err != nil {
		t.Fatal(err)
	}
	if val != "https://10.0.0.1:8006" {
		t.Errorf("got %q, want URL", val)
	}
}
