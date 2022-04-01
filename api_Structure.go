package forLAGraphGo

import GrB "github.com/intel/forGraphBLASGo"

func Structure[T any](A GrB.Matrix[T]) (C *GrB.Matrix[bool], functionErr error) {
	nrows, ncols, err := A.Size()
	if err != nil {
		functionErr = err
		return
	}
	C, err = GrB.MatrixNew[bool](nrows, ncols)
	if err != nil {
		functionErr = err
		return
	}
	functionErr = GrB.MatrixAssignConstant(C, A.AsMask(), nil, true, GrB.All(nrows), GrB.All(ncols), GrB.DescS)
	return
}
