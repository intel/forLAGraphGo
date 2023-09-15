package LAGraph_test

import (
	"container/heap"
	"errors"
	"github.com/intel/forGraphBLASGo/GrB"
	"github.com/intel/forLAGraphGo/LAGraph"
	"github.com/intel/forLAGraphGo/LAGraph/MatrixMarket"
	"math"
	"os"
	"path/filepath"
	"testing"
)

var ssspFiles = []string{
	"A.mtx",
	"cover.mtx",
	"jagmesh7.mtx",
	"ldbc-cdlp-directed-example.mtx",
	"ldbc-cdlp-undirected-example.mtx",
	"ldbc-directed-example.mtx",
	"ldbc-undirected-example.mtx",
	"ldbc-wcc-example.mtx",
	"LFAT5.mtx",
	"LFAT5_hypersparse.mtx",
	"msf1.mtx",
	"msf2.mtx",
	"msf3.mtx",
	"sample2.mtx",
	"sample.mtx",
	"olm1000.mtx",
	"bcsstk13.mtx",
	"cryg2500.mtx",
	"tree-example.mtx",
	"west0067.mtx",
	"karate.mtx",
	"matrix_bool.mtx",
	"test_BF.mtx",
	"test_FW_1000.mtx",
	"test_FW_2003.mtx",
	"test_FW_2500.mtx",
	"skew_fp32.mtx",
	"matrix_uint32.mtx",
	"matrix_uint64.mtx",
}

type (
	lgElement struct {
		name int
		key  float64
	}

	lgHeap struct {
		slice []lgElement
		iheap map[int]int
	}
)

func (h lgHeap) Len() int {
	return len(h.slice)
}

func (h lgHeap) Less(i, j int) bool {
	return h.slice[i].key < h.slice[j].key
}

func (h lgHeap) Swap(i, j int) {
	h.iheap[h.slice[i].name] = j
	h.iheap[h.slice[j].name] = i
	h.slice[i], h.slice[j] = h.slice[j], h.slice[i]
}

func (h *lgHeap) Push(x any) {
	e := x.(lgElement)
	h.iheap[e.name] = len(h.slice)
	h.slice = append(h.slice, e)
}

func (h *lgHeap) Pop() any {
	length := len(h.slice)
	e := h.slice[length-1]
	delete(h.iheap, e.name)
	h.slice = h.slice[:length-1]
	return e
}

func initHeap(slice []lgElement) lgHeap {
	iheap := make(map[int]int, len(slice))
	for i, e := range slice {
		iheap[e.name] = i
	}
	result := lgHeap{
		slice: slice,
		iheap: iheap,
	}
	heap.Init(&result)
	return result
}

func checkSSSP[D LAGraph.SingleSourceShortestPathDomains](pathLength GrB.Vector[D], G *LAGraph.Graph[D], src int) (err error) {
	defer GrB.CheckErrors(&err)

	GrB.OK(G.Check())
	n, err := G.A.Nrows()
	GrB.OK(err)
	dinf := float64(GrB.Maximum[D]())
	infinity := math.Inf(+1)

	pathLengthIn := make([]float64, n)
	reachableIn := make([]bool, n)
	for i := range pathLengthIn {
		pathLengthIn[i] = infinity
		t, ok, e := GrB.VectorView[float64, D](pathLength).ExtractElement(i)
		GrB.OK(e)
		if ok {
			pathLengthIn[i] = t
		}
		reachableIn[i] = pathLengthIn[i] < dinf
	}
	ap, aj, ax, iso, jumbled, err := G.A.UnpackCSR(true, nil)
	GrB.OK(err)
	aps := ap.UnsafeSlice()
	ajs := aj.UnsafeSlice()
	axs := ax.UnsafeSlice()

	distance := make([]float64, n)
	reachable := make([]bool, n)
	for i := range distance {
		distance[i] = infinity
	}
	distance[src] = 0
	reachable[src] = true

	slice := make([]lgElement, n)
	for i := range slice {
		slice[i].name = i
		if i != src {
			slice[i].key = infinity
		}
	}
	hp := initHeap(slice)

	for len(hp.slice) > 0 {
		e := heap.Pop(&hp).(lgElement)
		u := e.name
		uDistance := e.key
		if _, ok := hp.iheap[u]; ok {
			return errors.New("invalid heap")
		}
		if distance[u] != uDistance {
			return errors.New("distance[u] != uDistance")
		}
		reachable[u] = uDistance < dinf
		if uDistance == infinity {
			break
		}
		degree := aps[u+1] - aps[u]
		nodeUAdjancencyList := ajs[aps[u]:]
		var delta int
		if !iso {
			delta = aps[u]
		}
		weights := axs[delta:]

		for k := 0; k < degree; k++ {
			v := nodeUAdjancencyList[k]
			if _, ok := hp.iheap[v]; !ok {
				continue
			}
			var pos int
			if !iso {
				pos = k
			}
			w := weights[pos]
			if w <= 0 {
				return errors.New("invalid graph (weights must be > 0)")
			}
			newDistance := uDistance + float64(w)
			if distance[v] > newDistance {
				distance[v] = newDistance
				p := hp.iheap[v]
				if hp.slice[p].name != v {
					return errors.New("invalid heap")
				}
				hp.slice[p].key = newDistance
				heap.Fix(&hp, p)
			}
		}
	}

	GrB.OK(G.A.PackCSR(&ap, &aj, &ax, iso, jumbled, nil))

	for i := 0; i < n; i++ {
		var ok bool
		e := float64(0)
		if math.IsInf(distance[i], +1) {
			ok = pathLengthIn[i] == dinf || math.IsInf(pathLengthIn[i], +1)
		} else {
			e = math.Abs(pathLengthIn[i] - distance[i])
			d := math.Max(pathLengthIn[i], distance[i])
			if e > 0 {
				e = e / d
			}
			ok = e < 1e-5
		}
		if !ok {
			return errors.New("invalid path length")
		}
	}

	for i := 0; i < n; i++ {
		ok := reachable[i] == reachableIn[i]
		if !ok {
			return errors.New("invalid reach")
		}
	}

	return
}

