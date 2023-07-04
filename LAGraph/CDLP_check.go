package LAGraph

import (
	"github.com/intel/forGraphBLASGo/GrB"
)

func (G *Graph[D]) CDLPCheck(itermax int) (result GrB.Vector[int], err error) {
	defer GrB.CheckErrors(&err)
	try := func(f func() error) {
		GrB.OK(f())
	}

	A := G.A
	symmetric := G.Kind == AdjacencyUndirected ||
		(G.Kind == AdjacencyDirected &&
			G.IsSymmetricStructure == True)

	n, err := A.Nrows()
	GrB.OK(err)
	nz, err := A.Nvals()
	GrB.OK(err)
	var nnz int
	if !symmetric {
		nnz = 2 * nz
	} else {
		nnz = nz
	}

	S, err := GrB.MatrixNew[int](n, n)
	GrB.OK(err)
	GrB.OK(S.ApplyBinaryOp2nd(nil, nil, GrB.Oneb[int](), GrB.MatrixView[int, D](A), 0, nil))

	LP := GrB.MakeSystemSlice[int](n + 1)
	{
		LPs := LP.UnsafeSlice()
		for i := range LPs {
			LPs[i] = i
		}
	}
	LI := GrB.MakeSystemSlice[int](n)
	LX := GrB.MakeSystemSlice[int](n)
	{
		LIs := LI.UnsafeSlice()
		LXs := LX.UnsafeSlice()
		for i := 0; i < n; i++ {
			LIs[i] = i
			LXs[i] = i
		}
	}
	L, err := GrB.MatrixNew[int](n, n)
	GrB.OK(err)
	defer try(L.Free)

	if err = L.PackCSC(&LP, &LI, &LX, false, false, nil); err != nil {
		LP.Free()
		LI.Free()
		LX.Free()
		return
	}

	LPrev, err := GrB.MatrixNew[int](n, n)
	GrB.OK(err)
	defer try(LPrev.Free)

	var AT GrB.Matrix[int]
	if !symmetric {
		AT, err = GrB.MatrixNew[int](n, n)
		GrB.OK(err)
		defer try(AT.Free)

		GrB.OK(AT.Transpose(nil, nil, S, nil))
	}

	var I, X []int
	for iteration := 0; iteration < itermax; iteration++ {
		GrB.OK(S.MxM(nil, nil, GrB.MinSecondSemiring[int](), S, L, nil))
		I = I[:0]
		X = X[:0]
		GrB.OK(S.ExtractTuples(&I, nil, &X))
		if !symmetric {
			GrB.OK(AT.MxM(nil, nil, GrB.MinSecondSemiring[int](), AT, L, nil))
			GrB.OK(S.ExtractTuples(&I, nil, &X))
		}

		twoSliceSort(I, X)

		L, LPrev = LPrev, L

		modeValue := -1
		modeLength := 0
		runLength := 1

		for k := 1; k <= nnz; k++ {
			if k == nnz || I[k-1] != I[k] || X[k-1] != X[k] {
				if runLength > modeLength {
					modeValue = X[k-1]
					modeLength = runLength
				}
				runLength = 0
			}
			runLength++

			if k == nnz || I[k-1] != I[k] {
				GrB.OK(L.SetElement(modeValue, I[k-1], I[k-1]))
				modeLength = 0
			}
		}

		isequal, e := MatrixIsEqual(LPrev, L)
		GrB.OK(e)
		if isequal {
			break
		}
	}

	result, err = GrB.VectorNew[int](n)
	GrB.OK(err)
	defer func() {
		if err != nil {
			_ = result.Free()
		}
	}()
	GrB.OK(result.ExtractDiag(L, 0, nil))

	return
}