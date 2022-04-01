package forLAGraphGo_test

import (
	"bufio"
	GrB "github.com/intel/forGraphBLASGo"
	LAG "github.com/intel/forLAGraphGo"
	"github.com/intel/forLAGraphGo/MatrixMarket"
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

func setupTestTriangleCounts() *LAG.Graph[uint32] {
	A, err := GrB.MatrixNew[uint32](zacharyNumNodes, zacharyNumNodes)
	if err != nil {
		panic(err)
	}
	if err = A.Build(zacharyI, zacharyJ, zacharyV, nil); err != nil {
		panic(err)
	}
	G := LAG.New[uint32](A, LAG.AdjacencyUndirected)
	G.PropertyNDiag()
	if G.NDiag != 0 {
		panic("ndiag not zero")
	}
	return G
}

func TestTriangleCountMethodsBurkhardt(t *testing.T) {
	G := setupTestTriangleCounts()
	presort := LAG.AutoSelectSort
	ntriangles := LAG.TriangleCountMethods(G, LAG.Burkhardt, &presort)
	if ntriangles != 45 {
		t.Errorf("ntriangles is %v, expected 45", ntriangles)
	}
}

func TestTriangleCountMethodsCohen(t *testing.T) {
	G := setupTestTriangleCounts()
	presort := LAG.AutoSelectSort
	ntriangles := LAG.TriangleCountMethods(G, LAG.Cohen, &presort)
	if ntriangles != 45 {
		t.Errorf("ntriangles is %v, expected 45", ntriangles)
	}
}

func TestTriangleCountMethodsSandia(t *testing.T) {
	G := setupTestTriangleCounts()
	presort := LAG.AutoSelectSort
	G.PropertyRowDegree()
	ntriangles := LAG.TriangleCountMethods(G, LAG.Sandia, &presort)
	if ntriangles != 45 {
		t.Errorf("ntriangles is %v, expected 45", ntriangles)
	}
}

func TestTriangleCountMethodsSandia2(t *testing.T) {
	G := setupTestTriangleCounts()
	presort := LAG.AutoSelectSort
	G.PropertyRowDegree()
	ntriangles := LAG.TriangleCountMethods(G, LAG.Sandia2, &presort)
	if ntriangles != 45 {
		t.Errorf("ntriangles is %v, expected 45", ntriangles)
	}
}

func TestTriangleCountMethodsSandiaDot(t *testing.T) {
	G := setupTestTriangleCounts()
	presort := LAG.AutoSelectSort
	G.PropertyRowDegree()
	ntriangles := LAG.TriangleCountMethods(G, LAG.SandiaDot, &presort)
	if ntriangles != 45 {
		t.Errorf("ntriangles is %v, expected 45", ntriangles)
	}
}

func TestTriangleCountMethodsSandiaDot2(t *testing.T) {
	G := setupTestTriangleCounts()
	presort := LAG.AutoSelectSort
	G.PropertyRowDegree()
	ntriangles := LAG.TriangleCountMethods(G, LAG.SandiaDot2, &presort)
	if ntriangles != 45 {
		t.Errorf("ntriangles is %v, expected 45", ntriangles)
	}
}

func TestTriangleCount(t *testing.T) {
	G := setupTestTriangleCounts()
	ntriangles := LAG.TriangleCount(G)
	if ntriangles != 45 {
		t.Errorf("ntriangles is %v, expected 45", ntriangles)
	}
}

func runTestTriangleCountMany[T GrB.Number](f *os.File, header MatrixMarket.Header, scanner *bufio.Scanner, ntriangles int, t *testing.T) {
	A, err := MatrixMarket.Read[T](header, scanner)
	if err != nil {
		_ = f.Close()
		t.Error(err)
	}
	if err = f.Close(); err != nil {
		t.Error(err)
	}
	G := LAG.New(A, LAG.AdjacencyUndirected)
	G.DeleteDiag()
	nt := LAG.TriangleCount(G)
	if nt != ntriangles {
		t.Error(nt)
	}
	for _, method := range LAG.AllTriangleCountMethods {
		for _, presort := range LAG.AllTriangleCountPresorts {
			s := presort
			nt = LAG.TriangleCountMethods(G, method, &s)
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
		t.Log(name, ntriangles)
		f, err := os.Open(filepath.Join("testdata", name))
		if err != nil {
			_ = f.Close()
			t.Error(err)
		}
		header, scanner, err := MatrixMarket.ReadHeader(f)
		if err != nil {
			_ = f.Close()
			t.Error(err)
		}
		switch header.GrBType {
		case GrB.Int8:
			runTestTriangleCountMany[int8](f, header, scanner, ntriangles, t)
		case GrB.Int16:
			runTestTriangleCountMany[int16](f, header, scanner, ntriangles, t)
		case GrB.Int32:
			runTestTriangleCountMany[int32](f, header, scanner, ntriangles, t)
		case GrB.Int64:
			runTestTriangleCountMany[int64](f, header, scanner, ntriangles, t)
		case GrB.Uint8:
			runTestTriangleCountMany[uint8](f, header, scanner, ntriangles, t)
		case GrB.Uint16:
			runTestTriangleCountMany[uint16](f, header, scanner, ntriangles, t)
		case GrB.Uint32:
			runTestTriangleCountMany[uint32](f, header, scanner, ntriangles, t)
		case GrB.Uint64:
			runTestTriangleCountMany[uint64](f, header, scanner, ntriangles, t)
		case GrB.FP32:
			runTestTriangleCountMany[float32](f, header, scanner, ntriangles, t)
		case GrB.FP64:
			runTestTriangleCountMany[float64](f, header, scanner, ntriangles, t)
		default:
			_ = f.Close()
			t.Errorf("Unexpected Matrix Market type %v.", header.GrBType)
		}
	}
}

func TestTriangleCountAutosort(t *testing.T) {
	n := 50000
	A, err := GrB.MatrixNew[int](n, n)
	if err != nil {
		t.Error(err)
	}
	for k := 0; k <= 10; k++ {
		for i := 0; i < n; i++ {
			if err = A.SetElement(1, i, k); err != nil {
				t.Error(err)
			}
			if err = A.SetElement(1, k, i); err != nil {
				t.Error(err)
			}
		}
	}
	if err = A.Wait(GrB.Materialize); err != nil {
		t.Error(err)
	}
	G := LAG.New(A, LAG.AdjacencyUndirected)
	G.DeleteDiag()
	G.PropertyRowDegree()
	for _, method := range LAG.AllTriangleCountMethods {
		t.Logf("method %v", method)
		presort := LAG.AutoSelectSort
		nt := LAG.TriangleCountMethods(G, method, &presort)
		if nt != 2749560 {
			t.Errorf("ntriangles for method %v is %v, expected 2749560", method, nt)
		}
	}
	t.Logf("default method")
	nt := LAG.TriangleCount(G)
	if nt != 2749560 {
		t.Errorf("ntriangles is %v, expected 2749560", nt)
	}
}
