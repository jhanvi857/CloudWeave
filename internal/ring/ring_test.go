package ring

import (
	"fmt"
	"testing"
)

func TestGetNodesForKey_Deterministic(t *testing.T) {
	r := New()
	for _, n := range []string{"node-a", "node-b", "node-c"} {
		r.AddNode(n)
	}

	first := r.GetNodesForKey("file-123", 3)
	second := r.GetNodesForKey("file-123", 3)

	if fmt.Sprint(first) != fmt.Sprint(second) {
		t.Fatalf("same key returned different nodes: %v vs %v", first, second)
	}
	if len(first) != 3 {
		t.Fatalf("expected 3 distinct nodes, got %d: %v", len(first), first)
	}
}

func TestDistribution_IsReasonablyEven(t *testing.T) {
	r := New()
	nodeIDs := []string{"node-a", "node-b", "node-c", "node-d", "node-e"}
	for _, n := range nodeIDs {
		r.AddNode(n)
	}

	counts := make(map[string]int)
	const numKeys = 100_000
	for i := 0; i < numKeys; i++ {
		nodes := r.GetNodesForKey(fmt.Sprintf("key-%d", i), 1)
		counts[nodes[0]]++
	}

	expected := numKeys / len(nodeIDs)
	for node, count := range counts {
		deviation := float64(count-expected) / float64(expected)
		if deviation < -0.15 || deviation > 0.15 { // allow 15% skew
			t.Errorf("node %s got %d keys, expected ~%d (deviation %.1f%%)",
				node, count, expected, deviation*100)
		}
	}
}

// This is the test that matters most for the portfolio story: prove that
// removing one node out of five only remaps roughly 1/5 of keys, not all of them.
func TestRemoveNode_MinimalRemapping(t *testing.T) {
	r := New()
	nodeIDs := []string{"node-a", "node-b", "node-c", "node-d", "node-e"}
	for _, n := range nodeIDs {
		r.AddNode(n)
	}

	const numKeys = 10_000
	before := make(map[string]string, numKeys)
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("key-%d", i)
		before[key] = r.GetNodesForKey(key, 1)[0]
	}

	r.RemoveNode("node-c")

	moved := 0
	for key, oldNode := range before {
		newNode := r.GetNodesForKey(key, 1)[0]
		if newNode != oldNode {
			moved++
		}
	}

	fraction := float64(moved) / float64(numKeys)
	t.Logf("removed 1 of 5 nodes: %.1f%% of keys remapped (%d/%d)",
		fraction*100, moved, numKeys)

	// With naive hash % N, this would be ~80-100%. Consistent hashing
	// should keep it close to 1/5 = 20%.
	if fraction > 0.30 {
		t.Errorf("too much remapping: %.1f%%, expected close to 20%%", fraction*100)
	}
}
