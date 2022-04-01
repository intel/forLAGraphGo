package forLAGraphGo

import (
	GrB "github.com/intel/forGraphBLASGo"
	"log"
)

type Graph[T any] struct {
	A    *GrB.Matrix[T]
	Kind Kind

	AT                    *GrB.Matrix[T]
	RowDegree, ColDegree  *GrB.Vector[int]
	AStructureIsSymmetric Boolean
	NDiag                 int
}

func New[T GrB.Number](A *GrB.Matrix[T], kind Kind) *Graph[T] {
	return &Graph[T]{A: A, Kind: kind, NDiag: -1}
}

func (G *Graph[T]) Delete() {
	G.A.Clear()
	G.DeleteProperties()
}

func (G *Graph[T]) DeleteProperties() {
	G.AT = nil
	G.RowDegree = nil
	G.ColDegree = nil
	G.AStructureIsSymmetric = Unknown
	G.NDiag = -1
}

func (G *Graph[T]) PropertyAT() {
	if G.AT != nil || G.Kind == AdjacencyUndirected {
		return
	}
	A := G.A
	nrows, ncols, err := A.Size()
	try(err)
	AT, err := GrB.MatrixNew[T](ncols, nrows)
	try(err)
	log.Println("GrB.Transpose(A)")
	try(GrB.Transpose(AT, nil, nil, A, nil))
	G.AT = AT
}

func (G *Graph[T]) PropertyASymmetricStructure() {
	if G.Kind == AdjacencyUndirected {
		G.AStructureIsSymmetric = True
		return
	}
	A := G.A
	log.Println("A.Size()")
	n, ncols, err := A.Size()
	try(err)
	if n != ncols {
		G.AStructureIsSymmetric = False
		return
	}
	if G.AT == nil {
		log.Println("G.PropertyAT()")
		try(A.Wait(GrB.Materialize))
		G.PropertyAT()
	}
	C, err := GrB.MatrixNew[bool](n, n)
	try(err)
	log.Println("GrB.MatrixEWiseMultBinaryOp(C, A, G.AT)")
	try(GrB.MatrixEWiseMultBinaryOp(C, nil, nil, GrB.Trueb[T, T], A, G.AT, nil))
	log.Println("C.NVals()")
	nvals1, err := C.NVals()
	try(err)
	log.Println("A.NVals()")
	nvals2, err := A.NVals()
	try(err)
	if nvals1 == nvals2 {
		G.AStructureIsSymmetric = True
	} else {
		G.AStructureIsSymmetric = False
	}
}

func (G *Graph[T]) PropertyRowDegree() {
	if G.RowDegree != nil {
		return
	}
	A := G.A
	nrows, ncols, err := A.Size()
	try(err)
	rowDegree, err := GrB.VectorNew[int](nrows)
	try(err)
	x, err := GrB.VectorNew[int](ncols)
	try(err)
	try(GrB.VectorAssignConstant(x, nil, nil, 0, GrB.All(ncols), nil))
	try(GrB.MxV(rowDegree, nil, nil, PlusOne[int, T, int], A, x, nil))
	try(rowDegree.Wait(GrB.Materialize))
	G.RowDegree = rowDegree
}

func (G *Graph[T]) PropertyColDegree() {
	if G.ColDegree != nil || G.Kind == AdjacencyUndirected {
		return
	}
	A := G.A
	AT := G.AT
	nrows, ncols, err := A.Size()
	try(err)
	colDegree, err := GrB.VectorNew[int](ncols)
	try(err)
	x, err := GrB.VectorNew[int](nrows)
	try(err)
	try(GrB.VectorAssignConstant(x, nil, nil, 0, GrB.All(nrows), nil))
	if AT != nil {
		try(GrB.MxV(colDegree, nil, nil, PlusOne[int, T, int], AT, x, nil))
	} else {
		try(GrB.MxV(colDegree, nil, nil, PlusOne[int, T, int], A, x, GrB.DescT0))
	}
	try(colDegree.Wait(GrB.Materialize))
	G.ColDegree = colDegree
}

func (G *Graph[T]) PropertyNDiag() {
	if G.NDiag >= 0 {
		return
	}
	A := G.A
	nrows, ncols, err := A.Size()
	try(err)
	n := GrB.Min(nrows, ncols)
	M, err := GrB.MatrixNew[bool](nrows, ncols)
	try(err)
	for i := 0; i < n; i++ {
		try(M.SetElement(true, i, i))
	}
	try(M.Wait(GrB.Materialize))
	D, err := GrB.MatrixNew[T](nrows, ncols)
	try(err)
	try(GrB.MatrixAssign(D, M, nil, A, GrB.All(nrows), GrB.All(ncols), GrB.DescS))
	ndiag, err := D.NVals()
	try(err)
	G.NDiag = ndiag
}

func (G *Graph[T]) DeleteDiag() {
	if G.NDiag == 0 {
		return
	}
	aStructureIsSymmetric := G.AStructureIsSymmetric
	G.DeleteProperties()
	G.AStructureIsSymmetric = aStructureIsSymmetric
	try(GrB.MatrixSelect(G.A, nil, nil, GrB.OffDiag[T], G.A, 0, nil))
	G.NDiag = 0
}
