package LAGraph

import (
	"errors"
	"fmt"
	"github.com/intel/forGoParallel/parallel"
	"github.com/intel/forGraphBLASGo/GrB"
	"math/rand"
	"reflect"
	"sort"
)

const (
	VersionMajor = 1
	VersionMinor = 0
)

const Unknown = -1

type Kind int

const (
	AdjacencyUndirected Kind = iota
	AdjacencyDirected
	KindUnknown Kind = Unknown
)

func (kind Kind) String() string {
	switch kind {
	case AdjacencyUndirected:
		return "undirected"
	case AdjacencyDirected:
		return "directed"
	case KindUnknown:
		return "unknown"
	}
	panic("invalid kind")
}

type Boolean int

const (
	False Boolean = iota
	True
	BooleanUnknown Boolean = Unknown
)

func (b Boolean) String() string {
	switch b {
	case False:
		return "false"
	case True:
		return "true"
	case BooleanUnknown:
		return "unknown"
	}
	panic("invalid boolean")
}

type State int

const (
	Value State = iota
	Bound
	StateUnknown State = Unknown
)

func (state State) String() string {
	switch state {
	case Value:
		return "value"
	case Bound:
		return "bound"
	case StateUnknown:
		return "unknown"
	}
	panic("invalid state")
}

type Graph[D GrB.Predefined] struct {
	A    GrB.Matrix[D]
	Kind Kind

	AT                   GrB.Matrix[D]
	OutDegree, InDegree  GrB.Vector[int]
	IsSymmetricStructure Boolean
	NSelfEdges           int
	EMin                 GrB.Scalar[D]
	EMinState            State
	EMax                 GrB.Scalar[D]
	EMaxState            State
}

func Init(mode GrB.Mode) error {
	return GrB.Init(mode)
}

func PlusFirst[D GrB.Number]() GrB.Semiring[D, D, D] {
	return GrB.PlusFirst[D]()
}

func PlusSecond[D GrB.Number]() GrB.Semiring[D, D, D] {
	return GrB.PlusSecond[D]()
}

func PlusOne[D GrB.Number]() GrB.Semiring[D, D, D] {
	return GrB.PlusOneb[D]()
}

func AnyOne[D GrB.Number]() GrB.Semiring[D, D, D] {
	return GrB.AnyOneb[D]()
}

func Finalize() error {
	return GrB.Finalize()
}

func New[D GrB.Predefined](A GrB.Matrix[D], kind Kind) *Graph[D] {
	g := Graph[D]{
		A:                    A,
		Kind:                 Unknown,
		IsSymmetricStructure: Unknown,
		NSelfEdges:           Unknown,
		EMinState:            Unknown,
		EMaxState:            Unknown,
	}
	if A.Valid() {
		g.Kind = kind
		if kind == AdjacencyUndirected {
			g.IsSymmetricStructure = True
		}
	}
	return &g
}

func (G *Graph[D]) Delete() (err error) {
	if err = G.DeleteCached(); err != nil {
		return
	}
	return G.A.Free()
}

func (G *Graph[D]) DeleteCached() (err error) {
	defer GrB.CheckErrors(&err)
	GrB.OK(G.AT.Free())
	GrB.OK(G.OutDegree.Free())
	GrB.OK(G.InDegree.Free())
	GrB.OK(G.EMin.Free())
	GrB.OK(G.EMax.Free())
	if G.Kind == AdjacencyUndirected {
		G.IsSymmetricStructure = True
	} else {
		G.IsSymmetricStructure = Unknown
	}
	G.EMinState = Unknown
	G.EMaxState = Unknown
	G.NSelfEdges = Unknown
	return
}

func (G *Graph[D]) CachedAT() (cacheOk bool, err error) {
	A := G.A
	if G.AT.Valid() {
		return true, nil
	}
	if G.Kind == AdjacencyUndirected {
		return false, nil
	}
	defer GrB.CheckErrors(&err)
	nrows, ncols, err := A.Size()
	GrB.OK(err)
	AT, err := GrB.MatrixNew[D](ncols, nrows)
	GrB.OK(err)
	GrB.OK(GrB.Transpose(AT, nil, nil, A, nil))
	G.AT = AT
	return true, nil
}

