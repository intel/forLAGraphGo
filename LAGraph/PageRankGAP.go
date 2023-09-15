package LAGraph

import (
	"errors"
	"github.com/intel/forGraphBLASGo/GrB"
)

func (G *Graph[D]) PageRankGAP(damping, tolerance float32, iterMax int) (centrality GrB.Vector[float32], iterations int, err error) {
	defer GrB.CheckErrors(&err)
	try := func(f func() error) {
		GrB.OK(f())
	}
	GrB.OK(G.Check())
	var AT GrB.Matrix[D]
	if G.Kind == AdjacencyUndirected || G.IsSymmetricStructure == True {
		AT = G.A
	} else {
		AT = G.AT
		if !AT.Valid() {
			err = errors.New("G.AT is required")
			return
		}
	}
	dOut := G.OutDegree
	if !dOut.Valid() {
		err = errors.New("G.OutDegree is required")
		return
	}
	n, err := AT.Nrows()
	GrB.OK(err)
	scaledDamping := (1 - damping) / float32(n)
	teleport := scaledDamping
	rdiff := float32(1)

	t, err := GrB.VectorNew[float32](n)
	GrB.OK(err)
	defer try(t.Free)
	r, err := GrB.VectorNew[float32](n)
	GrB.OK(err)
	defer func() {
		if err != nil {
			_ = r.Free()
		}
	}()
	w, err := GrB.VectorNew[float32](n)
	GrB.OK(err)
	defer try(w.Free)
	GrB.OK(GrB.VectorAssignConstant(r, nil, nil, 1/float32(n), GrB.All(n), nil))

	d, err := GrB.VectorNew[float32](n)
	GrB.OK(err)
	defer try(d.Free)
	GrB.OK(GrB.VectorApplyBinaryOp2nd(d, nil, nil, GrB.Div[float32](), GrB.VectorView[float32, int](dOut), damping, nil))
	dmin := 1 / damping
	d1, err := GrB.VectorNew[float32](n)
	GrB.OK(err)
	defer try(d1.Free)
	GrB.OK(GrB.VectorAssignConstant(d1, nil, nil, dmin, GrB.All(n), nil))
	GrB.OK(GrB.VectorEWiseAddBinaryOp(d, nil, nil, GrB.Max[float32](), d1, d, nil))
	GrB.OK(d1.Free())

	for iterations = 0; iterations < iterMax && rdiff > tolerance; iterations++ {
		t, r = r, t
		GrB.OK(GrB.VectorEWiseMultBinaryOp(w, nil, nil, GrB.Div[float32](), t, d, nil))
		GrB.OK(GrB.VectorAssignConstant(r, nil, nil, teleport, GrB.All(n), nil))
		plus := GrB.Plus[float32]()
		GrB.OK(GrB.MxV(r, nil, &plus, PlusSecond[float32](), GrB.MatrixView[float32, D](AT), w, nil))
		minus := GrB.Minus[float32]()
		GrB.OK(GrB.VectorAssign(t, nil, &minus, r, GrB.All(n), nil))
		GrB.OK(GrB.VectorApply(t, nil, nil, GrB.Abs[float32](), t, nil))
		rdiff, err = GrB.VectorReduce(GrB.PlusMonoid[float32](), t, nil)
		GrB.OK(err)
	}

	centrality = r
	return
}
