package forLAGraphGo

import (
	"github.com/intel/forGoParallel/parallel"
	GrB "github.com/intel/forGraphBLASGo"
)

func (G *Graph[T]) SortByDegree(byRow, ascending bool) []int {
	G.Check()
	var Degree *GrB.Vector[int]
	if G.Kind == AdjacencyUndirected || (G.Kind == AdjacencyDirected && G.AStructureIsSymmetric == True) {
		Degree = G.RowDegree
	} else if byRow {
		Degree = G.RowDegree
	} else {
		Degree = G.ColDegree
	}
	if Degree == nil {
		panic("degree property unknown")
	}
	n, err := Degree.Size()
	try(err)
	P := make([]int, n)
	D := make([]int, n)
	parallel.Range(0, n, func(low, high int) {
		for i := low; i < high; i++ {
			P[i] = i
		}
	})
	W0, W1, err := Degree.ExtractTuples()
	try(err)
	nvals := len(W0)
	if ascending {
		parallel.Range(0, nvals, func(low, high int) {
			for i := low; i < high; i++ {
				D[W0[i]] = W1[i]
			}
		})
	} else {
		parallel.Range(0, nvals, func(low, high int) {
			for i := low; i < high; i++ {
				D[W0[i]] = -W1[i]
			}
		})
	}
	twoSliceSort(D, P)
	return P
}