func (G *Graph[D]) CachedIsSymmetricStructure() (err error) {
	if G.Kind == AdjacencyUndirected {
		G.IsSymmetricStructure = True
		return nil
	}
	if G.IsSymmetricStructure != Unknown {
		return nil
	}
	defer GrB.CheckErrors(&err)
	A := G.A
	n, ncols, err := A.Size()
	GrB.OK(err)
	if n != ncols {
		G.IsSymmetricStructure = False
		return
	}
	if !G.AT.Valid() {
		_, err = G.CachedAT()
		GrB.OK(err)
	}
	C, err := GrB.MatrixNew[bool](n, n)
	GrB.OK(err)
	defer func() {
		GrB.OK(C.Free())
	}()
	GrB.OK(GrB.MatrixEWiseMultBinaryOp(
		C, nil, nil,
		GrB.Oneb[bool](),
		GrB.MatrixView[bool, D](A),
		GrB.MatrixView[bool, D](G.AT),
		nil,
	))
	nvals1, err := C.Nvals()
	GrB.OK(err)
	nvals2, err := A.Nvals()
	GrB.OK(err)
	if nvals1 == nvals2 {
		G.IsSymmetricStructure = True
	} else {
		G.IsSymmetricStructure = False
	}
	return nil
}

func (G *Graph[D]) CachedOutDegree() (err error) {
	if G.OutDegree.Valid() {
		return
	}
	defer GrB.CheckErrors(&err)
	A := G.A
	nrows, ncols, err := A.Size()
	GrB.OK(err)
	outDegree, err := GrB.VectorNew[int](nrows)
	GrB.OK(err)
	defer func() {
		if err != nil {
			_ = outDegree.Free()
		}
	}()
	x, err := GrB.VectorNew[int](ncols)
	GrB.OK(err)
	defer func() {
		GrB.OK(x.Free())
	}()
	GrB.OK(GrB.VectorAssignConstant(x, nil, nil, 0, GrB.All(ncols), nil))
	GrB.OK(GrB.MxV(outDegree, nil, nil, PlusOne[int](), GrB.MatrixView[int, D](A), x, nil))
	G.OutDegree = outDegree
	return nil
}

func (G *Graph[D]) CachedInDegree() (cacheOk bool, err error) {
	if G.InDegree.Valid() {
		return true, nil
	}
	if G.Kind == AdjacencyUndirected {
		return
	}
	defer GrB.CheckErrors(&err)
	A := G.A
	AT := G.AT
	nrows, ncols, err := A.Size()
	GrB.OK(err)
	inDegree, err := GrB.VectorNew[int](ncols)
	GrB.OK(err)
	defer func() {
		if err != nil {
			_ = inDegree.Free()
		}
	}()
	x, err := GrB.VectorNew[int](nrows)
	GrB.OK(err)
	defer func() {
		GrB.OK(x.Free())
	}()
	GrB.OK(GrB.VectorAssignConstant(x, nil, nil, 0, GrB.All(nrows), nil))
	if AT.Valid() {
		GrB.OK(GrB.MxV(inDegree, nil, nil, PlusOne[int](), GrB.MatrixView[int, D](AT), x, nil))
	} else {
		GrB.OK(GrB.MxV(inDegree, nil, nil, PlusOne[int](), GrB.MatrixView[int, D](A), x, GrB.DescT0))
	}
	G.InDegree = inDegree
	return true, nil
}

func (G *Graph[D]) CachedNSelfEdges() (err error) {
	if G.NSelfEdges != Unknown {
		return
	}
	G.NSelfEdges, err = nselfEdges(G.A)
	return
}

