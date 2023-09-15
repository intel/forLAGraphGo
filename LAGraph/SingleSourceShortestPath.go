package LAGraph

import (
	"errors"
	"github.com/intel/forGraphBLASGo/GrB"
)

type SingleSourceShortestPathDomains interface {
	int | int32 | int64 | uint | uint32 | uint64 | float32 | float64
}

func SingleSourceShortestPath[D SingleSourceShortestPathDomains](G *Graph[D], source int, delta D) (pathLength GrB.Vector[D], err error) {
	defer GrB.CheckErrors(&err)
	try := func(f func() error) {
		GrB.OK(f())
	}
	GrB.OK(G.Check())
	A := G.A
	n, err := A.Nrows()
	GrB.OK(err)
	if source >= n {
		err = errors.New("invalid source node")
		return
	}
	t, err := GrB.VectorNew[D](n)
	GrB.OK(err)
	defer func() {
		if err != nil {
			_ = t.Free()
		}
	}()
	tmasked, err := GrB.VectorNew[D](n)
	GrB.OK(err)
	defer try(tmasked.Free)
	tReq, err := GrB.VectorNew[D](n)
	GrB.OK(err)
	defer try(tReq.Free)
	Empty, err := GrB.VectorNew[bool](n)
	GrB.OK(err)
	defer try(Empty.Free)
	tless, err := GrB.VectorNew[bool](n)
	GrB.OK(err)
	defer try(tless.Free)
	s, err := GrB.VectorNew[bool](n)
	GrB.OK(err)
	defer try(s.Free)
	reach, err := GrB.VectorNew[bool](n)
	GrB.OK(err)
	defer try(reach.Free)

	GrB.OK(t.SetSparsityControl(GrB.Bitmap))
	GrB.OK(tmasked.SetSparsityControl(GrB.Sparse))
	GrB.OK(tReq.SetSparsityControl(GrB.Sparse))
	GrB.OK(tless.SetSparsityControl(GrB.Sparse))
	GrB.OK(s.SetSparsityControl(GrB.Sparse))
	GrB.OK(reach.SetSparsityControl(GrB.Bitmap))

	ne := GrB.Valuene[bool]()
	le := GrB.Valuele[D]()
	ge := GrB.Valuege[D]()
	lt := GrB.Valuelt[D]()
	gt := GrB.Valuegt[D]()
	lessThan := GrB.Lt[D]()
	minPlus := GrB.MinPlusSemiring[D]()
	GrB.OK(GrB.VectorAssignConstant(t, nil, nil, GrB.Maximum[D](), GrB.All(n), nil))
	negativeEdgeWeights := true
	var x D
	switch any(x).(type) {
	case uint, uint32, uint64:
		negativeEdgeWeights = false
	}

	if negativeEdgeWeights {
		if G.EMin.Valid() && (G.EMinState == Value || G.EMinState == Bound) {
			emin, _, e := GrB.ScalarView[float64, D](G.EMin).ExtractElement()
			GrB.OK(e)
			negativeEdgeWeights = emin < 0
		}
	}

	GrB.OK(t.SetElement(0, source))
	GrB.OK(reach.SetElement(true, source))
	GrB.OK(s.SetElement(true, source))

	AL, err := GrB.MatrixNew[D](n, n)
	GrB.OK(err)
	defer try(AL.Free)
	GrB.OK(GrB.MatrixSelect(AL, nil, nil, le, A, delta, nil))
	GrB.OK(AL.Wait(GrB.Materialize))

	AH, err := GrB.MatrixNew[D](n, n)
	GrB.OK(err)
	defer try(AH.Free)
	GrB.OK(GrB.MatrixSelect(AH, nil, nil, gt, A, delta, nil))
	GrB.OK(AH.Wait(GrB.Materialize))

	for step := 0; ; step++ {
		uBound := D(step+1) * delta
		GrB.OK(tmasked.Clear())
		GrB.OK(GrB.VectorAssign(tmasked, &reach, nil, t, GrB.All(n), nil))
		GrB.OK(GrB.VectorSelect(tmasked, nil, nil, lt, tmasked, uBound, nil))
		tmaskedNvals, e := tmasked.Nvals()
		GrB.OK(e)
		for tmaskedNvals > 0 {
			GrB.OK(GrB.VxM(tReq, nil, nil, minPlus, tmasked, AL, nil))
			GrB.OK(GrB.VectorAssignConstant(s, tmasked.AsMask(), nil, true, GrB.All(n), GrB.DescS))

			tReqNvals, e := tReq.Nvals()
			GrB.OK(e)
			if tReqNvals == 0 {
				break
			}

			GrB.OK(GrB.VectorEWiseMultBinaryOp(tless, nil, nil, lessThan, tReq, t, nil))

			GrB.OK(GrB.VectorSelect(tless, nil, nil, ne, tless, false, nil))
			tLessNvals, e := tless.Nvals()
			GrB.OK(e)
			if tLessNvals == 0 {
				break
			}

			GrB.OK(GrB.VectorAssignConstant(reach, &tless, nil, true, GrB.All(n), GrB.DescS))

			GrB.OK(tmasked.Clear())
			GrB.OK(GrB.VectorSelect(tmasked, &tless, nil, lt, tReq, uBound, GrB.DescS))

			if negativeEdgeWeights {
				GrB.OK(GrB.VectorSelect(tmasked, nil, nil, ge, tmasked, D(step)*delta, nil))
			}

			GrB.OK(GrB.VectorAssign(t, &tless, nil, tReq, GrB.All(n), GrB.DescS))
			tmaskedNvals, err = tmasked.Nvals()
			GrB.OK(err)
		}

		GrB.OK(tmasked.Clear())
		GrB.OK(GrB.VectorAssign(tmasked, &s, nil, t, GrB.All(n), GrB.DescS))

		GrB.OK(GrB.VxM(tReq, nil, nil, minPlus, tmasked, AH, nil))
		GrB.OK(GrB.VectorEWiseMultBinaryOp(tless, nil, nil, lessThan, tReq, t, nil))
		GrB.OK(GrB.VectorAssign(t, &tless, nil, tReq, GrB.All(n), nil))

		GrB.OK(GrB.VectorAssignConstant(reach, &tless, nil, true, GrB.All(n), nil))

		GrB.OK(GrB.VectorAssign(reach, &s, nil, Empty, GrB.All(n), GrB.DescS))
		nreach, e := reach.Nvals()
		GrB.OK(e)
		if nreach == 0 {
			break
		}
		GrB.OK(s.Clear())
	}
	pathLength = t
	return
}
