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

var triangleCountTestFiles = []struct {
	ntriangles int
	name       string
}{
	{45, "karate.mtx"},
	{11, "A.mtx"},
	{2016, "jagmesh7.mtx"},
	{6, "ldbc-cdlp-undirected-example.mtx"},
	{4, "ldbc-undirected-example.mtx"},
	{5, "ldbc-wcc-example.mtx"},
	{0, "LFAT5.mtx"},
	{342300, "bcsstk13.mtx"},
	{0, "tree-example.mtx"},
}

func setupTestTriangleCounts() (G *LAGraph.Graph[uint32], err error) {
	defer GrB.CheckErrors(&err)

	A, err := GrB.MatrixNew[uint32](zacharyNumNodes, zacharyNumNodes)
	GrB.OK(err)
	defer func() {
		if err != nil {
			_ = A.Free()
		}
	}()
	GrB.OK(A.Build(zacharyI, zacharyJ, zacharyV, nil))
	G = LAGraph.New[uint32](A, LAGraph.AdjacencyUndirected)
	defer func() {
		if err != nil {
			_ = G.Delete()
		}
	}()
	GrB.OK(G.CachedNSelfEdges())
	if G.NSelfEdges != 0 {
		return nil, errors.New("NSelfEdges not zero")
	}
	return
}

func TestTriangleCountMethodsBurkhardt(t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	G, err := setupTestTriangleCounts()
	try(err)
	defer func() {
		try(G.Delete())
	}()
	ntriangles, _, _, err := G.TriangleCountMethods(LAGraph.TriangleCountBurkhardt, LAGraph.TriangleCountAutoSort)
	try(err)
	if ntriangles != 45 {
		t.Errorf("ntriangles is %v, expected 45", ntriangles)
	}
}

func TestTriangleCountMethodsCohen(t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	G, err := setupTestTriangleCounts()
	try(err)
	defer func() {
		try(G.Delete())
	}()
	ntriangles, _, _, err := G.TriangleCountMethods(LAGraph.TriangleCountCohen, LAGraph.TriangleCountAutoSort)
	try(err)
	if ntriangles != 45 {
		t.Errorf("ntriangles is %v, expected 45", ntriangles)
	}
}

func TestTriangleCountMethodsSandiaLL(t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	G, err := setupTestTriangleCounts()
	try(err)
	defer func() {
		try(G.Delete())
	}()
	try(G.CachedOutDegree())
	ntriangles, _, _, err := G.TriangleCountMethods(LAGraph.TriangleCountSandiaLL, LAGraph.TriangleCountAutoSort)
	try(err)
	if ntriangles != 45 {
		t.Errorf("ntriangles is %v, expected 45", ntriangles)
	}
}

func TestTriangleCountMethodsSandiaUU(t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	G, err := setupTestTriangleCounts()
	try(err)
	defer func() {
		try(G.Delete())
	}()
	try(G.CachedOutDegree())
	ntriangles, _, _, err := G.TriangleCountMethods(LAGraph.TriangleCountSandiaUU, LAGraph.TriangleCountAutoSort)
	try(err)
	if ntriangles != 45 {
		t.Errorf("ntriangles is %v, expected 45", ntriangles)
	}
}

func TestTriangleCountMethodsSandiaLUT(t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	G, err := setupTestTriangleCounts()
	try(err)
	defer func() {
		try(G.Delete())
	}()
	try(G.CachedOutDegree())
	ntriangles, _, _, err := G.TriangleCountMethods(LAGraph.TriangleCountSandiaLUT, LAGraph.TriangleCountAutoSort)
	try(err)
	if ntriangles != 45 {
		t.Errorf("ntriangles is %v, expected 45", ntriangles)
	}
}

func TestTriangleCountMethodsSandiaULT(t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	G, err := setupTestTriangleCounts()
	try(err)
	defer func() {
		try(G.Delete())
	}()
	try(G.CachedOutDegree())
	ntriangles, _, _, err := G.TriangleCountMethods(LAGraph.TriangleCountSandiaULT, LAGraph.TriangleCountAutoSort)
	try(err)
	if ntriangles != 45 {
		t.Errorf("ntriangles is %v, expected 45", ntriangles)
	}
}