func (G *Graph[D]) CachedEMin() (err error) {
	if G.EMin.Valid() {
		return
	}
	G.EMinState = Unknown
	var monoid GrB.Monoid[D]
	switch m := any(&monoid).(type) {
	case *GrB.Monoid[bool]:
		*m = GrB.LandMonoidBool
	case *GrB.Monoid[int]:
		*m = GrB.MinMonoid[int]()
	case *GrB.Monoid[int8]:
		*m = GrB.MinMonoid[int8]()
	case *GrB.Monoid[int16]:
		*m = GrB.MinMonoid[int16]()
	case *GrB.Monoid[int32]:
		*m = GrB.MinMonoid[int32]()
	case *GrB.Monoid[int64]:
		*m = GrB.MinMonoid[int64]()
	case *GrB.Monoid[uint]:
		*m = GrB.MinMonoid[uint]()
	case *GrB.Monoid[uint8]:
		*m = GrB.MinMonoid[uint8]()
	case *GrB.Monoid[uint16]:
		*m = GrB.MinMonoid[uint16]()
	case *GrB.Monoid[uint32]:
		*m = GrB.MinMonoid[uint32]()
	case *GrB.Monoid[uint64]:
		*m = GrB.MinMonoid[uint64]()
	case *GrB.Monoid[float32]:
		*m = GrB.MinMonoid[float32]()
	case *GrB.Monoid[float64]:
		*m = GrB.MinMonoid[float64]()
	default:
		return GrB.NotImplemented
	}
	if G.EMin, err = GrB.ScalarNew[D](); err != nil {
		return
	}
	if err = GrB.MatrixReduceMonoidScalar(G.EMin, nil, monoid, G.A, nil); err != nil {
		return
	}
	G.EMinState = Value
	return
}

func (G *Graph[D]) CachedEMax() (err error) {
	if G.EMax.Valid() {
		return
	}
	G.EMaxState = Unknown
	var monoid GrB.Monoid[D]
	switch m := any(&monoid).(type) {
	case *GrB.Monoid[bool]:
		*m = GrB.LorMonoidBool
	case *GrB.Monoid[int]:
		*m = GrB.MaxMonoid[int]()
	case *GrB.Monoid[int8]:
		*m = GrB.MaxMonoid[int8]()
	case *GrB.Monoid[int16]:
		*m = GrB.MaxMonoid[int16]()
	case *GrB.Monoid[int32]:
		*m = GrB.MaxMonoid[int32]()
	case *GrB.Monoid[int64]:
		*m = GrB.MaxMonoid[int64]()
	case *GrB.Monoid[uint]:
		*m = GrB.MaxMonoid[uint]()
	case *GrB.Monoid[uint8]:
		*m = GrB.MaxMonoid[uint8]()
	case *GrB.Monoid[uint16]:
		*m = GrB.MaxMonoid[uint16]()
	case *GrB.Monoid[uint32]:
		*m = GrB.MaxMonoid[uint32]()
	case *GrB.Monoid[uint64]:
		*m = GrB.MaxMonoid[uint64]()
	case *GrB.Monoid[float32]:
		*m = GrB.MaxMonoid[float32]()
	case *GrB.Monoid[float64]:
		*m = GrB.MaxMonoid[float64]()
	default:
		return GrB.NotImplemented
	}
	if G.EMax, err = GrB.ScalarNew[D](); err != nil {
		return
	}
	if err = GrB.MatrixReduceMonoidScalar(G.EMax, nil, monoid, G.A, nil); err != nil {
		return
	}
	G.EMaxState = Value
	return
}

func (G *Graph[D]) DeleteSelfEdges() (err error) {
	if G.NSelfEdges == 0 {
		return
	}
	isSymmetricStructure := G.IsSymmetricStructure
	if err = G.DeleteCached(); err != nil {
		return
	}
	G.IsSymmetricStructure = isSymmetricStructure
	if err = GrB.MatrixSelect(G.A, nil, nil, GrB.Offdiag[D](), G.A, 0, nil); err != nil {
		return
	}
	G.NSelfEdges = 0
	return
}

