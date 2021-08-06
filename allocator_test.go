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
	counts := make(map[Bin]int)
	for _, bin := range placement {
		counts[bin] += 1
	}

	require.Len(t, counts, 5)
	for _, count := range counts {
		require.Equal(t, 6, count)
	}
}

func TestChurn(t *testing.T) {
	const items = 5
	const bins = 3
	a := NewAllocator(items, bins, 1)
	// a.Options.DisableEvenDistribution = true // really slows things down otherwise, even when solvable

	placement, ok := a.Allocate()
	require.True(t, ok)
	require.Len(t, placement, items)
}