func TestSingleSourceShortestPath(t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	for _, aname := range ssspFiles {
		f, err := os.Open(filepath.Join("testdata", aname))
		try(err)
		A, err := MatrixMarket.Read[int32](f)
		try(err)
		try(f.Close())
		n, ncols, err := A.Size()
		try(err)
		if n != ncols {
			t.Fail()
		}
		if typ, ok, err := A.Type(); err != nil {
			t.Error(err)
		} else if !ok {
			t.Fail()
		} else if typ != GrB.Int32 {
			T, err := GrB.MatrixNew[int32](n, n)
			try(err)
			try(GrB.MatrixAssign(T, nil, nil, A, GrB.All(n), GrB.All(n), nil))
			try(A.Free())
			A = T
		}
		try(GrB.MatrixApplyBinaryOp2nd(A, nil, nil, GrB.Band[int32](), A, 255, nil))
		x, err := GrB.MatrixReduce(GrB.MinMonoid[int32](), A, nil)
		try(err)
		if x < 1 {
			try(GrB.MatrixApplyBinaryOp2nd(A, nil, nil, GrB.Max[int32](), A, 1, nil))
		}

		G := LAGraph.New(A, LAGraph.AdjacencyDirected)
		try(G.Check())
		G.EMin, err = GrB.ScalarNew[int32]()
		try(err)
		try(G.EMin.SetElement(1))
		deltas := [...]int32{30, 100, 50000}
		var step int
		if n > 100 {
			step = 3 * n / 4
		} else {
			step = n/4 + 1
		}
		for src := 0; src < n; src += step {
			for _, delta := range deltas {
				pathLength, err := LAGraph.SingleSourceShortestPath(G, src, delta)
				try(err)
				try(checkSSSP(pathLength, G, src))
				try(pathLength.Free())
			}
		}
		try(G.EMin.Free())
		G.EMinState = LAGraph.Unknown
		try(G.A.SetElement(-1, 0, 1))
		pathLength, err := LAGraph.SingleSourceShortestPath(G, 0, 30)
		try(err)
		length, _, err := pathLength.ExtractElement(1)
		try(err)
		if length != -1 {
			t.Fail()
		}
		try(G.Delete())
	}
}

func prepareSSSPTypes[To LAGraph.SingleSourceShortestPathDomains, From GrB.Number](A GrB.Matrix[From], n int) (T GrB.Matrix[To], err error) {
	defer GrB.CheckErrors(&err)

	var from From
	switch any(from).(type) {
	case int, int32, int64, float32, float64:
		GrB.OK(GrB.MatrixApply(A, nil, nil, GrB.Abs[From](), A, nil))
	}
	mx := uint64(255)
	if fmx := uint64(GrB.Maximum[From]()); fmx < mx {
		mx = fmx
	}
	switch any(from).(type) {
	case int, int32, int64, uint, uint32, uint64:
		GrB.OK(GrB.MatrixApplyBinaryOp2nd(A, nil, nil, GrB.Max[From](), A, 1, nil))
		GrB.OK(GrB.MatrixApplyBinaryOp2nd(A, nil, nil, GrB.Min[From](), A, From(mx), nil))
	case float32, float64:
		emax, e := GrB.MatrixReduce(GrB.MaxMonoid[From](), A, nil)
		GrB.OK(e)
		emax = From(float64(mx) / float64(emax))
		GrB.OK(GrB.MatrixApplyBinaryOp2nd(A, nil, nil, GrB.Times[From](), A, emax, nil))
		GrB.OK(GrB.MatrixApplyBinaryOp2nd(A, nil, nil, GrB.Max[From](), A, 1, nil))
	default:
		T, err = GrB.MatrixNew[To](n, n)
		GrB.OK(err)
		Tf := GrB.MatrixView[float64, To](T)
		Af := GrB.MatrixView[float64, From](A)
		GrB.OK(GrB.MatrixApply(Tf, nil, nil, GrB.Abs[float64](), Af, nil))
		GrB.OK(GrB.MatrixApplyBinaryOp2nd(Tf, nil, nil, GrB.Max[float64](), Tf, 0.1, nil))
		GrB.OK(A.Free())
		return T, nil
	}
	return GrB.MatrixView[To, From](A), nil
}

