package vectorclock

import (
	"testing"
)

func TestVectorClock_CausalityComparison(t *testing.T) {
	vc1 := NewVectorClock()
	vc2 := NewVectorClock()

	// Equal initially
	if rel := vc1.Compare(vc2); rel != Equal {
		t.Errorf("expected Equal, got %v", rel)
	}

	// Increment node A on vc1
	vc1.Increment("nodeA")
	if rel := vc1.Compare(vc2); rel != After {
		t.Errorf("expected vc1 After vc2, got %v", rel)
	}
	if rel := vc2.Compare(vc1); rel != Before {
		t.Errorf("expected vc2 Before vc1, got %v", rel)
	}

	// Increment node B on vc2 (making them Concurrent)
	vc2.Increment("nodeB")
	if rel := vc1.Compare(vc2); rel != Concurrent {
		t.Errorf("expected Concurrent, got %v", rel)
	}

	// Merge vc2 into vc1
	vc1.Merge(vc2)
	// vc1 is now {nodeA:1, nodeB:1}, vc2 is {nodeB:1}
	if rel := vc1.Compare(vc2); rel != After {
		t.Errorf("expected vc1 After vc2 after merge, got %v", rel)
	}
}
