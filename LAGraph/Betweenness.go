package LAGraph

import (
	"errors"
	"github.com/intel/forGraphBLASGo/GrB"
)

func (G *Graph[D]) Betweenness(sources []int) (centrality GrB.Vector[float64], err error) {
	defer GrB.CheckErrors(&err)

	try := func(f func() error) {
		GrB.OK(f())
	}

	GrB.OK(G.Check())

	A := GrB.MatrixView[float64, D](G.A)
	var AT GrB.Matrix[float64]
	if G.Kind == AdjacencyUndirected || G.IsSymmetricStructure == True {
		AT = A
	} else {
		AT = GrB.MatrixView[float64, D](G.AT)
		if !AT.Valid() {
			err = errors.New("G.AT is required")
			return
		}
	}
	n, err := A.Nrows()
	GrB.OK(err)
	ns := len(sources)

	paths, err := GrB.MatrixNew[float64](ns, n)
	GrB.OK(err)
	defer try(paths.Free)

	frontier, err := GrB.MatrixNew[float64](ns, n)
	GrB.OK(err)
	defer try(frontier.Free)

	GrB.OK(paths.SetSparsityControl(GrB.Bitmap + GrB.Full))

	for i, src := range sources {
		if src >= n {
			err = errors.New("invalid source node")
			return
		}
		GrB.OK(paths.SetElement(1, i, src))
		GrB.OK(frontier.SetElement(1, i, src))
	}

	GrB.OK(GrB.MxM(frontier, paths.AsMask(), nil, GrB.PlusFirst[float64](), frontier, A, GrB.DescRSC))

	S := make([]GrB.Matrix[bool], n+1)
	defer func() {
		for _, s := range S {
			GrB.OK(s.Free())
		}
	}()

	lastWasPull := false

	frontierSize, err := frontier.Nvals()
	GrB.OK(err)

	plus := GrB.Plus[float64]()
	depth := 0
	for ; frontierSize > 0 && depth < n; depth++ {
		S[depth], err = MatrixStructure(frontier)
		GrB.OK(err)
		GrB.OK(GrB.MatrixAssign(paths, nil, &plus, frontier, GrB.All(ns), GrB.All(n), nil))

		frontierDensity := float64(frontierSize) / float64(ns*n)
		var doPull bool
		if lastWasPull {
			doPull = frontierDensity > 0.06
		} else {
			doPull = frontierDensity > 0.10
		}

		if doPull {
			GrB.OK(frontier.SetSparsityControl(GrB.Bitmap))
			GrB.OK(GrB.MxM(frontier, paths.AsMask(), nil, GrB.PlusFirst[float64](), frontier, AT, GrB.DescRSCT1))
		} else {
			GrB.OK(frontier.SetSparsityControl(GrB.Sparse))
			GrB.OK(GrB.MxM(frontier, paths.AsMask(), nil, GrB.PlusFirst[float64](), frontier, A, GrB.DescRSC))
		}

		lastWasPull = doPull
		frontierSize, err = frontier.Nvals()
		GrB.OK(err)
	}

	GrB.OK(frontier.Free())

	bcUpdate, err := GrB.MatrixNew[float64](ns, n)
	GrB.OK(err)
	defer try(bcUpdate.Free)
	GrB.OK(GrB.MatrixAssignConstant(bcUpdate, nil, nil, 1, GrB.All(ns), GrB.All(n), nil))

	W, err := GrB.MatrixNew[float64](ns, n)
	GrB.OK(err)
	defer try(W.Free)

	for i := depth - 1; i > 0; i-- {
		GrB.OK(GrB.MatrixEWiseMultBinaryOp(W, &S[i], nil, GrB.Div[float64](), bcUpdate, paths, GrB.DescRS))
		wsize, e := W.Nvals()
		GrB.OK(e)
		ssize, e := S[i-1].Nvals()
		GrB.OK(e)
		wDensity := float64(wsize) / float64(ns*n)
		wToSRatio := float64(wsize) / float64(ssize)
		doPull := (wDensity > 0.1 && wToSRatio > 1) || (wDensity > 0.01 && wToSRatio > 10)

		if doPull {
			GrB.OK(W.SetSparsityControl(GrB.Bitmap))
			GrB.OK(GrB.MxM(W, &S[i-1], nil, GrB.PlusFirst[float64](), W, A, GrB.DescRST1))
		} else {
			GrB.OK(W.SetSparsityControl(GrB.Sparse))
			GrB.OK(GrB.MxM(W, &S[i-1], nil, GrB.PlusFirst[float64](), W, AT, GrB.DescRS))
		}

		GrB.OK(GrB.MatrixEWiseMultBinaryOp(bcUpdate, nil, &plus, GrB.Times[float64](), W, paths, nil))
	}

	centrality, err = GrB.VectorNew[float64](n)
	GrB.OK(err)
	GrB.OK(GrB.VectorAssignConstant(centrality, nil, nil, float64(-ns), GrB.All(n), nil))
	GrB.OK(GrB.MatrixReduceMonoid(centrality, nil, &plus, GrB.PlusMonoid[float64](), bcUpdate, GrB.DescT0))

	return
}
