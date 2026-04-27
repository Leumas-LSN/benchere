package benchmark

import (
	"reflect"
	"testing"
)

func TestSplitWorkers_RoundRobinStable(t *testing.T) {
	got := SplitWorkers([]string{"A", "B", "C"}, 2)
	want := []WorkerSpec{
		{Index: 0, Node: "A"},
		{Index: 1, Node: "B"},
		{Index: 2, Node: "C"},
		{Index: 3, Node: "A"},
		{Index: 4, Node: "B"},
		{Index: 5, Node: "C"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}

func TestSplitWorkers_SingleNode(t *testing.T) {
	got := SplitWorkers([]string{"only"}, 3)
	if len(got) != 3 {
		t.Fatalf("expected 3 workers, got %d", len(got))
	}
	for i, spec := range got {
		if spec.Node != "only" {
			t.Errorf("worker %d: got node %q, want \"only\"", i, spec.Node)
		}
	}
}

func TestSplitWorkers_EmptyNodesYieldsNil(t *testing.T) {
	if got := SplitWorkers(nil, 5); got != nil {
		t.Errorf("expected nil for empty nodes, got %v", got)
	}
	if got := SplitWorkers([]string{}, 5); got != nil {
		t.Errorf("expected nil for empty nodes slice, got %v", got)
	}
}

func TestSplitWorkers_ZeroPerNodeYieldsNil(t *testing.T) {
	if got := SplitWorkers([]string{"A"}, 0); got != nil {
		t.Errorf("expected nil for perNode=0, got %v", got)
	}
}
