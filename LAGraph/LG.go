package LAGraph

import "github.com/intel/forGraphBLASGo/GrB"

func nselfEdges[D GrB.Predefined](A GrB.Matrix[D]) (nselfEdges int, err error) {
	defer GrB.CheckErrors(&err)
	nselfEdges = Unknown
	nrows, ncols, err := A.Size()
	GrB.OK(err)
	n := Min(nrows, ncols)
	d, err := GrB.VectorNew[D](n)
	GrB.OK(err)
	defer func() {
		GrB.OK(d.Free())
	}()
	GrB.OK(d.ExtractDiag(A, 0, nil))
	return d.Nvals()
}