func runSingleSourceShortestPathTypes[D LAGraph.SingleSourceShortestPathDomains](A GrB.Matrix[D], n int) (err error) {
	defer GrB.CheckErrors(&err)
	G := LAGraph.New(A, LAGraph.AdjacencyDirected)
	GrB.OK(G.Check())
	GrB.OK(G.CachedEMin())
	deltas := [...]int32{30, 100, 50000}
	var step int
	if n > 100 {
		step = 3 * n / 4
	} else {
		step = n/4 + 1
	}
	for src := 0; src < n; src += step {
		for _, delta := range deltas {
			pathLength, e := LAGraph.SingleSourceShortestPath(G, src, D(delta))
			GrB.OK(e)
			GrB.OK(checkSSSP(pathLength, G, src))
			GrB.OK(pathLength.Free())
		}
	}
	return G.Delete()
}

func TestSingleSourceShortestPathTypes(t *testing.T) {
	try := func(err error) {
		if err != nil {
			t.Error(err)
		}
	}
	for _, aname := range ssspFiles {
		f, err := os.Open(filepath.Join("testdata", aname))
		try(err)
		A, err := MatrixMarket.Read[float64](f)
		try(err)
		try(f.Close())
		n, ncols, err := A.Size()
		try(err)
		if n != ncols {
			t.Fail()
		}
		if typ, ok, err := A.Type(); err != nil {
			t.Error(err)
		} else if !ok {
			t.Fail()
		} else {
			switch typ {
			case GrB.Int:
				T, err := prepareSSSPTypes[int, int](GrB.MatrixView[int, float64](A), n)
				try(err)
				try(runSingleSourceShortestPathTypes(T, n))
			case GrB.Int8:
				T, err := prepareSSSPTypes[float64, int8](GrB.MatrixView[int8, float64](A), n)
				try(err)
				try(runSingleSourceShortestPathTypes(T, n))
			case GrB.Int16:
				T, err := prepareSSSPTypes[float64, int16](GrB.MatrixView[int16, float64](A), n)
				try(err)
				try(runSingleSourceShortestPathTypes(T, n))
			case GrB.Int32:
				T, err := prepareSSSPTypes[int32, int32](GrB.MatrixView[int32, float64](A), n)
				try(err)
				try(runSingleSourceShortestPathTypes(T, n))
			case GrB.Int64:
				T, err := prepareSSSPTypes[int64, int64](GrB.MatrixView[int64, float64](A), n)
				try(err)
				try(runSingleSourceShortestPathTypes(T, n))
			case GrB.Uint:
				T, err := prepareSSSPTypes[uint, uint](GrB.MatrixView[uint, float64](A), n)
				try(err)
				try(runSingleSourceShortestPathTypes(T, n))
			case GrB.Uint8:
				T, err := prepareSSSPTypes[float64, uint8](GrB.MatrixView[uint8, float64](A), n)
				try(err)
				try(runSingleSourceShortestPathTypes(T, n))
			case GrB.Uint16:
				T, err := prepareSSSPTypes[float64, uint16](GrB.MatrixView[uint16, float64](A), n)
				try(err)
				try(runSingleSourceShortestPathTypes(T, n))
			case GrB.Uint32:
				T, err := prepareSSSPTypes[uint32, uint32](GrB.MatrixView[uint32, float64](A), n)
				try(err)
				try(runSingleSourceShortestPathTypes(T, n))
			case GrB.Uint64:
				T, err := prepareSSSPTypes[uint64, uint64](GrB.MatrixView[uint64, float64](A), n)
				try(err)
				try(runSingleSourceShortestPathTypes(T, n))
			case GrB.Float32:
				T, err := prepareSSSPTypes[float32, float32](GrB.MatrixView[float32, float64](A), n)
				try(err)
				try(runSingleSourceShortestPathTypes(T, n))
			case GrB.Float64:
				T, err := prepareSSSPTypes[float64, float64](A, n)
				try(err)
				try(runSingleSourceShortestPathTypes(T, n))
			default:
				t.Fail()
			}
		}
	}
}
