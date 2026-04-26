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
func PollProxmoxMetrics(ctx context.Context, jobID string, pxClient *proxmox.Client, database *db.DB, hub *ws.Hub, workers []db.Worker) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			nodes, err := pxClient.GetNodes(ctx)
			if err == nil {
				for _, node := range nodes {
					status, err := pxClient.GetNodeStatus(ctx, node)
					if err != nil {
						continue
					}
					snap := db.ProxmoxSnapshot{
						ID: uuid.NewString(), JobID: jobID, Timestamp: time.Now(),
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
				snap := db.ProxmoxVMSnapshot{
					ID: uuid.NewString(), JobID: jobID, Timestamp: time.Now(),
					WorkerID: w.ID, CPUPct: vmStatus.CPUPct,
				}
				_ = database.InsertProxmoxVMSnapshot(snap)
				hub.Broadcast(ws.Event{
					Type:  ws.EventProxmoxVM,
					JobID: jobID,
					Payload: ws.MustMarshal(ws.ProxmoxVMPayload{
						WorkerID: w.ID, CPUPct: vmStatus.CPUPct,
					}),
				})
			}
		}
	}
}
