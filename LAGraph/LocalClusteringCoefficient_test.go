package LAGraph_test

import (
	"github.com/intel/forGraphBLASGo/GrB"
	"github.com/intel/forLAGraphGo/LAGraph"
	"github.com/intel/forLAGraphGo/LAGraph/MatrixMarket"
	"os"
	"path/filepath"
	"testing"
)

var lccFiles = []string{
	"A.mtx",
	"jagmesh7.mtx",
	"west0067.mtx",
	"bcsstk13.mtx",
	"karate.mtx",
	"ldbc-cdlp-undirected-example.mtx",
	"ldbc-undirected-example-bool.mtx",
	"ldbc-undirected-example-unweighted.mtx",
	"ldbc-undirected-example.mtx",
	"ldbc-wcc-example.mtx",
}

func TestLCC(t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	for _, filename := range lccFiles {
		f, err := os.Open(filepath.Join("testdata", filename))
		try(err)
		A, err := MatrixMarket.Read[float64](f)
		try(err)
		try(f.Close())
		G := LAGraph.New(A, LAGraph.AdjacencyDirected)

		try(G.CachedIsSymmetricStructure())
		try(G.CachedNSelfEdges())

		cgood, err := G.LCCCheck()
		try(err)
		try(cgood.Wait(GrB.Materialize))

		c, err := G.LocalClusteringCoefficient()
		try(err)

		try(GrB.VectorEWiseAddBinaryOp(cgood, nil, nil, GrB.Minus[float64](), cgood, c, nil))
		try(GrB.VectorApply(cgood, nil, nil, GrB.Abs[float64](), cgood, nil))
		diff, err := GrB.VectorReduce(GrB.MaxMonoid[float64](), cgood, nil)
		try(err)
		if diff >= 1e-6 {
			t.Log(filename)
			t.Fail()
		}
		try(cgood.Free())
		try(c.Free())

		try(G.Delete())
	}
}
