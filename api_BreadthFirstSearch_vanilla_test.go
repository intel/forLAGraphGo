package forLAGraphGo_test

import (
	"bufio"
	"github.com/intel/forGoParallel/parallel"
	GrB "github.com/intel/forGraphBLASGo"
	LAG "github.com/intel/forLAGraphGo"
	"github.com/intel/forLAGraphGo/MatrixMarket"
	"os"
	"path/filepath"
	"testing"
)

const src = 30

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
	kind LAG.Kind
	name string
}{
	{LAG.AdjacencyUndirected, "A.mtx"},
	{LAG.AdjacencyDirected, "cover.mtx"},
	{LAG.AdjacencyUndirected, "jagmesh7.mtx"},
	{LAG.AdjacencyDirected, "ldbc-cdlp-directed-example.mtx"},
	{LAG.AdjacencyUndirected, "ldbc-cdlp-undirected-example.mtx"},
	{LAG.AdjacencyDirected, "ldbc-directed-example.mtx"},
	{LAG.AdjacencyUndirected, "ldbc-undirected-example.mtx"},
	{LAG.AdjacencyUndirected, "ldbc-wcc-example.mtx"},
	{LAG.AdjacencyUndirected, "LFAT5.mtx"},
	{LAG.AdjacencyDirected, "msf1.mtx"},
	{LAG.AdjacencyDirected, "msf2.mtx"},
	{LAG.AdjacencyDirected, "msf3.mtx"},
	{LAG.AdjacencyDirected, "sample2.mtx"},
	{LAG.AdjacencyDirected, "sample.mtx"},
	{LAG.AdjacencyDirected, "olm1000.mtx"},
	{LAG.AdjacencyUndirected, "bcsstk13.mtx"},
	{LAG.AdjacencyDirected, "cryg2500.mtx"},
	{LAG.AdjacencyUndirected, "tree-example.mtx"},
	{LAG.AdjacencyDirected, "west0067.mtx"},
	{LAG.AdjacencyUndirected, "karate.mtx"},
	{LAG.AdjacencyDirected, "matrix_bool.mtx"},
	{LAG.AdjacencyDirected, "skew_fp32.mtx"},
	{LAG.AdjacencyUndirected, "pushpull.mtx"},
}

func checkKarateParents30(parents *GrB.Vector[int]) bool {
	n, err := parents.Size()
	if err != nil {
		panic(err)
	}
	if n != zacharyNumNodes {
		panic("incorrect size")
	}
	if err = parents.Wait(GrB.Materialize); err != nil {
		panic(err)
	}
	n, err = parents.NVals()
	if err != nil {
		panic(err)
	}
	if n != zacharyNumNodes {
		panic("incorrect number of nodes")
	}

	return parallel.RangeAnd(0, zacharyNumNodes, func(low, high int) bool {
		ok := false
		for ix := low; ix < high; ix++ {
			parentId, err := parents.ExtractElement(ix)
			if err != nil {
				panic(err)
			}

			ok = false
			for k := 0; k <= 2; k++ {
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
				break
			}
		}
		return ok
	})
}

func checkKarateLevels30(levels *GrB.Vector[int]) bool {
	n, err := levels.Size()
	if err != nil {
		panic(err)
	}
	if n != zacharyNumNodes {
		panic("incorrect size")
	}
	if err = levels.Wait(GrB.Materialize); err != nil {
		panic(err)
	}
	n, err = levels.NVals()
	if err != nil {
		panic(err)
	}
	if n != zacharyNumNodes {
		panic("incorrect number of nodes")
	}

	return parallel.RangeAnd(0, zacharyNumNodes, func(low, high int) bool {
		for ix := low; ix < high; ix++ {
			lvl, err := levels.ExtractElement(ix)
			if err != nil {
				panic(err)
			}
			if lvl != levels30[ix] {
				return false
			}
		}
		return true
	})
}

