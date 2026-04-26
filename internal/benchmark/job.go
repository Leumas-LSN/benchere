package benchmark

import (
	"fmt"
	"time"

	"github.com/Leumas-LSN/benchere/internal/db"
	"github.com/Leumas-LSN/benchere/internal/stress"
	"github.com/google/uuid"
)

type Mode string

const (
	ModeStorage Mode = "storage"
	ModeCPU     Mode = "cpu"
	ModeMixed   Mode = "mixed"
)

type JobConfig struct {
	Name          string
	ClientName    string
	Mode          Mode
	WorkerCount   int
	WorkerCPU     int
	WorkerRAMMB   int
	OSDiskGB      int
	DataDisks     int
	DataDiskGB    int
	ProxmoxNode   string
	StoragePool   string
	ImageStorage  string
	CloudImageURL string
	Profiles      []string
	StressConfig  *stress.Config
}

func NewJob(cfg JobConfig) db.Job {
	return db.Job{
		ID:         uuid.NewString(),
		Name:       cfg.Name,
		ClientName: cfg.ClientName,
		Status:     "pending",
		Mode:       string(cfg.Mode),
		CreatedAt:  time.Now(),
	}
}

var validTransitions = map[string][]string{
	"pending":      {"provisioning", "failed"},
	"provisioning": {"running", "failed"},
	"running":      {"done", "failed", "cancelled"},
}

func CanTransition(from, to string) bool {
	nexts, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, n := range nexts {
		if n == to {
			return true
		}
	}
	return false
}

func AssertTransition(from, to string) error {
	if !CanTransition(from, to) {
		return fmt.Errorf("invalid transition %s → %s", from, to)
	}
	return nil
}
