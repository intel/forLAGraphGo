package forLAGraphGo

import (
	GrB "github.com/intel/forGraphBLASGo"
)

type TriangleCountPresort int

const (
	NoSort                 TriangleCountPresort = 0
	SortByDegreeAscending  TriangleCountPresort = 1
	SortByDegreeDescending TriangleCountPresort = -1
	AutoSelectSort         TriangleCountPresort = 2
)

var AllTriangleCountPresorts = []TriangleCountPresort{NoSort, SortByDegreeAscending, SortByDegreeDescending, AutoSelectSort}

type TriangleCountMethod int

const (
	minitri TriangleCountMethod = iota
	Burkhardt
	Cohen
	Sandia
	Sandia2
	SandiaDot
	SandiaDot2
)

var AllTriangleCountMethods = []TriangleCountMethod{Burkhardt, Cohen, Sandia, Sandia2, SandiaDot, SandiaDot2}

func TriangleCount[T GrB.Number](G *Graph[T]) (ntriangles int) {
	G.PropertyASymmetricStructure()
	G.PropertyRowDegree()
	G.PropertyNDiag()
	method := SandiaDot
	presort := AutoSelectSort
	return TriangleCountMethods(G, method, &presort)
}

func tricountPrep(A *GrB.Matrix[bool], l, u bool) (L, U *GrB.Matrix[bool]) {
	n, err := A.NRows()
	try(err)
	if l {
		ll, err := GrB.MatrixNew[bool](n, n)
		try(err)
		try(GrB.MatrixSelect(ll, nil, nil, GrB.TriL[bool], A, -1, nil))
		L = ll
	}
	if u {
		uu, err := GrB.MatrixNew[bool](n, n)
		try(err)
		try(GrB.MatrixSelect(uu, nil, nil, GrB.TriU[bool], A, 1, nil))
		U = uu
	}
	return
}

func TriangleCountMethods[T GrB.Number](G *Graph[T], method TriangleCountMethod, presort *TriangleCountPresort) (ntriangles int) {
	G.Check()
	if G.NDiag != 0 {
		panic(-104)
	}
	if !(G.Kind == AdjacencyUndirected || (G.Kind == AdjacencyDirected && G.AStructureIsSymmetric == True)) {
		panic("G.A must be known to be symmetric")
	}
	A := G.A
	Degree := G.RowDegree
	autosort := *presort == AutoSelectSort
	if autosort && method >= Sandia {
		if Degree == nil {
			panic("G.RowDegree must be defined")
		}
	}
	n, err := G.A.NRows()
	try(err)
	C, err := GrB.MatrixNew[int](n, n)
	try(err)
	semiring := PlusOne[int, bool, bool]
	monoid := GrB.PlusMonoid[int]
	if autosort {
		*presort = 0
		if method >= Sandia {
			const nSamples = 1000
			nvals, err := A.NVals()
			try(err)
			if n > nSamples && (float64(nvals)/float64(n) >= 10) {
				mean, median := G.SampleDegree(true, nSamples, uint64(n))
				if mean > 4*median {
					switch method {
					case Sandia:
						*presort = SortByDegreeAscending
					case SandiaDot:
						*presort = SortByDegreeDescending
					case Sandia2:
						*presort = SortByDegreeAscending
					case SandiaDot2:
						*presort = SortByDegreeDescending
					}
				}
			}
		}
	}

	B, err := GrB.MatrixNew[bool](n, n)
	try(err)
	try(GrB.MatrixApply(B, nil, nil, func(i T) bool { return i != 0 }, A, nil))

	if *presort != NoSort {
		P := G.SortByDegree(true, *presort > 0)
		try(GrB.MatrixExtract(B, nil, nil, B, P, P, nil))
	}

	switch method {
	case Burkhardt:
		try(B.Wait(GrB.Materialize))
		try(GrB.MxM(C, B, nil, semiring, B, B, GrB.DescS))
		try(GrB.MatrixReduce(&ntriangles, nil, monoid, C, nil))
		ntriangles /= 6
	case Cohen:
		try(B.Wait(GrB.Materialize))
		L, U := tricountPrep(B, true, true)
		try(GrB.MxM(C, B, nil, semiring, L, U, GrB.DescS))
		try(GrB.MatrixReduce(&ntriangles, nil, monoid, C, nil))
		ntriangles /= 2
	case Sandia:
		L, _ := tricountPrep(B, true, false)
		try(L.Wait(GrB.Materialize))
		try(GrB.MxM(C, L, nil, semiring, L, L, GrB.DescS))
		try(GrB.MatrixReduce(&ntriangles, nil, monoid, C, nil))
	case Sandia2:
		_, U := tricountPrep(B, false, true)
		try(U.Wait(GrB.Materialize))
		try(GrB.MxM(C, U, nil, semiring, U, U, GrB.DescS))
		try(GrB.MatrixReduce(&ntriangles, nil, monoid, C, nil))
	case SandiaDot:
		try(B.Wait(GrB.Materialize))
		L, U := tricountPrep(B, true, true)
		try(GrB.MxM(C, L, nil, semiring, L, U, GrB.DescST1))
		try(GrB.MatrixReduce(&ntriangles, nil, monoid, C, nil))
	case SandiaDot2:
		try(B.Wait(GrB.Materialize))
		L, U := tricountPrep(B, true, true)
		try(GrB.MxM(C, U, nil, semiring, U, L, GrB.DescST1))
		try(GrB.MatrixReduce(&ntriangles, nil, monoid, C, nil))
	default:
		panic("unknown / unimplemented TriangleCount method")
	}
	return
}
