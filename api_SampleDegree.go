package forLAGraphGo

import (
	GrB "github.com/intel/forGraphBLASGo"
	"sort"
)

func (G *Graph[T]) SampleDegree(byRow bool, nSamples int, seed uint64) (sampleMean, sampleMedian float64) {
	if nSamples < 1 {
		nSamples = 1
	}
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
	samples := make([]int, nSamples)
	n, err := Degree.Size()
	try(err)
	dsum := 0
	for k := 0; k < nSamples; k++ {
		result := Random60(&seed)
		i := int(result % uint64(n))
		d, err := Degree.ExtractElement(i)
		try(err)
		samples[k] = d
		dsum += d
	}
	sampleMean = float64(dsum) / float64(nSamples)
	sort.Ints(samples)
	sampleMedian = float64(samples[nSamples/2])
	return
}
