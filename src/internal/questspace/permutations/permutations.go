// Package permutations is used to validate incoming order changes in tasks and task groups
package permutations

import "github.com/spkg/ptr"

type OrderChange struct {
	Prev int
	Next int
}

// FindTreesAndCycles finds all sequences of reordering and groups them depending
// on whether sequence ends with nil-node (e.g. tree) or has no end (e.g. cycle)
//
// Panics:
//  1. When `orderChanges` contain vertex greater or equal to `n`
//  2. When `orderChanges` contain vertices with node ranks greater of 1
func FindTreesAndCycles(orderChanges []OrderChange, n int) (trees, cycles [][]int) {
	componentMap := findConnectedComponents(n, orderChanges)
	graph := make([]*int, n)
	for _, edge := range orderChanges {
		graph[edge.Prev] = ptr.Int(edge.Next)
	}

	for _, comp := range componentMap {
		if len(comp) == 1 {
			continue
		}
		sorted, isTree := checkCycleOrTopoSort(graph, orderChanges, comp, n)
		if isTree {
			trees = append(trees, sorted)
		} else {
			cycles = append(cycles, sorted)
		}
	}
	return trees, cycles
}

func checkCycleOrTopoSort(graph []*int, edges []OrderChange, nodes []int, n int) ([]int, bool) {
	inDegree := make([]int, n)

	for _, edge := range edges {
		inDegree[edge.Next]++
	}

	topoSorted := make([]int, 0, n)
	var zeroInDegree []int
	for _, node := range nodes {
		if inDegree[node] == 0 {
			zeroInDegree = append(zeroInDegree, node)
		}
	}

	// this operation will panic on broken invariants
	if len(zeroInDegree) == 0 {
		orderedCycle := make([]int, len(nodes))
		first := nodes[0]
		for i := 0; i < len(nodes); i++ {
			orderedCycle[i] = first
			first = *graph[first]
		}
		return orderedCycle, false
	}

	for len(zeroInDegree) > 0 {
		node := zeroInDegree[0]
		zeroInDegree = zeroInDegree[1:]
		topoSorted = append(topoSorted, node)

		next := graph[node]
		if next == nil {
			continue
		}
		inDegree[*next]--
		if inDegree[*next] == 0 {
			zeroInDegree = append(zeroInDegree, *next)
		}
	}

	return topoSorted, true
}

func findConnectedComponents(n int, edges []OrderChange) map[int][]int {
	nodeParents := make([]int, n)
	for i := range nodeParents {
		nodeParents[i] = i
	}
	for _, edge := range edges {
		nodeParents[merge(nodeParents, edge.Prev)] = merge(nodeParents, edge.Next)
	}
	compCount := 0
	for i, p := range nodeParents {
		if i == p {
			compCount++
		}
		nodeParents[i] = merge(nodeParents, p)
	}
	componentsMap := make(map[int][]int, compCount)
	for i, p := range nodeParents {
		componentsMap[p] = append(componentsMap[p], i)
	}
	return componentsMap
}

func merge(parent []int, x int) int {
	for parent[x] != x {
		parent[x] = parent[parent[x]]
		x = parent[x]
	}
	return x
}
