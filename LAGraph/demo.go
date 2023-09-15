package LAGraph

import (
	"encoding/binary"
	"github.com/intel/forGraphBLASGo/GrB"
	"github.com/intel/forLAGraphGo/LAGraph/MatrixMarket"
	"io"
	"log"
	"os"
	"path/filepath"
	"unsafe"
)

/*
#if defined ( __linux__ )
#include <malloc.h>

void demo_init() {
	mallopt(M_MMAP_MAX, 0);
	mallopt(M_TRIM_THRESHOLD, -1);
	mallopt(M_TOP_PAD,16*1024*1024);
}
#else
void demo_init() {}
#endif
*/
import "C"
import (
	"errors"
	"math/rand"
)

func DemoInit() error {
	C.demo_init()
	err := Init(GrB.NonBlocking)
	if err != nil {
		return err
	}
	log.Print(
		GrB.SuiteSparseImplementationName, " ",
		GrB.SuiteSparseImplementationMajor, ".",
		GrB.SuiteSparseImplementationMinor, ".",
		GrB.SuiteSparseImplementationSub,
	)
	if omp, err := GrB.GlobalGetOpenMP(); err != nil {
		return err
	} else {
		log.Println("OpenMP:", omp)
	}
	return nil
}

func binread[D GrB.Number](f *os.File) (matrix GrB.Matrix[D], err error) {
	defer GrB.CheckErrors(&err)

	_, err = f.Seek(512, 0)
	GrB.OK(err)

	var fmt int32 = -999
	var kind, typecode int32
	var hyper float64 = -999
	var nrows, ncols, nvec, nvals, typesize uint64
	var nonempty int64
	GrB.OK(binary.Read(f, binary.LittleEndian, &fmt))
	GrB.OK(binary.Read(f, binary.LittleEndian, &kind))
	GrB.OK(binary.Read(f, binary.LittleEndian, &hyper))
	GrB.OK(binary.Read(f, binary.LittleEndian, &nrows))
	GrB.OK(binary.Read(f, binary.LittleEndian, &ncols))
	GrB.OK(binary.Read(f, binary.LittleEndian, &nonempty))
	GrB.OK(binary.Read(f, binary.LittleEndian, &nvec))
	GrB.OK(binary.Read(f, binary.LittleEndian, &nvals))
	GrB.OK(binary.Read(f, binary.LittleEndian, &typecode))
	GrB.OK(binary.Read(f, binary.LittleEndian, &typesize))

	iso := false
	if kind > 100 {
		iso = true
		kind -= 100
	}

	isHyper := kind == 1
	isSparse := kind == 0 || kind == int32(GrB.Sparse)
	isBitmap := kind == int32(GrB.Bitmap)
	isFull := kind == int32(GrB.Full)

	switch typecode {
	case 0:
		m, e := GrB.MatrixNew[bool](int(nrows), int(ncols))
		GrB.OK(e)
		matrix = GrB.MatrixView[D, bool](m)
	case 1:
		m, e := GrB.MatrixNew[int8](int(nrows), int(ncols))
		GrB.OK(e)
		matrix = GrB.MatrixView[D, int8](m)
	case 2:
		m, e := GrB.MatrixNew[int16](int(nrows), int(ncols))
		GrB.OK(e)
		matrix = GrB.MatrixView[D, int16](m)
	case 3:
		m, e := GrB.MatrixNew[int32](int(nrows), int(ncols))
		GrB.OK(e)
		matrix = GrB.MatrixView[D, int32](m)
	case 4:
		m, e := GrB.MatrixNew[int64](int(nrows), int(ncols))
		GrB.OK(e)
		matrix = GrB.MatrixView[D, int64](m)
	case 5:
		m, e := GrB.MatrixNew[uint8](int(nrows), int(ncols))
		GrB.OK(e)
		matrix = GrB.MatrixView[D, uint8](m)
	case 6:
		m, e := GrB.MatrixNew[uint16](int(nrows), int(ncols))
		GrB.OK(e)
		matrix = GrB.MatrixView[D, uint16](m)
	case 7:
		m, e := GrB.MatrixNew[uint32](int(nrows), int(ncols))
		GrB.OK(e)
		matrix = GrB.MatrixView[D, uint32](m)
	case 8:
		m, e := GrB.MatrixNew[uint64](int(nrows), int(ncols))
		GrB.OK(e)
		matrix = GrB.MatrixView[D, uint64](m)
	case 9:
		m, e := GrB.MatrixNew[float32](int(nrows), int(ncols))
		GrB.OK(e)
		matrix = GrB.MatrixView[D, float32](m)
	case 10:
		m, e := GrB.MatrixNew[float64](int(nrows), int(ncols))
		GrB.OK(e)
		matrix = GrB.MatrixView[D, float64](m)
	default:
		err = GrB.NotImplemented
		return
	}

	GrB.OK(matrix.SetHyperSwitch(hyper))

	var ap, ai, ah, ab, ax GrB.SystemSlice[byte]
	var axLen int

	indexSize := int(unsafe.Sizeof(uint64(0)))
	switch {
	case isHyper:
		apLen := int(nvec + 1)
		ahLen := int(nvec)
		aiLen := int(nvals)
		axLen = int(nvals)
		ap = GrB.MakeSystemSlice[byte](apLen * indexSize)
		ah = GrB.MakeSystemSlice[byte](ahLen * indexSize)
		ai = GrB.MakeSystemSlice[byte](aiLen * indexSize)
		defer func() {
			if err != nil {
				ai.Free()
				ah.Free()
				ap.Free()
			}
		}()
	case isSparse:
		apLen := int(nvec + 1)
		aiLen := int(nvals)
		axLen = int(nvals)
		ap = GrB.MakeSystemSlice[byte](apLen * indexSize)
		ai = GrB.MakeSystemSlice[byte](aiLen * indexSize)
		defer func() {
			if err != nil {
				ai.Free()
				ap.Free()
			}
		}()
	case isBitmap:
		axLen = int(nrows * ncols)
		ab = GrB.MakeSystemSlice[byte](int(nrows * ncols))
		defer func() {
			if err != nil {
				ab.Free()
			}
		}()
	case isFull:
		axLen = int(nrows * ncols)
	default:
		panic("unreachable code")
	}
	var axSize int
	if iso {
		axSize = int(typesize)
	} else {
		axSize = axLen * int(typesize)
	}
	ax = GrB.MakeSystemSlice[byte](axSize)
	defer func() {
		if err != nil {
			ax.Free()
		}
	}()

	{
		tryRead := func(bytes GrB.SystemSlice[byte]) {
			_, err = io.ReadFull(f, bytes.UnsafeSlice())
			GrB.OK(err)
		}
		switch {
		case isHyper:
			tryRead(ap)
			tryRead(ah)
			tryRead(ai)
		case isSparse:
			tryRead(ap)
			tryRead(ai)
		case isBitmap:
			tryRead(ab)
		}
		tryRead(ax)
	}

	switch fmt := GrB.Layout(fmt); {
	case fmt == GrB.ByCol && isHyper:
		err = matrix.PackHyperCSCBytes(&ap, &ah, &ai, &ax, iso, int(nvec), false, nil)
	case fmt == GrB.ByRow && isHyper:
		err = matrix.PackHyperCSRBytes(&ap, &ah, &ai, &ax, iso, int(nvec), false, nil)
	case fmt == GrB.ByCol && isSparse:
		err = matrix.PackCSCBytes(&ap, &ai, &ax, iso, false, nil)
	case fmt == GrB.ByRow && isSparse:
		err = matrix.PackCSRBytes(&ap, &ai, &ax, iso, false, nil)
	case fmt == GrB.ByCol && isBitmap:
		err = matrix.PackBitmapCBytes(&ab, &ax, iso, int(nvals), nil)
	case fmt == GrB.ByRow && isBitmap:
		err = matrix.PackBitmapRBytes(&ab, &ax, iso, int(nvals), nil)
	case fmt == GrB.ByCol && isFull:
		err = matrix.PackFullCBytes(&ax, iso, nil)
	case fmt == GrB.ByRow && isFull:
		err = matrix.PackFullRBytes(&ax, iso, nil)
	default:
		panic("unreachable code")
	}
	return
}