var AllTriangleCountMethods = []LAGraph.TriangleCountMethod{
	LAGraph.TriangleCountAutoMethod,
	LAGraph.TriangleCountBurkhardt,
	LAGraph.TriangleCountCohen,
	LAGraph.TriangleCountSandiaLL,
	LAGraph.TriangleCountSandiaUU,
	LAGraph.TriangleCountSandiaLUT,
	LAGraph.TriangleCountSandiaULT,
}

func TestTriangleCount(t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	G, err := setupTestTriangleCounts()
	try(err)
	defer func() {
		try(G.Delete())
	}()
	ntriangles, err := G.TriangleCount()
	try(err)
	if ntriangles != 45 {
		t.Errorf("ntriangles is %v, expected 45", ntriangles)
	}
}

func runTestTriangleCountMany[D GrB.Number](A GrB.Matrix[D], ntriangles int, t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	G := LAGraph.New(A, LAGraph.AdjacencyUndirected)
	defer func() {
		try(G.Delete())
	}()
	try(G.DeleteSelfEdges())
	nt, err := G.TriangleCount()
	try(err)
	if nt != ntriangles {
		t.Error(nt)
	}
	for _, method := range AllTriangleCountMethods {
		for _, presort := range []LAGraph.TriangleCountPresort{
			LAGraph.TriangleCountNoSort,
			LAGraph.TriangleCountAscending,
			LAGraph.TriangleCountDescending,
			LAGraph.TriangleCountAutoSort,
		} {
			nt, _, _, err := G.TriangleCountMethods(method, presort)
			try(err)
			if nt != ntriangles {
				t.Error(nt)
			}
		}
	}
}

func TestTriangleCountMany(t *testing.T) {
	for _, file := range triangleCountTestFiles {
		name := file.name
		ntriangles := file.ntriangles
		f, err := os.Open(filepath.Join("testdata", name))
		if err != nil {
			_ = f.Close()
			t.Error(err)
		}
		m, err := MatrixMarket.Read[float64](f)
		if err != nil {
			_ = f.Close()
			t.Error(err)
		}
		if err = f.Close(); err != nil {
			t.Error(err)
		}
		typ, ok, err := m.Type()
		if err != nil {
			t.Error(err)
		}
		if !ok {
			t.Fail()
		}
		switch typ {
		case GrB.Int8:
			runTestTriangleCountMany(GrB.MatrixView[int8, float64](m), ntriangles, t)
		case GrB.Int16:
			runTestTriangleCountMany(GrB.MatrixView[int16, float64](m), ntriangles, t)
		case GrB.Int32:
			runTestTriangleCountMany(GrB.MatrixView[int32, float64](m), ntriangles, t)
		case GrB.Int64:
			runTestTriangleCountMany(GrB.MatrixView[int64, float64](m), ntriangles, t)
		case GrB.Uint8:
			runTestTriangleCountMany(GrB.MatrixView[uint8, float64](m), ntriangles, t)
		case GrB.Uint16:
			runTestTriangleCountMany(GrB.MatrixView[uint16, float64](m), ntriangles, t)
		case GrB.Uint32:
			runTestTriangleCountMany(GrB.MatrixView[uint32, float64](m), ntriangles, t)
		case GrB.Uint64:
			runTestTriangleCountMany(GrB.MatrixView[uint64, float64](m), ntriangles, t)
		case GrB.Float32:
			runTestTriangleCountMany(GrB.MatrixView[float32, float64](m), ntriangles, t)
		case GrB.Float64:
			runTestTriangleCountMany(m, ntriangles, t)
		default:
			panic("unreachable code")
		}
	}
}

func TestTriangleCountAutosort(t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	n := 50000
	A, err := GrB.MatrixNew[bool](n, n)
	try(err)
	defer func() {
		try(A.Free())
	}()
	for k := range 11 {
		for i := range n {
			try(A.SetElement(true, i, k))
			try(A.SetElement(true, k, i))
		}
	}
	G := LAGraph.New(A, LAGraph.AdjacencyUndirected)
	defer func() {
		try(G.Delete())
	}()
	try(G.DeleteSelfEdges())
	try(G.CachedOutDegree())
	for _, method := range AllTriangleCountMethods {
		nt, _, _, err := G.TriangleCountMethods(method, LAGraph.TriangleCountAutoSort)
		try(err)
		if nt != 2749560 {
			t.Errorf("ntriangles for method %v is %v, expected 2749560", method, nt)
		}
	}
	nt, err := G.TriangleCount()
	try(err)
	if nt != 2749560 {
		t.Errorf("ntriangles is %v, expected 2749560", nt)
	}
}
