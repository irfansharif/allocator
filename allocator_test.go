package allocator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSingleBucket(t *testing.T) {
	a := NewAllocator(1, 1, 1)
	placement, ok := a.Allocate()
	require.True(t, ok)
	require.Len(t, placement, 1)
	for item, bin := range placement {
		t.Logf("placed %s in %s", item, bin)
	}
}

func TestInfeasible(t *testing.T) {
	a := NewAllocator(15, 1, 1)
	_, ok := a.Allocate()
	require.False(t, ok)
}

func TestMultipleBuckets(t *testing.T) {
	a := NewAllocator(30, 5, 1)
	placement, ok := a.Allocate()

	require.True(t, ok)
	require.Len(t, placement, 30)
	for item, bin := range placement {
		t.Logf("placed %s in %s", item, bin)
	}
}
