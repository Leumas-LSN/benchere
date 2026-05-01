package benchmark

import (
	"context"
	"time"

	"github.com/Leumas-LSN/benchere/internal/db"
	"github.com/Leumas-LSN/benchere/internal/proxmox"
	"github.com/Leumas-LSN/benchere/internal/ws"
	"github.com/google/uuid"
)

// PollProxmoxMetrics polls node and VM metrics every 2s until ctx is cancelled.
// Cumulative VM counters (netin/netout/diskread/diskwrite) are converted to
// per-second rates using the previous tick as a reference.
func PollProxmoxMetrics(ctx context.Context, jobID string, pxClient *proxmox.Client, database *db.DB, hub *ws.Hub, workers []db.Worker) {
	const interval = 2 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	type prev struct {
		netIn, netOut, diskRead, diskWrite float64
		ts                                 time.Time
	}
	prevs := make(map[string]prev) // worker.ID to previous sample

	type satState struct {
		cpuStreak int // consecutive samples > 80%
	}
	satStates := make(map[string]*satState)
	const cpuSatThresh = 80.0
	const cpuSatStreak = 5

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()

			nodes, err := pxClient.GetNodes(ctx)
			if err == nil {
				for _, node := range nodes {
					status, err := pxClient.GetNodeStatus(ctx, node)
					if err != nil {
						continue
					}
					snap := db.ProxmoxSnapshot{
						ID: uuid.NewString(), JobID: jobID, Timestamp: now,
						NodeName: node, CPUPct: status.CPUPct, RAMPct: status.RAMPct, LoadAvg: status.LoadAvg,
					}
					_ = database.InsertProxmoxSnapshot(snap)
					hub.Broadcast(ws.Event{
						Type:  ws.EventProxmoxNode,
						JobID: jobID,
						Payload: ws.MustMarshal(ws.ProxmoxNodePayload{
							NodeName: node, CPUPct: status.CPUPct, RAMPct: status.RAMPct, LoadAvg: status.LoadAvg,
						}),
					})
				}
			}

			for _, w := range workers {
				vmStatus, err := pxClient.GetVMStatus(ctx, w.ProxmoxNode, w.VMID)
				if err != nil {
					continue
				}

				// Compute per-second rates from cumulative counters.
				var netInBps, netOutBps, diskReadBps, diskWriteBps float64
				if p, ok := prevs[w.ID]; ok {
					dt := now.Sub(p.ts).Seconds()
					if dt > 0 {
						netInBps = (vmStatus.NetIn - p.netIn) / dt
						netOutBps = (vmStatus.NetOut - p.netOut) / dt
						diskReadBps = (vmStatus.DiskRead - p.diskRead) / dt
						diskWriteBps = (vmStatus.DiskWrite - p.diskWrite) / dt
					}
				}
				prevs[w.ID] = prev{
					netIn:     vmStatus.NetIn,
					netOut:    vmStatus.NetOut,
					diskRead:  vmStatus.DiskRead,
					diskWrite: vmStatus.DiskWrite,
					ts:        now,
				}

				snap := db.ProxmoxVMSnapshot{
					ID: uuid.NewString(), JobID: jobID, Timestamp: now,
					WorkerID: w.ID, CPUPct: vmStatus.CPUPct,
				}
				_ = database.InsertProxmoxVMSnapshot(snap)
				hub.Broadcast(ws.Event{
					Type:  ws.EventProxmoxVM,
					JobID: jobID,
					Payload: ws.MustMarshal(ws.ProxmoxVMPayload{
						WorkerID:     w.ID,
						CPUPct:       vmStatus.CPUPct,
						RAMPct:       vmStatus.RAMPct,
						NetInBps:     netInBps,
						NetOutBps:    netOutBps,
						DiskReadBps:  diskReadBps,
						DiskWriteBps: diskWriteBps,
					}),
				})

				st, ok := satStates[w.ID]
				if !ok {
					st = &satState{}
					satStates[w.ID] = st
				}
				if vmStatus.CPUPct >= cpuSatThresh {
					st.cpuStreak++
				} else {
					st.cpuStreak = 0
				}
				if st.cpuStreak == cpuSatStreak {
					hub.Broadcast(ws.Event{
						Type:  ws.EventWorkerSaturation,
						JobID: jobID,
						Payload: ws.MustMarshal(ws.WorkerSaturationPayload{
							WorkerID: w.ID, Kind: "cpu", Value: vmStatus.CPUPct, Threshold: cpuSatThresh,
						}),
					})
				}
			}
		}
	}
}
