package forLAGraphGo

import (
	"errors"
	GrB "github.com/intel/forGraphBLASGo"
	"github.com/intel/forLAGraphGo/MatrixMarket"
	"log"
	"math/rand"
	"os"
	"reflect"
)

func ReadProblem[T GrB.Number](computeSourceNodes, makeSymmetric, removeSelfEdges, structural, ensurePositive bool, args []string) (G *Graph[T], srcNodes *GrB.Matrix[int], functionErr error) {
	var RA *GrB.Matrix[int64]
	var srcNodesDone bool
	if len(args) < 1 {
		log.Fatalln("Missing input file.")
	}
	filename := args[0]
	log.Printf("Reading matrix market file: %v\n", filename)
	f, err := os.Open(filename)
	if err != nil {
		functionErr = err
		return
	}
	header, scanner, err := MatrixMarket.ReadHeader(f)
	if err != nil {
		_ = f.Close()
		functionErr = err
		return
	}
	RA, err = MatrixMarket.Read[int64](header, scanner)
	if err != nil {
		_ = f.Close()
		functionErr = err
		return
	}
	if err = f.Close(); err != nil {
		functionErr = err
		return
	}

	if len(args) > 1 {
		filename = args[1]
		if filename[0] != '-' {
			log.Printf("Sources: %v\n", filename)
			f, err = os.Open(filename)
			if err != nil {
				functionErr = err
				return
			}
			header, scanner, err = MatrixMarket.ReadHeader(f)
			if err != nil {
				_ = f.Close()
				functionErr = err
				return
			}
			srcNodes, err = MatrixMarket.Read[int](header, scanner)
			if err != nil {
				_ = f.Close()
				functionErr = err
				return
			}
			if err = f.Close(); err != nil {
				functionErr = err
				return
			}
			srcNodesDone = true
		}
	}

	n, ncols, err := RA.Size()
	if err != nil {
		functionErr = err
		return
	}
	if n != ncols {
		err = errors.New("A must be square")
		return
	}

	if structural {
		log.Println("make structural")
		GrB.MatrixAssignConstant(RA, RA.AsMask(), nil, 1, GrB.All(n), GrB.All(n), GrB.DescS)
	}

	var A *GrB.Matrix[T]
	switch reflect.ValueOf(T(0)).Kind() {
	case GrB.Int64:
		A = any(RA).(*GrB.Matrix[T])
	default:
		log.Println("convert element type")
		A, err = GrB.MatrixNew[T](n, n)
		if err != nil {
			functionErr = err
			return
		}
		if err = GrB.MatrixApply(A, nil, nil, func(x int64) T { return T(x) }, RA, nil); err != nil {
			functionErr = err
			return
		}
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
		log.Println("remove self edges")
		G.DeleteDiag()
	}

	if !structural && ensurePositive {
		log.Println("ensure positive")
		if err = GrB.MatrixSelect(G.A, nil, nil, GrB.ValueNE[T], G.A, 0, nil); err != nil {
			functionErr = err
			return
		}
		if err = GrB.MatrixApply(G.A, nil, nil, GrB.Abs[T], G.A, nil); err != nil {
			functionErr = err
			return
		}
	}

	if !AIsSymmetric {
		log.Println("G.PropertyASymmetricStructure()")
		G.PropertyASymmetricStructure()
		if G.AStructureIsSymmetric == True && structural {
			G.Kind = AdjacencyUndirected
			G.AT = nil
		} else if makeSymmetric {
			log.Println("make symmetric")
			var sym bool
			OK, err := GrB.MatrixNew[bool](n, n)
			if err != nil {
				functionErr = err
				return
			}
			log.Println("compute OK")
			if err = GrB.MatrixEWiseMultBinaryOp(OK, nil, nil, GrB.Eq[T], G.A, G.AT, nil); err != nil {
				functionErr = err
				return
			}
			try(G.A.Wait(GrB.Materialize))
			nvals, err := G.A.NVals()
			if err != nil {
				functionErr = err
				return
			}
			try(OK.Wait(GrB.Materialize))
			oknvals, err := OK.NVals()
			if err != nil {
				functionErr = err
				return
			}
			sym = oknvals == nvals
			if sym {
				log.Println("reduce sym")
				if err = GrB.MatrixReduce(&sym, nil, GrB.LAndMonoid, OK, nil); err != nil {
					functionErr = err
					return
				}
			}
			if !sym {
				log.Println("force symmetric")
				if err = GrB.MatrixEWiseAddBinaryOp(G.A, nil, nil, GrB.Plus[T], G.A, G.AT, nil); err != nil {
					functionErr = err
					return
				}
			}
			G.Kind = AdjacencyUndirected
			G.AStructureIsSymmetric = True
		}
	}

	try(G.A.Wait(GrB.Materialize))

	const nSources = 64

	if computeSourceNodes && !srcNodesDone {
		log.Println("compute sources")
		srcNodes, err = GrB.MatrixNew[int](nSources, 1)
		if err != nil {
			functionErr = err
			return
		}
		for k := 0; k < nSources; k++ {
			i := 1 + rand.Intn(n)
			if err = srcNodes.SetElement(i, k, 0); err != nil {
				functionErr = err
				return
			}
		}
		try(srcNodes.Wait(GrB.Materialize))
	}

	log.Println("ReadProblem done")

	return G, srcNodes, nil
}