func setupTestBreadthFirstSearch() *LAG.Graph[uint32] {
	A, err := GrB.MatrixNew[uint32](zacharyNumNodes, zacharyNumNodes)
	if err != nil {
		panic(err)
	}
	if err = A.Build(zacharyI, zacharyJ, zacharyV, nil); err != nil {
		panic(err)
	}
	G := LAG.New(A, LAG.AdjacencyUndirected)
	return G
}

func TestBreadthFirstSearchParent(t *testing.T) {
	G := setupTestBreadthFirstSearch()
	_, parent := LAG.BreadthFirstSearchVanilla(G, 30, false, true)
	if !checkKarateParents30(parent) {
		t.Error("incorrect parents")
	}
	LAG.CheckBFS(nil, parent, G, 30)
}

func TestBreadthFirstSearchLevel(t *testing.T) {
	G := setupTestBreadthFirstSearch()
	level, _ := LAG.BreadthFirstSearchVanilla(G, 30, true, false)
	if !checkKarateLevels30(level) {
		t.Error("incorrect levels")
	}
	LAG.CheckBFS(level, nil, G, 30)
}

func TestBreadthFirstSearchBoth(t *testing.T) {
	G := setupTestBreadthFirstSearch()
	level, parent := LAG.BreadthFirstSearchVanilla(G, 30, true, true)
	if !checkKarateLevels30(level) {
		t.Error("incorrect levels")
	}
	if !checkKarateParents30(parent) {
		t.Error("incorrect parents")
	}
	LAG.CheckBFS(level, parent, G, 30)
}

func runTestBreadthFirstSearchMany[T GrB.Number](f *os.File, header MatrixMarket.Header, scanner *bufio.Scanner, kind LAG.Kind, t *testing.T) {
	A, err := MatrixMarket.Read[T](header, scanner)
	if err != nil {
		_ = f.Close()
		t.Error(err)
	}
	if err = f.Close(); err != nil {
		t.Error(err)
	}
	G := LAG.New(A, kind)
	n, err := G.A.NRows()
	if err != nil {
		t.Error(err)
	}
	var step int
	if n > 100 {
		step = 3 * n / 4
	} else {
		step = n/4 + 1
	}
	for src := 0; src < n; src += step {
		level, parent := LAG.BreadthFirstSearchVanilla(G, src, true, true)
		LAG.CheckBFS(level, parent, G, src)
		level, _ = LAG.BreadthFirstSearchVanilla(G, src, true, false)
		LAG.CheckBFS(level, nil, G, src)
		_, parent = LAG.BreadthFirstSearchVanilla(G, src, false, true)
		LAG.CheckBFS(nil, parent, G, src)
	}
}

func TestBreadthFirstSearchMany(t *testing.T) {
	for _, file := range breadthFirstSearchFiles {
		name := file.name
		kind := file.kind
		t.Log(name)
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
			runTestBreadthFirstSearchMany[int8](f, header, scanner, kind, t)
		case GrB.Int16:
			runTestBreadthFirstSearchMany[int16](f, header, scanner, kind, t)
		case GrB.Int32:
			runTestBreadthFirstSearchMany[int32](f, header, scanner, kind, t)
		case GrB.Int64:
			runTestBreadthFirstSearchMany[int64](f, header, scanner, kind, t)
		case GrB.Uint8:
			runTestBreadthFirstSearchMany[uint8](f, header, scanner, kind, t)
		case GrB.Uint16:
			runTestBreadthFirstSearchMany[uint16](f, header, scanner, kind, t)
		case GrB.Uint32:
			runTestBreadthFirstSearchMany[uint32](f, header, scanner, kind, t)
		case GrB.Uint64:
			runTestBreadthFirstSearchMany[uint64](f, header, scanner, kind, t)
		case GrB.FP32:
			runTestBreadthFirstSearchMany[float32](f, header, scanner, kind, t)
		case GrB.FP64:
			runTestBreadthFirstSearchMany[float64](f, header, scanner, kind, t)
		default:
			_ = f.Close()
			t.Error("invalid Matrix Market type")
		}
	}
}
