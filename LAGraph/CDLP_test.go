package LAGraph_test

import (
	"github.com/intel/forLAGraphGo/LAGraph"
	"github.com/intel/forLAGraphGo/LAGraph/MatrixMarket"
	"os"
	"path/filepath"
	"testing"
)

var cdlpFiles = []string{
	"A.mtx",
	"jagmesh7.mtx",
	"west0067.mtx", // unsymmetric
	"bcsstk13.mtx",
	"karate.mtx",
	"ldbc-cdlp-undirected-example.mtx",
	"ldbc-undirected-example-bool.mtx",
	"ldbc-undirected-example-unweighted.mtx",
	"ldbc-undirected-example.mtx",
	"ldbc-wcc-example.mtx",
}

func TestCDLP(t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	for _, file := range cdlpFiles {
		f, err := os.Open(filepath.Join("testdata", file))
		try(err)
		A, err := MatrixMarket.Read[int](f)
		try(err)
		try(f.Close())

		G := LAGraph.New(A, LAGraph.AdjacencyDirected)
		try(G.DeleteSelfEdges())
		try(G.CachedIsSymmetricStructure())

		c, err := G.CDLP(100)
		try(err)

		t.Log("checking", file)
		cgood, err := G.CDLPCheck(100)
		try(err)
		ok, err := LAGraph.VectorIsEqual(c, cgood)
		try(err)
		if !ok {
			t.Log(file, "failed")
			t.Fail()
		}
		try(cgood.Free())

		try(c.Free())
		try(G.Delete())
	}
}
