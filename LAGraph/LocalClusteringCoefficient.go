package LAGraph

import (
	"errors"
	"github.com/intel/forGraphBLASGo/GrB"
)

const (
	combDirName = "comb_dir"
	combDirDef  = `void comb_dir(void *z, const void *x) {
		double xd = *(double *) x;
		double *zd = (double *) z;
		(*zd) = ((xd) * (xd - 1));
    }`
	combUndirName = "comb_undir"
	combUndirDef  = `void comb_undir(void *z, const void *x) {
		double xd = *(double *) x;
		double *zd = (double *) z;
		(*zd) = ((xd) * (xd - 1)) / 2;
    }`
)

func (G *Graph[D]) LocalClusteringCoefficient() (coefficients GrB.Vector[float64], err error) {
	defer GrB.CheckErrors(&err)
	try := func(f func() error) {
		GrB.OK(f())
	}

	if G.IsSymmetricStructure == BooleanUnknown {
		err = errors.New("G.IsSymmetricStructure is required")
		return
	}
	if G.NSelfEdges == Unknown {
		err = errors.New("G.NSelfEdges is required")
		return
	}

	A := G.A
	n, ncols, err := A.Size()
	GrB.OK(err)
	if n != ncols {
		err = GrB.InvalidValue
		return
	}

	S, err := GrB.MatrixNew[float64](n, n)
	GrB.OK(err)
	defer try(S.Free)

	GrB.OK(GrB.MatrixApplyBinaryOp2nd(S, nil, nil, GrB.Oneb[float64](), GrB.MatrixView[float64, D](A), 0, nil))
	if G.NSelfEdges != 0 {
		GrB.OK(GrB.MatrixSelect(S, nil, nil, GrB.Offdiag[float64](), S, 0, nil))
	}

	var comb GrB.UnaryOp[float64, float64]
	if G.IsSymmetricStructure == True {
		comb, err = GrB.NamedUnaryOpNew[float64, float64](nil, combUndirName, combUndirDef)
	} else {
		comb, err = GrB.NamedUnaryOpNew[float64, float64](nil, combDirName, combDirDef)
	}
	GrB.OK(err)
	defer try(comb.Free)

	U, err := GrB.MatrixNew[float64](n, n)
	GrB.OK(err)
	defer try(U.Free)

	if G.IsSymmetricStructure == False {
		GrB.OK(GrB.MatrixEWiseAddBinaryOp(S, nil, nil, GrB.Plus[float64](), S, S, GrB.DescT1))
	}
	GrB.OK(GrB.MatrixSelect(U, nil, nil, GrB.Triu[float64](), S, 0, nil))

	W, err := GrB.VectorNew[float64](n)
	GrB.OK(err)
	defer try(W.Free)

	x, err := GrB.VectorNew[float64](n)
	GrB.OK(err)
	defer try(x.Free)
	GrB.OK(GrB.VectorAssignConstant(x, nil, nil, 0, GrB.All(n), nil))
	GrB.OK(GrB.MxV(W, nil, nil, PlusOne[float64](), S, x, nil))
	GrB.OK(x.Free())

	GrB.OK(GrB.VectorApply(W, nil, nil, comb, W, nil))

	CL, err := GrB.MatrixNew[float64](n, n)
	GrB.OK(err)
	defer try(CL.Free)

	GrB.OK(GrB.MxM(CL, S.AsMask(), nil, GrB.PlusSecond[float64](), S, U, GrB.DescST1))
	GrB.OK(S.Free())
	GrB.OK(U.Free())

	LCC, err := GrB.VectorNew[float64](n)
	GrB.OK(err)
	defer func() {
		if err != nil {
			_ = LCC.Free()
		}
	}()
	GrB.OK(GrB.MatrixReduceBinaryOp(LCC, nil, nil, GrB.Plus[float64](), CL, nil))
	GrB.OK(CL.Free())

	GrB.OK(GrB.VectorEWiseMultBinaryOp(LCC, nil, nil, GrB.Div[float64](), LCC, W, nil))

	coefficients = LCC
	return
}
