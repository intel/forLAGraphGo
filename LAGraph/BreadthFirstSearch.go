package LAGraph

import (
	"errors"
	"github.com/intel/forGraphBLASGo/GrB"
	"math"
)

func (G *Graph[D]) BreadthFirstSearch(src int, computeLevel, computeParent bool) (level, parent GrB.Vector[int], err error) {
	defer GrB.CheckErrors(&err)

	GrB.OK(G.Check())

	A := G.A
	n, err := A.Nrows()
	GrB.OK(err)

	if src >= n {
		err = errors.New("invalid source node")
		return
	}

	if n > math.MaxInt32 {
		l, p := breadthFirstSearchDispatch[D, int64](G, n, src, computeLevel, computeParent)
		level = GrB.VectorView[int, int64](l)
		parent = GrB.VectorView[int, int64](p)
		return
	}
	l, p := breadthFirstSearchDispatch[D, int32](G, n, src, computeLevel, computeParent)
	level = GrB.VectorView[int, int32](l)
	parent = GrB.VectorView[int, int32](p)
	return
}

func breadthFirstSearchDispatch[D GrB.Predefined, Int int32 | int64](G *Graph[D], n, src int, computeLevel, computeParent bool) (level, parent GrB.Vector[Int]) {
	if computeParent {
		return breadthFirstSearch[D, Int, Int](G, n, src, computeLevel, computeParent, GrB.AnySecondi[Int]())
	}
	return breadthFirstSearch[D, Int, bool](G, n, src, computeLevel, computeParent, GrB.AnyOneb[bool]())
}

func breadthFirstSearch[D GrB.Predefined, Int int32 | int64, Q int32 | int64 | bool](G *Graph[D], n, src int, computeLevel, computeParent bool, semiring GrB.Semiring[Q, Q, Q]) (level, parent GrB.Vector[Int]) {
	try := func(f func() error) {
		GrB.OK(f())
	}

	A := G.A

	nvals, err := A.Nvals()
	GrB.OK(err)

	Degree := G.OutDegree

	var AT GrB.Matrix[D]
	if G.Kind == AdjacencyUndirected || (G.Kind == AdjacencyDirected && G.IsSymmetricStructure == True) {
		AT = G.A
	} else {
		AT = G.AT
	}

	pushPull := Degree.Valid() && AT.Valid()

	var pi GrB.Vector[Int]
	var q GrB.Vector[Q]
	if computeParent {
		pi, err = GrB.VectorNew[Int](n)
		GrB.OK(err)
		GrB.OK(pi.SetSparsityControl(GrB.Bitmap + GrB.Full))
		GrB.OK(pi.SetElement(Int(src), src))

		q, err = GrB.VectorNew[Q](n)
		GrB.OK(err)
		defer try(q.Free)
		GrB.OK(GrB.VectorView[Int, Q](q).SetElement(Int(src), src))
	} else {
		q, err = GrB.VectorNew[Q](n)
		GrB.OK(err)
		defer try(q.Free)
		GrB.OK(GrB.VectorView[bool, Q](q).SetElement(true, src))
	}

	var v GrB.Vector[Int]
	if computeLevel {
		v, err = GrB.VectorNew[Int](n)
		GrB.OK(err)
		GrB.OK(v.SetSparsityControl(GrB.Bitmap + GrB.Full))
		GrB.OK(v.SetElement(Int(0), src))
	}

	w, err := GrB.VectorNew[int](n)
	GrB.OK(err)
	defer try(w.Free)

	nq := 1
	const (
		alpha = 8
		beta1 = 8
		beta2 = 512
	)
	nOverBeta1 := int(float64(n) / beta1)
	nOverBeta2 := int(float64(n) / beta2)

	doPush := true
	lastNq := 0
	edgesUnexplored := nvals
	anyPull := false

	var mask *GrB.Vector[bool]
	if computeParent {
		mask = pi.AsMask()
	} else {
		mask = v.AsMask()
	}

	for nvisited, k := 1, 1; nvisited < n; nvisited, k = nvisited+nq, k+1 {
		if pushPull {
			if doPush {
				growing := nq > lastNq
				switchToPull := false
				if edgesUnexplored < n {
					pushPull = false
				} else if anyPull {
					switchToPull = growing && nq > nOverBeta1
				} else {
					GrB.OK(w.Assign(q.AsMask(), nil, Degree, GrB.All(n), GrB.DescRS))
					edgesInFrontier, e := w.Reduce(GrB.PlusMonoid[int](), nil)
					GrB.OK(e)
					edgesUnexplored -= edgesInFrontier
					switchToPull = growing && edgesInFrontier > int(float64(edgesUnexplored)/alpha)
				}
				if switchToPull {
					doPush = false
				}
			} else {
				shrinking := nq < lastNq
				if shrinking && nq <= nOverBeta2 {
					doPush = true
				}
			}
			anyPull = anyPull || !doPush
		}

		var sparsity GrB.Sparsity
		if doPush {
			sparsity = GrB.Sparse
		} else {
			sparsity = GrB.Bitmap
		}
		GrB.OK(q.SetSparsityControl(sparsity))

		if doPush {
			GrB.OK(q.VxM(mask, nil, semiring, q, GrB.MatrixView[Q, D](A), GrB.DescRSC))
		} else {
			GrB.OK(q.MxV(mask, nil, semiring, GrB.MatrixView[Q, D](AT), q, GrB.DescRSC))
		}

		lastNq = nq
		nq, err = q.Nvals()
		GrB.OK(err)
		if nq == 0 {
			break
		}

		if computeParent {
			GrB.OK(pi.Assign(q.AsMask(), nil, GrB.VectorView[Int, Q](q), GrB.All(n), GrB.DescS))
		}
		if computeLevel {
			GrB.OK(v.AssignConstant(q.AsMask(), nil, Int(k), GrB.All(n), GrB.DescS))
		}
	}

	if computeParent {
		parent = pi
	}
	if computeLevel {
		level = v
	}
	return
}