func (G *Graph[D]) Check() (err error) {
	defer GrB.CheckErrors(&err)
	A := G.A
	kind := G.Kind
	var nrows, ncols int
	switch kind {
	case AdjacencyUndirected, AdjacencyDirected:
		nrows, ncols, err = A.Size()
		GrB.OK(err)
		if nrows != ncols {
			return errors.New("adjacency matrix must be square")
		}
	}
	format, err := A.GetLayout()
	GrB.OK(err)
	if format != GrB.ByRow {
		return errors.New("only by-row format supported")
	}
	AT := G.AT
	if AT.Valid() {
		nrows2, ncols2, e := AT.Size()
		GrB.OK(e)
		if nrows != ncols2 || ncols != nrows2 {
			return errors.New("G.AT matrix has the wrong dimensions")
		}
		format, err = AT.GetLayout()
		GrB.OK(err)
		if format != GrB.ByRow {
			return errors.New("only by-row format supported")
		}
	}
	outDegree := G.OutDegree
	if outDegree.Valid() {
		n, e := outDegree.Size()
		GrB.OK(e)
		if n != nrows {
			return errors.New("OutDegree invalid size")
		}
	}
	inDegree := G.InDegree
	if inDegree.Valid() {
		n, e := inDegree.Size()
		GrB.OK(e)
		if n != ncols {
			return errors.New("InDegree invalid size")
		}
	}
	return nil
}

func MatrixStructure[D any](A GrB.Matrix[D]) (C GrB.Matrix[bool], err error) {
	defer GrB.CheckErrors(&err)
	nrows, ncols, err := A.Size()
	GrB.OK(err)
	C, err = GrB.MatrixNew[bool](nrows, ncols)
	GrB.OK(err)
	defer func() {
		if err != nil {
			_ = C.Free()
		}
	}()
	GrB.OK(GrB.MatrixAssignConstant(C, A.AsMask(), nil, true, GrB.All(nrows), GrB.All(ncols), GrB.DescS))
	return
}

func VectorStructure[D any](u GrB.Vector[D]) (w GrB.Vector[bool], err error) {
	defer GrB.CheckErrors(&err)
	n, err := u.Size()
	GrB.OK(err)
	w, err = GrB.VectorNew[bool](n)
	GrB.OK(err)
	defer func() {
		if err != nil {
			_ = w.Free()
		}
	}()
	GrB.OK(GrB.VectorAssignConstant(w, u.AsMask(), nil, true, GrB.All(n), GrB.DescS))
	return
}

func typename[D any]() string {
	var d D
	return reflect.TypeOf(d).String()
}

func (G *Graph[D]) Print(printLevel GrB.PrintLevel) (err error) {
	defer GrB.CheckErrors(&err)

	GrB.OK(G.Check())

	if printLevel == 0 {
		return
	}

	A := G.A
	kind := G.Kind
	n, err := A.Nrows()
	GrB.OK(err)
	nvals, err := A.Nvals()
	GrB.OK(err)
	prln := func(a ...any) {
		_, err = fmt.Println(a...)
		GrB.OK(err)
	}
	pr := func(a ...any) {
		_, err = fmt.Print(a...)
		GrB.OK(err)
	}
	prln("Graph: kind:", kind, "nodes:", n, "entries:", nvals, "type:", typename[D]())
	pr("  structural symmetry: ")
	switch G.IsSymmetricStructure {
	case False:
		pr("unsymmetric")
	case True:
		pr("symmetric")
	default:
		pr("unknown")
	}
	if G.NSelfEdges >= 0 {
		pr("  self-edges:", G.NSelfEdges)
	}
	prln()
	pr("  adjacency matrix: ")
	GrB.OK(MatrixPrint(A, printLevel))
	AT := G.AT
	if AT.Valid() {
		pr("  adjacency matrix transposed: ")
		GrB.OK(MatrixPrint(AT, printLevel))
	}
	outDegree := G.OutDegree
	if outDegree.Valid() {
		pr("  out degree: ")
		GrB.OK(VectorPrint(outDegree, printLevel))
	}
	inDegree := G.InDegree
	if inDegree.Valid() {
		pr("  in degree: ")
		GrB.OK(VectorPrint(inDegree, printLevel))
	}
	return
}

