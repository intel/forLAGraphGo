package LAGraph

import (
	"errors"
	"github.com/intel/forGraphBLASGo/GrB"
)

var ConvergenceFailure = errors.New("failed to converge")

func (G *Graph[D]) PageRank(damping, tolerance float32, iterMax int) (centrality GrB.Vector[float32], iterations int, err error) {
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
	dampingOverN := damping / float32(n)
	scaledDamping := (1 - damping) / float32(n)
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

	nvals, err := dOut.Nvals()
	GrB.OK(err)
	nsinks := n - nvals
	var sink GrB.Vector[bool]
	var rsink GrB.Vector[float32]
	if nsinks > 0 {
		sink, err = GrB.VectorNew[bool](n)
		GrB.OK(err)
		defer try(sink.Free)
		GrB.OK(GrB.VectorAssignConstant(sink, dOut.AsMask(), nil, true, GrB.All(n), GrB.DescSC))
		rsink, err = GrB.VectorNew[float32](n)
		GrB.OK(err)
		defer try(rsink.Free)
	}
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

	for iterations = 0; rdiff > tolerance; iterations++ {
		if iterations >= iterMax {
			err = ConvergenceFailure
			return
		}
		teleport := scaledDamping
		if nsinks > 0 {
			GrB.OK(rsink.Clear())
			GrB.OK(GrB.VectorAssign(rsink, &sink, nil, r, GrB.All(n), GrB.DescS))
			sumRsink, e := GrB.VectorReduce(GrB.PlusMonoid[float32](), rsink, nil)
			GrB.OK(e)
			teleport += dampingOverN * sumRsink
		}
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
