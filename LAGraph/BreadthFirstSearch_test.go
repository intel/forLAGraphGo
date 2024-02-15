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

var (
	invalidLevel  = errors.New("invalid level")
	invalidParent = errors.New("invalid parent")
)

func checkBFS[D GrB.Predefined](level, parent *GrB.Vector[int], G *LAGraph.Graph[D], src int) (err error) {
	defer GrB.CheckErrors(&err)
	try := func(f func() error) {
		GrB.OK(f())
	}

	GrB.OK(G.Check())

	n, err := G.A.Nrows()
	GrB.OK(err)

	queue := make([]int, n)

	var levelIn []int
	if level != nil {
		levelIn, err = checkVector(*level, n, -1)
		GrB.OK(err)
	}
	var parentIn []int
	if parent != nil {
		parentIn, err = checkVector(*parent, n, -1)
		GrB.OK(err)
	}

	queue[0] = src
	head := 0
	tail := 1
	visited := make([]bool, n)
	visited[src] = true
	levelCheck := make([]int, n)
	for i := range levelCheck {
		levelCheck[i] = -1
	}
	levelCheck[src] = 0
	Row, err := GrB.VectorNew[bool](n)
	GrB.OK(err)
	defer try(Row.Free)

	var neighbors []int
	var rowValues []bool

	for head < tail {
		u := queue[head]
		head++
		GrB.OK(GrB.MatrixColExtract(Row, nil, nil, GrB.MatrixView[bool](G.A), GrB.All(n), u, GrB.DescT0))
		neighbors = neighbors[:0]
		rowValues = rowValues[:0]
		GrB.OK(Row.ExtractTuples(&neighbors, &rowValues))
		for _, v := range neighbors {
			if !visited[v] {
				visited[v] = true
				levelCheck[v] = levelCheck[u] + 1
				queue[tail] = v
				tail++
			}
		}
	}

	if len(levelIn) > 0 {
		for i := range n {
			if levelIn[i] != levelCheck[i] {
				err = invalidLevel
				return
			}
		}
	}

	if len(parentIn) > 0 {
		for i := range n {
			if i == src {
				if parentIn[src] != src || !visited[src] {
					err = invalidParent
					return
				}
			} else if visited[i] {
				if pi := parentIn[i]; pi < 0 || pi >= n || !visited[pi] {
					err = invalidParent
					return
				} else if _, ok, e := G.A.ExtractElement(pi, i); e != nil || !ok {
					err = invalidParent
					return
				} else if levelCheck[i] != levelCheck[pi]+1 {
					err = invalidParent
					return
				}
			}
		}
	}

	return
}

const testBFSSrc = 30

var levels30 = []int{
	2, 1, 2, 2, 3, 3, 3, 2, 1, 2, 3, 3,
	3, 2, 2, 2, 4, 2, 2, 2, 2, 2, 2, 2,
	3, 3, 2, 2, 2, 2, 0, 2, 1, 1,
}

const xx = -1

var parent30 = [34][3]int{
	{1, 8, xx},   // node 0 can have parents 1 or 8
	{30, xx, xx}, // node 1, parent 30
	{1, 8, 32},   // node 2, parents 1, 8, or 32, etc
	{1, xx, xx},  // node 3
	{0, xx, xx},  // node 4
	{0, xx, xx},  // node 5
	{0, xx, xx},  // node 6
	{1, xx, xx},  // node 7
	{30, xx, xx}, // node 8
	{33, xx, xx}, // node 9
	{0, xx, xx},  // node 10
	{0, xx, xx},  // node 11
	{0, 3, xx},   // node 12
	{1, 33, xx},  // node 13
	{32, 33, xx}, // node 14
	{32, 33, xx}, // node 15
	{5, 6, xx},   // node 16
	{1, xx, xx},  // node 17
	{32, 33, xx}, // node 18
	{1, 33, xx},  // node 19
	{32, 33, xx}, // node 20
	{1, xx, xx},  // node 21
	{32, 33, xx}, // node 22
	{32, 33, xx}, // node 23
	{27, 31, xx}, // node 24
	{23, 31, xx}, // node 25
	{33, xx, xx}, // node 26
	{33, xx, xx}, // node 27
	{33, xx, xx}, // node 28
	{32, 33, xx}, // node 29
	{30, xx, xx}, // node 30, source node
	{32, 33, xx}, // node 31
	{30, xx, xx}, // node 32
	{30, xx, xx}, // node 33
}

