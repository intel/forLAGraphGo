package LAGraph

import (
	"errors"
	"github.com/intel/forGraphBLASGo/GrB"
	"math/rand"
)

type TriangleCountMethod int

const (
	TriangleCountAutoMethod TriangleCountMethod = iota
	TriangleCountBurkhardt
	TriangleCountCohen
	TriangleCountSandiaLL
	TriangleCountSandiaUU
	TriangleCountSandiaLUT
	TriangleCountSandiaULT
)

func (m TriangleCountMethod) String() string {
	switch m {
	case TriangleCountAutoMethod:
		return "auto"
	case TriangleCountBurkhardt:
		return "Burkhardt"
	case TriangleCountCohen:
		return "Cohen"
	case TriangleCountSandiaLL:
		return "Sandia LL"
	case TriangleCountSandiaUU:
		return "Sandia UU"
	case TriangleCountSandiaLUT:
		return "Sandia LUT"
	case TriangleCountSandiaULT:
		return "Sandia ULT"
	default:
		panic("invalid triangle count method")
	}
}

type TriangleCountPresort int

const (
	TriangleCountNoSort     TriangleCountPresort = 2
	TriangleCountAscending  TriangleCountPresort = 1
	TriangleCountDescending TriangleCountPresort = -1
	TriangleCountAutoSort   TriangleCountPresort = 0
)

func (s TriangleCountPresort) String() string {
	switch s {
	case TriangleCountNoSort:
		return "no sort"
	case TriangleCountAscending:
		return "sort by degree, ascending"
	case TriangleCountDescending:
		return "sort by degree, descending"
	case TriangleCountAutoSort:
		return "auto"
	default:
		panic("invalid triangle count presort")
	}
}

func (G *Graph[D]) TriangleCount() (ntriangles int, err error) {
	try := func(f func() error) {
		if err != nil {
			return
		}
		err = f()
	}
	try(G.CachedIsSymmetricStructure)
	try(G.CachedOutDegree)
	try(G.CachedNSelfEdges)
	ntriangles, _, _, err = G.TriangleCountMethods(TriangleCountAutoMethod, TriangleCountAutoSort)
	return
}

func tricountPrep(A GrB.Matrix[bool], l, u bool) (L, U GrB.Matrix[bool], err error) {
	defer GrB.CheckErrors(&err)

	n, err := A.Nrows()
	GrB.OK(err)
	if l {
		L, err = GrB.MatrixNew[bool](n, n)
		GrB.OK(err)
		defer func() {
			if err != nil {
				_ = L.Free()
			}
		}()
		GrB.OK(GrB.MatrixSelect(L, nil, nil, GrB.Tril[bool](), A, -1, nil))
		GrB.OK(L.Wait(GrB.Materialize))
	}
	if u {
		U, err = GrB.MatrixNew[bool](n, n)
		GrB.OK(err)
		defer func() {
			if err != nil {
				_ = U.Free()
			}
		}()
		GrB.OK(GrB.MatrixSelect(U, nil, nil, GrB.Triu[bool](), A, 1, nil))
		GrB.OK(U.Wait(GrB.Materialize))
	}
	return
}

