package LAGraph_test

import (
	"errors"
	"github.com/intel/forGraphBLASGo/GrB"
	"github.com/intel/forLAGraphGo/LAGraph"
	"github.com/intel/forLAGraphGo/LAGraph/MatrixMarket"
	"os"
	"path/filepath"
	"testing"
)

func prDifference(centrality GrB.Vector[float32], matlabResult []float64) (diff float32, err error) {
	defer GrB.CheckErrors(&err)
	try := func(f func() error) {
		GrB.OK(f())
	}
	n, err := centrality.Size()
	GrB.OK(err)
	if n != len(matlabResult) {
		err = errors.New("centrality and matlab result do not match")
		return
	}
	matlab, err := GrB.VectorNew[float32](n)
	GrB.OK(err)
	defer try(matlab.Free)
	for i, x := range matlabResult {
		GrB.OK(matlab.SetElement(float32(x), i))
	}
	delta, err := GrB.VectorNew[float32](n)
	GrB.OK(err)
	defer try(delta.Free)
	GrB.OK(GrB.VectorEWiseAddBinaryOp(delta, nil, nil, GrB.Minus[float32](), matlab, centrality, nil))
	GrB.OK(GrB.VectorApply(delta, nil, nil, GrB.Abs[float32](), delta, nil))
	return GrB.VectorReduce(GrB.MaxMonoid[float32](), delta, nil)
}

var (
	karateRank = []float64{
		0.0970011147,
		0.0528720584,
		0.0570750515,
		0.0358615175,
		0.0219857202,
		0.0291233505,
		0.0291233505,
		0.0244945048,
		0.0297681451,
		0.0143104668,
		0.0219857202,
		0.0095668739,
		0.0146475355,
		0.0295415677,
		0.0145381625,
		0.0145381625,
		0.0167900065,
		0.0145622041,
		0.0145381625,
		0.0196092670,
		0.0145381625,
		0.0145622041,
		0.0145381625,
		0.0315206825,
		0.0210719482,
		0.0210013837,
		0.0150430281,
		0.0256382216,
		0.0195723309,
		0.0262863139,
		0.0245921424,
		0.0371606178,
		0.0716632142,
		0.1008786453,
	}

	west0067Rank = []float64{
		0.0233753869,
		0.0139102552,
		0.0123441027,
		0.0145657095,
		0.0142018541,
		0.0100791606,
		0.0128753395,
		0.0143945684,
		0.0110203141,
		0.0110525383,
		0.0119311961,
		0.0072382247,
		0.0188680398,
		0.0141596605,
		0.0174877889,
		0.0170362099,
		0.0120433909,
		0.0219844489,
		0.0195274443,
		0.0394465722,
		0.0112038726,
		0.0090174094,
		0.0140088120,
		0.0122532937,
		0.0153346283,
		0.0135241334,
		0.0158714693,
		0.0149689529,
		0.0144097230,
		0.0137583019,
		0.0314386080,
		0.0092857745,
		0.0081814168,
		0.0102137827,
		0.0096547214,
		0.0129622400,
		0.0244173417,
		0.0173963657,
		0.0127705717,
		0.0143297446,
		0.0140509341,
		0.0104117131,
		0.0173516407,
		0.0149175105,
		0.0119979624,
		0.0095043613,
		0.0153295328,
		0.0077710930,
		0.0259969472,
		0.0126926269,
		0.0088870166,
		0.0080836101,
		0.0096023576,
		0.0091000837,
		0.0246131958,
		0.0159589365,
		0.0183500031,
		0.0155811507,
		0.0157693756,
		0.0116319823,
		0.0230649292,
		0.0149070613,
		0.0157469640,
		0.0134396036,
		0.0189218603,
		0.0114528518,
		0.0223213267,
	}

	ldbcDirectedExampleRank = []float64{
		0.1697481823,
		0.0361514465,
		0.1673241104,
		0.1669092572,
		0.1540948145,
		0.0361514465,
		0.0361514465,
		0.1153655134,
		0.0361514465,
		0.0819523364,
	}
)

func TestRanker(t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	{
		f, err := os.Open(filepath.Join("testdata", "karate.mtx"))
		try(err)
		A, err := MatrixMarket.Read[float32](f)
		try(err)
		try(f.Close())
		G := LAGraph.New(A, LAGraph.AdjacencyUndirected)
		try(G.CachedOutDegree())
		centrality, _, err := G.PageRankGAP(0.85, 1e-4, 100)
		try(err)
		diff, err := prDifference(centrality, karateRank)
		try(err)
		_, err = GrB.VectorReduce(GrB.PlusMonoid[float32](), centrality, nil)
		try(err)
		if diff >= 1e-4 {
			t.Fail()
		}
		try(centrality.Free())

		centrality, _, err = G.PageRank(0.85, 1e-4, 100)
		try(err)
		diff, err = prDifference(centrality, karateRank)
		try(err)
		_, err = GrB.VectorReduce(GrB.PlusMonoid[float32](), centrality, nil)
		try(err)
		if diff >= 1e-4 {
			t.Fail()
		}
		try(centrality.Free())

		_, _, err = G.PageRank(0.85, 1e-4, 2)
		if err != LAGraph.ConvergenceFailure {
			t.Fail()
		}

		try(G.Delete())
	}

	{
		f, err := os.Open(filepath.Join("testdata", "west0067.mtx"))
		try(err)
		A, err := MatrixMarket.Read[float32](f)
		try(err)
		try(f.Close())
		G := LAGraph.New(A, LAGraph.AdjacencyDirected)
		_, err = G.CachedAT()
		try(err)
		try(G.CachedOutDegree())
		centrality, _, err := G.PageRankGAP(0.85, 1e-4, 100)
		try(err)
		diff, err := prDifference(centrality, west0067Rank)
		try(err)
		_, err = GrB.VectorReduce(GrB.PlusMonoid[float32](), centrality, nil)
		try(err)
		if diff >= 1e-4 {
			t.Fail()
		}
		try(centrality.Free())

		centrality, _, err = G.PageRank(0.85, 1e-4, 100)
		try(err)
		diff, err = prDifference(centrality, west0067Rank)
		try(err)
		_, err = GrB.VectorReduce(GrB.PlusMonoid[float32](), centrality, nil)
		try(err)
		if diff >= 1e-4 {
			t.Fail()
		}
		try(centrality.Free())

		try(G.Delete())
	}

	{
		f, err := os.Open(filepath.Join("testdata", "ldbc-directed-example.mtx"))
		try(err)
		A, err := MatrixMarket.Read[float32](f)
		try(err)
		try(f.Close())
		G := LAGraph.New(A, LAGraph.AdjacencyDirected)
		_, err = G.CachedAT()
		try(err)
		try(G.CachedOutDegree())
		centrality, _, err := G.PageRankGAP(0.85, 1e-4, 100)
		try(err)
		_, err = prDifference(centrality, ldbcDirectedExampleRank)
		try(err)
		_, err = GrB.VectorReduce(GrB.PlusMonoid[float32](), centrality, nil)
		try(err)
		try(centrality.Free())

		centrality, _, err = G.PageRank(0.85, 1e-4, 100)
		try(err)
		diff, err := prDifference(centrality, ldbcDirectedExampleRank)
		try(err)
		_, err = GrB.VectorReduce(GrB.PlusMonoid[float32](), centrality, nil)
		try(err)
		if diff >= 1e-4 {
			t.Fail()
		}
		try(centrality.Free())

		try(G.Delete())
	}

}
