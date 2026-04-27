package benchmark

// WorkerSpec describes one VM to provision: its zero-based index (used for
// naming and IP assignment ordering) and the Proxmox node it lives on.
type WorkerSpec struct {
	Index int
	Node  string
	VMID  int // populated later by the orchestrator after VMID allocation
}

// SplitWorkers distributes count = len(nodes) * perNode workers across the
// nodes in round-robin order. Worker indices are stable: index i goes to
// nodes[i % len(nodes)]. Returns nil if either input is empty/zero.
func SplitWorkers(nodes []string, perNode int) []WorkerSpec {
	if len(nodes) == 0 || perNode <= 0 {
		return nil
	}
	total := len(nodes) * perNode
	out := make([]WorkerSpec, total)
	for i := 0; i < total; i++ {
		out[i] = WorkerSpec{Index: i, Node: nodes[i%len(nodes)]}
	}
	return out
}