var breadthFirstSearchFiles = []struct {
	kind LAGraph.Kind
	name string
}{
	{LAGraph.AdjacencyUndirected, "A.mtx"},
	{LAGraph.AdjacencyDirected, "cover.mtx"},
	{LAGraph.AdjacencyUndirected, "jagmesh7.mtx"},
	{LAGraph.AdjacencyDirected, "ldbc-cdlp-directed-example.mtx"},
	{LAGraph.AdjacencyUndirected, "ldbc-cdlp-undirected-example.mtx"},
	{LAGraph.AdjacencyDirected, "ldbc-directed-example.mtx"},
	{LAGraph.AdjacencyUndirected, "ldbc-undirected-example.mtx"},
	{LAGraph.AdjacencyUndirected, "ldbc-wcc-example.mtx"},
	{LAGraph.AdjacencyUndirected, "LFAT5.mtx"},
	{LAGraph.AdjacencyDirected, "msf1.mtx"},
	{LAGraph.AdjacencyDirected, "msf2.mtx"},
	{LAGraph.AdjacencyDirected, "msf3.mtx"},
	{LAGraph.AdjacencyDirected, "sample2.mtx"},
	{LAGraph.AdjacencyDirected, "sample.mtx"},
	{LAGraph.AdjacencyDirected, "olm1000.mtx"},
	{LAGraph.AdjacencyUndirected, "bcsstk13.mtx"},
	{LAGraph.AdjacencyDirected, "cryg2500.mtx"},
	{LAGraph.AdjacencyUndirected, "tree-example.mtx"},
	{LAGraph.AdjacencyDirected, "west0067.mtx"},
	{LAGraph.AdjacencyUndirected, "karate.mtx"},
	{LAGraph.AdjacencyDirected, "matrix_bool.mtx"},
	{LAGraph.AdjacencyDirected, "skew_fp32.mtx"},
	{LAGraph.AdjacencyUndirected, "pushpull.mtx"},
}

func checkKarateParents30(parents GrB.Vector[int]) (ok bool, err error) {
	defer GrB.CheckErrors(&err)

	n, err := parents.Size()
	GrB.OK(err)

	if n != zacharyNumNodes {
		err = errors.New("incorrect size")
		return
	}

	n, err = parents.Nvals()
	GrB.OK(err)

	if n != zacharyNumNodes {
		err = errors.New("incorrect number of nodes")
		return
	}

	for ix := range zacharyNumNodes {
		var parentId int
		if parentId, ok, err = parents.ExtractElement(ix); err != nil || !ok {
			return
		}

		ok = false
		for k := range 3 {
			validParentId := parent30[ix][k]
			if validParentId < 0 {
				ok = false
				break
			}
			if parentId == validParentId {
				ok = true
				break
			}
		}
		if !ok {
			return
		}
	}
	return
}

func checkKarateLevels30(levels GrB.Vector[int]) (ok bool, err error) {
	defer GrB.CheckErrors(&err)

	n, err := levels.Size()
	GrB.OK(err)

	if n != zacharyNumNodes {
		err = errors.New("incorrect size")
		return
	}

	n, err = levels.Nvals()
	GrB.OK(err)

	if n != zacharyNumNodes {
		err = errors.New("incorrect number of nodes")
		return
	}

	for ix := range zacharyNumNodes {
		var lvl int
		if lvl, ok, err = levels.ExtractElement(ix); err != nil || !ok {
			return
		}
		if lvl != levels30[ix] {
			return false, nil
		}
	}
	return true, nil
}

func setupTestBreadthFirstSearch() (G *LAGraph.Graph[uint32], err error) {
	defer GrB.CheckErrors(&err)

	A, err := GrB.MatrixNew[uint32](zacharyNumNodes, zacharyNumNodes)
	GrB.OK(err)
	defer func() {
		if err != nil {
			_ = A.Free()
		}
	}()

	GrB.OK(A.Build(zacharyI, zacharyJ, zacharyV, nil))
	return LAGraph.New(A, LAGraph.AdjacencyUndirected), nil
}