func (G *Graph[D]) TriangleCountMethods(inMethod TriangleCountMethod, inPresort TriangleCountPresort) (ntriangles int, method TriangleCountMethod, presort TriangleCountPresort, err error) {
	defer GrB.CheckErrors(&err)
	try := func(f func() error) {
		GrB.OK(f())
	}
	method = inMethod
	presort = inPresort
	GrB.OK(G.Check())
	if G.NSelfEdges != 0 {
		err = errors.New("no self edges allowed")
		return
	}
	if !(G.Kind == AdjacencyUndirected || (G.Kind == AdjacencyDirected && G.IsSymmetricStructure == True)) {
		err = errors.New("G.A must be known to be symmetric")
		return
	}
	if method == TriangleCountAutoMethod {
		method = TriangleCountSandiaLUT
	}
	var canUsePresort bool
	switch method {
	case TriangleCountSandiaLL, TriangleCountSandiaUU, TriangleCountSandiaLUT, TriangleCountSandiaULT:
		canUsePresort = true
	}
	A := G.A
	Degree := G.OutDegree
	autosort := presort == TriangleCountAutoSort
	if autosort && canUsePresort {
		if !Degree.Valid() {
			err = errors.New("G.OutDegree must be defined")
			return
		}
	}
	n, err := G.A.Nrows()
	GrB.OK(err)
	C, err := GrB.MatrixNew[int](n, n)
	GrB.OK(err)
	defer try(C.Free)
	semiring := GrB.PlusOneb[int]()
	monoid := GrB.PlusMonoid[int]()
	if !canUsePresort {
		presort = TriangleCountNoSort
	} else if autosort {
		presort = TriangleCountNoSort
		if canUsePresort {
			const nSamples = 1000
			nvals, e := A.Nvals()
			GrB.OK(e)
			if n > nSamples && (float64(nvals)/float64(n) >= 10) {
				mean, median, e := G.SampleDegree(true, nSamples, rand.New(rand.NewSource(rand.Int63())))
				GrB.OK(e)
				if mean > 4*median {
					switch method {
					case TriangleCountSandiaLL:
						presort = TriangleCountAscending
					case TriangleCountSandiaUU:
						presort = TriangleCountDescending
					case TriangleCountSandiaLUT:
						presort = TriangleCountAscending
					case TriangleCountSandiaULT:
						presort = TriangleCountDescending
					}
				}
			}
		}
	}

	if presort != TriangleCountNoSort {
		P, e := G.SortByDegree(true, presort == TriangleCountAscending)
		GrB.OK(e)
		T, e := GrB.MatrixNew[bool](n, n)
		GrB.OK(e)
		defer try(T.Free)
		GrB.OK(T.Extract(nil, nil, GrB.MatrixView[bool, D](A), P, P, nil))
		A = GrB.MatrixView[D, bool](T)
	}

	switch method {
	case TriangleCountBurkhardt:
		GrB.OK(C.MxM(A.AsMask(), nil, semiring, GrB.MatrixView[int, D](A), GrB.MatrixView[int, D](A), GrB.DescS))
		ntriangles, err = C.Reduce(monoid, nil)
		GrB.OK(err)
		ntriangles /= 6
	case TriangleCountCohen:
		L, U, e := tricountPrep(GrB.MatrixView[bool, D](A), true, true)
		GrB.OK(e)
		defer try(L.Free)
		defer try(U.Free)
		GrB.OK(C.MxM(A.AsMask(), nil, semiring, GrB.MatrixView[int, bool](L), GrB.MatrixView[int, bool](U), GrB.DescS))
		ntriangles, err = C.Reduce(monoid, nil)
		GrB.OK(err)
		ntriangles /= 2
	case TriangleCountSandiaLL:
		L, _, e := tricountPrep(GrB.MatrixView[bool, D](A), true, false)
		GrB.OK(e)
		defer try(L.Free)
		GrB.OK(C.MxM(L.AsMask(), nil, semiring, GrB.MatrixView[int, bool](L), GrB.MatrixView[int, bool](L), GrB.DescS))
		ntriangles, err = C.Reduce(monoid, nil)
	case TriangleCountSandiaUU:
		_, U, e := tricountPrep(GrB.MatrixView[bool, D](A), false, true)
		GrB.OK(e)
		defer try(U.Free)
		GrB.OK(C.MxM(U.AsMask(), nil, semiring, GrB.MatrixView[int, bool](U), GrB.MatrixView[int, bool](U), GrB.DescS))
		ntriangles, err = C.Reduce(monoid, nil)
	case TriangleCountSandiaLUT:
		L, U, e := tricountPrep(GrB.MatrixView[bool, D](A), true, true)
		GrB.OK(e)
		defer try(L.Free)
		defer try(U.Free)
		GrB.OK(C.MxM(L.AsMask(), nil, semiring, GrB.MatrixView[int, bool](L), GrB.MatrixView[int, bool](U), GrB.DescST1))
		ntriangles, err = C.Reduce(monoid, nil)
	case TriangleCountSandiaULT:
		L, U, e := tricountPrep(GrB.MatrixView[bool, D](A), true, true)
		GrB.OK(e)
		defer try(L.Free)
		defer try(U.Free)
		GrB.OK(C.MxM(U.AsMask(), nil, semiring, GrB.MatrixView[int, bool](U), GrB.MatrixView[int, bool](L), GrB.DescST1))
		ntriangles, err = C.Reduce(monoid, nil)
	default:
		panic("unreachable code")
	}
	return
}
