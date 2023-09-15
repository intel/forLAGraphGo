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

func bcDifference(t *testing.T, bc GrB.Vector[float64], gapResult []float64) float32 {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	tryf := func(f func() error) {
		try(f())
	}
	n, err := bc.Size()
	try(err)
	if n != len(gapResult) {
		t.Error(errors.New("incorrect length of GAP result"))
	}
	gapBc, err := GrB.VectorNew[float32](n)
	try(err)
	defer tryf(gapBc.Free)
	for i, v := range gapResult {
		try(gapBc.SetElement(float32(v), i))
	}
	delta, err := GrB.VectorNew[float32](n)
	try(err)
	defer tryf(delta.Free)
	try(GrB.VectorEWiseAddBinaryOp(delta, nil, nil, GrB.Minus[float32](), gapBc, GrB.VectorView[float32, float64](bc), nil))
	try(GrB.VectorApply(delta, nil, nil, GrB.Abs[float32](), delta, nil))
	diff, err := GrB.VectorReduce(GrB.MaxMonoid[float32](), delta, nil)
	try(err)
	return diff
}

var (
	karateSources = []int{6, 29, 0, 9}

	karateBc = []float64{
		43.7778,
		2.83333,
		26.9143,
		0.722222,
		0.333333,
		1.83333,
		1.5,
		0,
		9.09524,
		0,
		0,
		0,
		0,
		5.19206,
		0,
		0,
		0,
		0,
		0,
		4.58095,
		0,
		0,
		0,
		2.4,
		0,
		0.422222,
		0,
		1.28889,
		0,
		0,
		0.733333,
		14.5508,
		17.1873,
		40.6349,
	}
)

var (
	west067Sources = []int{13, 58, 1, 18}

	west0067Bc = []float64{
		7.37262,
		5.3892,
		4.53788,
		3.25952,
		11.9139,
		5.73571,
		5.65336,
		1.5,
		19.2719,
		0.343137,
		0.0833333,
		0.666667,
		1.80882,
		12.4246,
		1.92647,
		22.0458,
		4.7381,
		34.8611,
		0.1,
		29.8358,
		9.52807,
		9.71836,
		17.3334,
		54.654,
		23.3118,
		7.31765,
		2.52381,
		6.96905,
		19.2291,
		6.97003,
		33.0464,
		7.20128,
		3.78571,
		7.87698,
		15.3556,
		7.43333,
		7.19091,
		9.20411,
		1.10325,
		6.38095,
		17.808,
		5.18172,
		25.8441,
		7.91581,
		1.13501,
		0,
		2.53004,
		2.48168,
		8.84857,
		3.80708,
		1.16978,
		0.0714286,
		1.76786,
		3.06661,
		12.0742,
		1.6,
		4.73908,
		2.3701,
		3.75,
		1.08571,
		1.69697,
		0,
		0.571429,
		0,
		0,
		2.22381,
		0.659341,
	}
)

func TestBc(t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	tryf := func(f func() error) {
		try(f())
	}
	{
		f, err := os.Open(filepath.Join("testdata", "karate.mtx"))
		if err != nil {
			_ = f.Close()
			t.Error(err)
		}
		A, err := MatrixMarket.Read[int8](f)
		if err != nil {
			_ = f.Close()
			t.Error(err)
		}
		try(f.Close())
		G := LAGraph.New(A, LAGraph.AdjacencyUndirected)
		defer tryf(G.Delete)
		centrality, err := G.Betweenness(karateSources)
		try(err)
		defer tryf(centrality.Free)
		try(G.Delete())
		diffErr := bcDifference(t, centrality, karateBc)
		if diffErr >= 1e-4 {
			t.Fail()
		}
		try(centrality.Free())
	}

	{
		f, err := os.Open(filepath.Join("testdata", "west0067.mtx"))
		if err != nil {
			_ = f.Close()
			t.Error(err)
		}
		A, err := MatrixMarket.Read[float64](f)
		if err != nil {
			_ = f.Close()
			t.Error(err)
		}
		try(f.Close())
		G := LAGraph.New(A, LAGraph.AdjacencyDirected)
		defer tryf(G.Delete)
		ok, err := G.CachedAT()
		try(err)
		if !ok {
			t.Fail()
		}
		centrality, err := G.Betweenness(west067Sources)
		try(err)
		defer tryf(centrality.Free)
		try(G.Delete())
		diffErr := bcDifference(t, centrality, west0067Bc)
		if diffErr >= 1e-4 {
			t.Fail()
		}
		try(centrality.Free())
	}
}
