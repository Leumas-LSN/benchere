package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Leumas-LSN/benchere/internal/benchmark"
	"github.com/Leumas-LSN/benchere/internal/db"
)

func TestEstimateJob_ReturnsWallclockAndBytes(t *testing.T) {
	database, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer database.Close()

	srv := &Server{DB: database}
	body := benchmark.EstimateInput{
		Mode:           "storage",
		Workers:        4,
		DataDisks:      1,
		DataDiskGB:     32,
		Profiles:       []string{"oltp-4k-70-30"},
		RuntimeSeconds: 120,
		RampSeconds:    30,
	}
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/jobs/estimate", bytes.NewReader(buf))
	rec := httptest.NewRecorder()

	srv.estimateJob(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var out benchmark.EstimateOutput
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.WallclockSec <= 0 {
		t.Errorf("wallclock = %d, want > 0", out.WallclockSec)
	}
	if out.BytesWritten <= 0 {
		t.Errorf("bytes = %d, want > 0", out.BytesWritten)
	}
}

func TestEstimateJob_RejectsBadJSON(t *testing.T) {
	database, _ := db.Open(":memory:")
	defer database.Close()

	srv := &Server{DB: database}
	req := httptest.NewRequest(http.MethodPost, "/api/jobs/estimate", bytes.NewReader([]byte("not json")))
	rec := httptest.NewRecorder()

	srv.estimateJob(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}
