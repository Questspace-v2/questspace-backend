package permutations

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateLargeTestCase(nodes int, components int) []OrderChange {
	edges := make([]OrderChange, 0)
	nodesPerComponent := nodes / components

	for c := 0; c < components; c++ {
		// Generate a connected component
		startNode := c * nodesPerComponent
		for i := 0; i < nodesPerComponent-1; i++ {
			edges = append(edges, OrderChange{startNode + i, startNode + i + 1})
		}

		// Randomly connect nodes within the component to make it more complex
		for i := 0; i < nodesPerComponent/2; i++ {
			x := startNode + rand.Intn(nodesPerComponent)
			y := startNode + rand.Intn(nodesPerComponent)
			if x != y {
				edges = append(edges, OrderChange{x, y})
			}
		}
	}

	// Randomly connect some components
	for i := 0; i < components-1; i++ {
		x := i*nodesPerComponent + rand.Intn(nodesPerComponent)
		y := (i+1)*nodesPerComponent + rand.Intn(nodesPerComponent)
		edges = append(edges, OrderChange{x, y})
	}

	return edges
}

func TestFindTreesAndCycles(t *testing.T) {
	testCases := []struct {
		name           string
		changesets     []OrderChange
		targetLen      int
		expectedChains [][]int
		expectedCycles [][]int
		panics         bool
	}{
		{
			name: "basic",
			changesets: []OrderChange{
				{0, 1},
				{2, 4},
				{4, 3},
				{3, 2},
			},
			targetLen:      5,
			expectedChains: [][]int{{0, 1}},
			expectedCycles: [][]int{{2, 4, 3}},
		},
		{
			name: "big chain",
			changesets: []OrderChange{
				{3, 5},
				{0, 2},
				{4, 6},
				{2, 1},
				{1, 3},
				{5, 4},
			},
			targetLen:      7,
			expectedChains: [][]int{{0, 2, 1, 3, 5, 4, 6}},
			expectedCycles: nil,
		},
		{
			name: "big cycle",
			changesets: []OrderChange{
				{3, 5},
				{0, 2},
				{4, 6},
				{2, 1},
				{1, 3},
				{5, 4},
				{6, 0},
			},
			targetLen:      7,
			expectedChains: nil,
			expectedCycles: [][]int{{0, 2, 1, 3, 5, 4, 6}},
		},
		{
			name:       "panics on incorrect len",
			changesets: []OrderChange{{0, 256}},
			targetLen:  10,
			panics:     true,
		},
		{
			name: "panics on incorrect out ranks",
			changesets: []OrderChange{
				{1, 2},
				{2, 3},
				{3, 1},
				{1, 0},
			},
			targetLen: 4,
			panics:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.panics {
				chains, cycles := FindTreesAndCycles(tc.changesets, tc.targetLen)
				assert.Equal(t, tc.expectedChains, chains)
				assert.Equal(t, tc.expectedCycles, cycles)
			} else {
				require.Panics(t, func() {
					FindTreesAndCycles(tc.changesets, tc.targetLen)
				})
			}
		})
	}
}

func BenchmarkFindTreesAndCycles(b *testing.B) {
	// because test case builder does not guarantee invariants needed for
	// tested function, panics may occur
	//TODO(svayp11): Build test cases with max rank of node less or equal to 1
	defer func() {
		if smth := recover(); smth != nil {
			b.Log("panic happened", smth)
		}
	}()
	inputGraph := generateLargeTestCase(1000, 200)
	inputLen := 1000
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = FindTreesAndCycles(inputGraph, inputLen)
	}
}

func TestFindConnectedComponents(t *testing.T) {
	testCases := []struct {
		name           string
		edges          []OrderChange
		n              int
		expectedValues [][]int
	}{
		{
			name: "basic",
			edges: []OrderChange{
				{0, 1},
				{2, 1},
				{3, 4},
			},
			n:              5,
			expectedValues: [][]int{{0, 1, 2}, {3, 4}},
		},
		{
			name: "four components",
			edges: []OrderChange{
				{0, 1},
				{2, 4},
				{4, 3},
				{5, 6},
				{6, 7},
				{7, 10},
				{9, 5},
			},
			n:              11,
			expectedValues: [][]int{{0, 1}, {2, 3, 4}, {5, 6, 7, 9, 10}, {8}},
		},
		{
			name: "one cycle component",
			edges: []OrderChange{
				{0, 2},
				{2, 1},
				{1, 0},
			},
			n:              3,
			expectedValues: [][]int{{0, 1, 2}},
		},
		{
			name: "one chain component",
			edges: []OrderChange{
				{0, 2},
				{2, 1},
			},
			n:              3,
			expectedValues: [][]int{{0, 1, 2}},
		},
		{
			name:           "zero permutation components",
			n:              3,
			expectedValues: [][]int{{0}, {1}, {2}},
		},
		{
			name: "whatever",
			edges: []OrderChange{
				{0, 1},
				{2, 4},
				{4, 3},
				{3, 2},
			},
			n:              5,
			expectedValues: [][]int{{0, 1}, {2, 3, 4}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			comps := findConnectedComponents(tc.n, tc.edges)
			vals := make([][]int, 0, len(comps))
			for _, v := range comps {
				vals = append(vals, v)
			}
			assert.ElementsMatch(t, vals, tc.expectedValues)
		})
	}
}
func BenchmarkFindConnectedComponents(b *testing.B) {
	inputGraph := generateLargeTestCase(1000, 200)
	inputLen := 1000
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = findConnectedComponents(inputLen, inputGraph)
	}
}