func ReadProblem[D GrB.Number](computeSourceNodes, makeSymmetric, removeSelfEdges, structural, forceType, ensurePositive bool, args []string) (G *Graph[D], srcNodes GrB.Matrix[int], err error) {
	defer GrB.CheckErrors(&err)
	try := func(f func() error) {
		GrB.OK(f())
	}

	var RA GrB.Matrix[int64]
	var srcNodesDone bool
	if len(args) < 1 {
		log.Fatalln("Missing input file.")
	}
	filename := args[0]
	f, err := os.Open(filename)
	GrB.OK(err)
	isBinary := filepath.Ext(filename) == ".grb"
	if isBinary {
		RA, err = binread[int64](f)
		if err != nil {
			_ = f.Close()
			return
		}
	} else {
		RA, err = MatrixMarket.Read[int64](f)
		if err != nil {
			_ = f.Close()
			return
		}
	}
	defer try(RA.Free)
	GrB.OK(f.Close())

	if len(args) > 1 {
		filename = args[1]
		if filename[0] != '-' {
			f, err = os.Open(filename)
			GrB.OK(err)
			if srcNodes, err = MatrixMarket.Read[int](f); err != nil {
				_ = f.Close()
				return
			}
			defer func() {
				if err != nil {
					_ = srcNodes.Free()
				}
			}()
			GrB.OK(f.Close())
			srcNodesDone = true
		}
	}

	n, ncols, err := RA.Size()
	GrB.OK(err)
	if n != ncols {
		err = errors.New("A must be square")
		return
	}

	var A GrB.Matrix[D]

	if structural {
		A2, e := GrB.MatrixNew[bool](n, n)
		GrB.OK(e)
		defer try(A2.Free)

		GrB.OK(GrB.MatrixAssignConstant(A2, RA.AsMask(), nil, true, GrB.All(n), GrB.All(n), GrB.DescS))
		GrB.OK(RA.Free())
		GrB.OK(A2.Wait(GrB.Materialize))
		A = GrB.MatrixView[D, bool](A2)
		A2 = GrB.Matrix[bool]{}
	} else if forceType {
		typ, ok, e := RA.Type()
		GrB.OK(e)
		if !ok {
			panic("unreachable code")
		}
		var d D
		if typ == GrB.TypeOf(d) {
			A = GrB.MatrixView[D, int64](RA)
			RA = GrB.Matrix[int64]{}
		} else {
			A, err = GrB.MatrixNew[D](n, n)
			GrB.OK(err)
			GrB.OK(GrB.MatrixApply(A, nil, nil, GrB.Identity[D](), GrB.MatrixView[D, int64](RA), nil))
			GrB.OK(A.Wait(GrB.Materialize))
			GrB.OK(RA.Free())
		}
	} else {
		A = GrB.MatrixView[D, int64](RA)
		RA = GrB.Matrix[int64]{}
	}

	AIsSymmetric := n == 134217726 || n == 134217728 // hacks for kron and urand

	var gKind Kind
	if AIsSymmetric {
		gKind = AdjacencyUndirected
	} else {
		gKind = AdjacencyDirected
	}

	G = New(A, gKind)

	if removeSelfEdges {
		GrB.OK(G.DeleteSelfEdges())
	}

	if !structural && ensurePositive {
		GrB.OK(GrB.MatrixSelect(G.A, nil, nil, GrB.Valuene[D](), G.A, 0, nil))
		GrB.OK(GrB.MatrixApply(G.A, nil, nil, GrB.Abs[D](), G.A, nil))
	}

	if !AIsSymmetric {
		GrB.OK(G.CachedIsSymmetricStructure())
		if G.IsSymmetricStructure == True && structural {
			G.Kind = AdjacencyUndirected
			GrB.OK(G.AT.Free())
		} else if makeSymmetric {
			OK, e := GrB.MatrixNew[bool](n, n)
			GrB.OK(e)
			defer try(OK.Free)

			GrB.OK(GrB.MatrixEWiseMultBinaryOp(OK, nil, nil, GrB.Eq[D](), G.A, G.AT, nil))

			nvals, e := G.A.Nvals()
			GrB.OK(e)
			oknvals, e := OK.Nvals()
			GrB.OK(e)
			sym := oknvals == nvals
			if sym {
				sym, e = GrB.MatrixReduce(GrB.LandMonoidBool, OK, nil)
				GrB.OK(e)
			}
			if !sym {
				GrB.OK(GrB.MatrixEWiseAddBinaryOp(G.A, nil, nil, GrB.Plus[D](), G.A, G.AT, nil))
			}
			G.Kind = AdjacencyUndirected
			G.IsSymmetricStructure = True
		}
	}

	const nSources = 64

	if computeSourceNodes && !srcNodesDone {
		srcNodes, err = GrB.MatrixNew[int](nSources, 1)
		GrB.OK(err)
		defer func() {
			if err != nil {
				_ = srcNodes.Free()
			}
		}()
		for k := 0; k < nSources; k++ {
			i := 1 + rand.Intn(n)
			GrB.OK(srcNodes.SetElement(i, k, 0))
		}
		GrB.OK(srcNodes.Wait(GrB.Materialize))
	}

	return G, srcNodes, nil
}