const lgShortLen = 30

func MatrixPrint[D GrB.Predefined](A GrB.Matrix[D], printLevel GrB.PrintLevel) (err error) {
	if printLevel <= 0 {
		return
	}
	defer GrB.CheckErrors(&err)
	nrows, ncols, err := A.Size()
	GrB.OK(err)
	nvals, err := A.Nvals()
	GrB.OK(err)
	_, err = fmt.Println(typename[D](), "matrix:", nrows, "by", ncols, "entries:", nvals)
	GrB.OK(err)
	if printLevel <= 1 {
		return
	}
	var I, J []int
	var X []D
	GrB.OK(A.ExtractTuples(&I, &J, &X))
	summary := (printLevel == 2 || printLevel == 4) && (nvals > lgShortLen)
	for k := range nvals {
		_, err = fmt.Printf("   (%v, %v)   %v\n", I[k], J[k], X[k])
		GrB.OK(err)
		if summary && k > lgShortLen {
			_, err = fmt.Println("   ...")
			GrB.OK(err)
			break
		}
	}
	return
}

func VectorPrint[D GrB.Predefined](v GrB.Vector[D], printLevel GrB.PrintLevel) (err error) {
	if printLevel <= 0 {
		return
	}
	defer GrB.CheckErrors(&err)
	n, err := v.Size()
	GrB.OK(err)
	nvals, err := v.Nvals()
	GrB.OK(err)
	_, err = fmt.Println(typename[D](), "vector: n:", n, "entries:", nvals)
	GrB.OK(err)
	if printLevel <= 1 {
		return
	}
	var I []int
	var X []D
	GrB.OK(v.ExtractTuples(&I, &X))
	summary := (printLevel == 2 || printLevel == 4) && nvals > lgShortLen
	for k := range nvals {
		_, err = fmt.Printf("   (%v)   %v\n", I[k], X[k])
		GrB.OK(err)
		if summary && k > lgShortLen {
			_, err = fmt.Println("   ...")
			GrB.OK(err)
			break
		}
	}
	return
}

func MatrixIsEqual[D GrB.Predefined | GrB.Complex](A, B GrB.Matrix[D]) (result bool, err error) {
	if A == B {
		return true, nil
	}
	return MatrixIsEqualOp(A, B, GrB.Eq[D]())
}

func MatrixIsEqualOp[DA, DB any](A GrB.Matrix[DA], B GrB.Matrix[DB], op GrB.BinaryOp[bool, DA, DB]) (result bool, err error) {
	if avalid, bvalid := A.Valid(), B.Valid(); !avalid || !bvalid {
		return !avalid && !bvalid, nil
	}
	defer GrB.CheckErrors(&err)
	nrows1, ncols1, err := A.Size()
	GrB.OK(err)
	nrows2, ncols2, err := B.Size()
	GrB.OK(err)
	if nrows1 != nrows2 || ncols1 != ncols2 {
		return false, nil
	}
	nvals1, err := A.Nvals()
	GrB.OK(err)
	nvals2, err := B.Nvals()
	GrB.OK(err)
	if nvals1 != nvals2 {
		return false, nil
	}
	C, err := GrB.MatrixNew[bool](nrows1, ncols1)
	GrB.OK(err)
	defer func() {
		GrB.OK(C.Free())
	}()
	GrB.OK(GrB.MatrixEWiseMultBinaryOp(C, nil, nil, op, A, B, nil))
	nvals, err := C.Nvals()
	GrB.OK(err)
	if nvals != nvals1 {
		return false, nil
	}
	return GrB.MatrixReduce(GrB.LandMonoidBool, C, nil)
}

