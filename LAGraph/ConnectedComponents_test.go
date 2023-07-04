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

func checkCC[D GrB.Number](t *testing.T, Component GrB.Vector[int], G *LAGraph.Graph[D]) (err error) {
	defer GrB.CheckErrors(&err)

	GrB.OK(G.Check())

	n, err := G.A.Nrows()
	GrB.OK(err)
	if !Component.Valid() {
		return errors.New("invalid connected components")
	}
	if !(G.Kind == LAGraph.AdjacencyUndirected ||
		(G.Kind == LAGraph.AdjacencyDirected && G.IsSymmetricStructure == LAGraph.True)) {
		return errors.New("G.A must be known to be symmetric")
	}
	queue := make([]int, n)
	componentIn, err := checkVector(Component, n, -1)
	GrB.OK(err)
	count := queue
	ncompIn := 0
	for i := 0; i < n; i++ {
		comp := componentIn[i]
		if comp < 0 || comp > n {
			return errors.New("comp out of range")
		}
		count[comp]++
		if comp == i {
			ncompIn++
		}
	}
	ap, aj, ax, iso, jumbled, err := G.A.UnpackCSR(true, nil)
	GrB.OK(err)
	defer func() {
		GrB.OK(G.A.PackCSR(&ap, &aj, &ax, iso, jumbled, nil))
	}()
	aps := ap.UnsafeSlice()
	ajs := aj.UnsafeSlice()
	visited := make([]bool, n)

	ncomp := 0
	for src := 0; src < n; src++ {
		if visited[src] {
			continue
		}
		comp := componentIn[src]
		ncomp++
		if ncomp > ncompIn {
			return errors.New("wrong number of components")
		}
		queue[0] = src
		head := 0
		tail := 1
		visited[src] = true
		for head < tail {
			u := queue[head]
			head++
			degree := aps[u+1] - aps[u]
			nodeUAdjacencyList := ajs[aps[u]:]
			for k := 0; k < degree; k++ {
				v := nodeUAdjacencyList[k]
				if comp != componentIn[u] {
					return errors.New("component not the same as source")
				}
				if !visited[v] {
					visited[v] = true
					queue[tail] = v
					tail++
				}
			}
		}
	}

	if ncomp != ncompIn {
		return errors.New("wrong number of components")
	}
	return
}

var connectedComponentsFiles = []struct {
	ncomponents int
	name        string
}{
	{1, "karate.mtx"},
	{1, "A.mtx"},
	{1, "jagmesh7.mtx"},
	{1, "ldbc-cdlp-undirected-example.mtx"},
	{1, "ldbc-undirected-example.mtx"},
	{1, "ldbc-wcc-example.mtx"},
	{3, "LFAT5.mtx"},
	{1989, "LFAT5_hypersparse.mtx"},
	{6, "LFAT5_two.mtx"},
	{1, "bcsstk13.mtx"},
	{1, "tree-example.mtx"},
	{1391, "zenios.mtx"},
}

func countConnectedComponents(t *testing.T, C GrB.Vector[int]) int {
	n, err := C.Size()
	if err != nil {
		t.Error(err)
	}
	ncomponents := 0
	for i := 0; i < n; i++ {
		if comp, ok, err := C.ExtractElement(i); err != nil {
			t.Error(err)
		} else if ok && comp == i {
			ncomponents++
		}
	}
	return ncomponents
}

func runTestCCMatrices[D GrB.Number](A GrB.Matrix[D], ncomp int, t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	tryf := func(f func() error) {
		try(f())
	}
	G := LAGraph.New(A, LAGraph.AdjacencyUndirected)
	defer tryf(G.Delete)
	n, err := A.Nrows()
	try(err)

	for trial := 0; trial <= 1; trial++ {
		C, err := G.ConnectedComponents()
		try(err)
		defer tryf(C.Free)
		ncomponents := countConnectedComponents(t, C)
		if ncomponents != ncomp {
			t.Fail()
		}
		cnvals, err := C.Nvals()
		try(err)
		if cnvals != n {
			t.Fail()
		}

		try(checkCC(t, C, G))

		G.Kind = LAGraph.AdjacencyDirected
		G.IsSymmetricStructure = LAGraph.True
	}
}

func TestCCMatrices(t *testing.T) {
	for _, file := range connectedComponentsFiles {
		ncomp := file.ncomponents
		name := file.name
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
			runTestCCMatrices(GrB.MatrixView[int8, float64](m), ncomp, t)
		case GrB.Int16:
			runTestCCMatrices(GrB.MatrixView[int16, float64](m), ncomp, t)
		case GrB.Int32:
			runTestCCMatrices(GrB.MatrixView[int32, float64](m), ncomp, t)
		case GrB.Int64:
			runTestCCMatrices(GrB.MatrixView[int64, float64](m), ncomp, t)
		case GrB.Uint8:
			runTestCCMatrices(GrB.MatrixView[uint8, float64](m), ncomp, t)
		case GrB.Uint16:
			runTestCCMatrices(GrB.MatrixView[uint16, float64](m), ncomp, t)
		case GrB.Uint32:
			runTestCCMatrices(GrB.MatrixView[uint32, float64](m), ncomp, t)
		case GrB.Uint64:
			runTestCCMatrices(GrB.MatrixView[uint64, float64](m), ncomp, t)
		case GrB.Float32:
			runTestCCMatrices(GrB.MatrixView[float32, float64](m), ncomp, t)
		case GrB.Float64:
			runTestCCMatrices(m, ncomp, t)
		default:
			panic("unreachable code")
		}
	}
}