func TestBreadthFirstSearchParent(t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	G, err := setupTestBreadthFirstSearch()
	try(err)
	defer func() {
		try(G.Delete())
	}()
	_, parent, err := G.BreadthFirstSearch(testBFSSrc, false, true)
	try(err)
	defer func() {
		try(parent.Free())
	}()
	ok, err := checkKarateParents30(parent)
	try(err)
	if !ok {
		t.Error("incorrect parents")
	}
	try(checkBFS(nil, &parent, G, testBFSSrc))
}

func TestBreadthFirstSearchLevel(t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	G, err := setupTestBreadthFirstSearch()
	try(err)
	defer func() {
		try(G.Delete())
	}()
	level, _, err := G.BreadthFirstSearch(testBFSSrc, true, false)
	try(err)
	defer func() {
		try(level.Free())
	}()
	ok, err := checkKarateLevels30(level)
	try(err)
	if !ok {
		t.Error("incorrect levels")
	}
	try(checkBFS(&level, nil, G, testBFSSrc))
}

func TestBreadthFirstSearchBoth(t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	G, err := setupTestBreadthFirstSearch()
	try(err)
	defer func() {
		try(G.Delete())
	}()
	level, parent, err := G.BreadthFirstSearch(testBFSSrc, true, true)
	try(err)
	defer func() {
		try(level.Free())
		try(parent.Free())
	}()
	ok, err := checkKarateLevels30(level)
	try(err)
	if !ok {
		t.Error("incorrect levels")
	}
	ok, err = checkKarateParents30(parent)
	try(err)
	if !ok {
		t.Error("incorrect parents")
	}
	try(checkBFS(&level, &parent, G, testBFSSrc))
}

func runTestBreadthFirstSearchMany[D GrB.Number](A GrB.Matrix[D], kind LAGraph.Kind, t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	G := LAGraph.New(A, kind)
	defer func() {
		try(G.Delete())
	}()
	n, err := G.A.Nrows()
	try(err)
	var step int
	if n > 100 {
		step = 3 * n / 4
	} else {
		step = n/4 + 1
	}
	for range 2 {
		for src := 0; src < n; src += step {
			level, parent, err := G.BreadthFirstSearch(src, true, true)
			try(err)
			try(checkBFS(&level, &parent, G, src))
			try(level.Free())
			try(parent.Free())
			level, _, err = G.BreadthFirstSearch(src, true, false)
			try(err)
			try(checkBFS(&level, nil, G, src))
			try(level.Free())
			_, parent, err = G.BreadthFirstSearch(src, false, true)
			try(err)
			try(checkBFS(nil, &parent, G, src))
			try(parent.Free())
		}

		_, err = G.CachedAT()
		try(err)
		try(G.Check())
		try(G.CachedOutDegree())
		try(G.Check())
		_, err = G.CachedInDegree()
		try(err)
		try(G.Check())
	}
}

func TestBreadthFirstSearchMany(t *testing.T) {
	for _, file := range breadthFirstSearchFiles {
		name := file.name
		kind := file.kind
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
			runTestBreadthFirstSearchMany(GrB.MatrixView[int8, float64](m), kind, t)
		case GrB.Int16:
			runTestBreadthFirstSearchMany(GrB.MatrixView[int16, float64](m), kind, t)
		case GrB.Int32:
			runTestBreadthFirstSearchMany(GrB.MatrixView[int32, float64](m), kind, t)
		case GrB.Int64:
			runTestBreadthFirstSearchMany(GrB.MatrixView[int64, float64](m), kind, t)
		case GrB.Uint8:
			runTestBreadthFirstSearchMany(GrB.MatrixView[uint8, float64](m), kind, t)
		case GrB.Uint16:
			runTestBreadthFirstSearchMany(GrB.MatrixView[uint16, float64](m), kind, t)
		case GrB.Uint32:
			runTestBreadthFirstSearchMany(GrB.MatrixView[uint32, float64](m), kind, t)
		case GrB.Uint64:
			runTestBreadthFirstSearchMany(GrB.MatrixView[uint64, float64](m), kind, t)
		case GrB.Float32:
			runTestBreadthFirstSearchMany(GrB.MatrixView[float32, float64](m), kind, t)
		case GrB.Float64:
			runTestBreadthFirstSearchMany(m, kind, t)
		default:
			panic("unreachable code")
		}
	}
}