func VectorIsEqual[D GrB.Predefined | GrB.Complex](u, v GrB.Vector[D]) (result bool, err error) {
	if u == v {
		return true, nil
	}
	return VectorIsEqualOp(u, v, GrB.Eq[D]())
}

func VectorIsEqualOp[Du, Dv any](u GrB.Vector[Du], v GrB.Vector[Dv], op GrB.BinaryOp[bool, Du, Dv]) (result bool, err error) {
	if avalid, bvalid := u.Valid(), v.Valid(); !avalid || !bvalid {
		return !avalid && !bvalid, nil
	}
	defer GrB.CheckErrors(&err)
	n1, err := u.Size()
	GrB.OK(err)
	n2, err := v.Size()
	GrB.OK(err)
	if n1 != n2 {
		return
	}
	nvals1, err := u.Nvals()
	GrB.OK(err)
	nvals2, err := v.Nvals()
	GrB.OK(err)
	if nvals1 != nvals2 {
		return
	}
	w, err := GrB.VectorNew[bool](n1)
	GrB.OK(err)
	defer func() {
		GrB.OK(w.Free())
	}()
	GrB.OK(GrB.VectorEWiseMultBinaryOp(w, nil, nil, op, u, v, nil))
	nvals, err := w.Nvals()
	GrB.OK(err)
	if nvals != nvals1 {
		return
	}
	return GrB.VectorReduce(GrB.LandMonoidBool, w, nil)
}

func (G *Graph[D]) SortByDegree(byOut, ascending bool) (permutationVector []int, err error) {
	defer GrB.CheckErrors(&err)

	GrB.OK(G.Check())

	var Degree GrB.Vector[int]
	if G.Kind == AdjacencyUndirected || (G.Kind == AdjacencyDirected && G.IsSymmetricStructure == True) {
		Degree = G.OutDegree
	} else if byOut {
		Degree = G.OutDegree
	} else {
		Degree = G.InDegree
	}
	if !Degree.Valid() {
		panic("degree unknown")
	}
	n, err := Degree.Size()
	GrB.OK(err)
	p := make([]int, n)
	d := make([]int, n)
	parallel.Range(0, n, 0, func(low, high int) {
		for i := low; i < high; i++ {
			p[i] = i
		}
	})
	var w0, w1 []int
	GrB.OK(Degree.ExtractTuples(&w0, &w1))
	if ascending {
		parallel.Range(0, n, 0, func(low, high int) {
			for i := low; i < high; i++ {
				d[w0[i]] = w1[i]
			}
		})
	} else {
		parallel.Range(0, n, 0, func(low, high int) {
			for i := low; i < high; i++ {
				d[w0[i]] = -w1[i]
			}
		})
	}
	twoSliceSort(d, p)
	return p, nil
}

func (G *Graph[D]) SampleDegree(byOut bool, nSamples int, rnd *rand.Rand) (sampleMean, sampleMedian float64, err error) {
	defer GrB.CheckErrors(&err)
	nSamples = max(nSamples, 1)

	GrB.OK(G.Check())
	var Degree GrB.Vector[int]
	if G.Kind == AdjacencyUndirected || (G.Kind == AdjacencyDirected && G.IsSymmetricStructure == True) {
		Degree = G.OutDegree
	} else if byOut {
		Degree = G.OutDegree
	} else {
		Degree = G.InDegree
	}
	if !Degree.Valid() {
		panic("degree unknown")
	}
	samples := make([]int, nSamples)
	n, err := Degree.Size()
	GrB.OK(err)
	dsum := 0
	for k := range nSamples {
		i := rnd.Intn(n)
		d, _, e := Degree.ExtractElement(i)
		GrB.OK(e)
		samples[k] = d
		dsum += d
	}
	sampleMean = float64(dsum) / float64(nSamples)
	sort.Ints(samples)
	sampleMedian = float64(samples[nSamples/2])
	return
}
