package LAGraph

import (
	"errors"
	"github.com/intel/forGoParallel/parallel"
	"github.com/intel/forGraphBLASGo/GrB"
	"sort"
)

func find(ints []int, element int) (index int, found bool) {
	index = sort.SearchInts(ints, element)
	found = index < len(ints) && ints[index] == element
	return
}

func intersectionSize(x, y []int) (n int) {
	for len(x) > 0 && len(y) > 0 {
		if y[0] > x[0] {
			x, y = y, x
		}
		if index, ok := find(y, x[0]); ok {
			n++
			y = y[index+1:]
		} else {
			y = y[index:]
		}
		x = x[1:]
	}
	return
}

// LCCCheck is a slow version. It's purpose is to have a simple algorithm
// whose result can be used to verify the results of more optimized versions.
func (G *Graph[D]) LCCCheck() (coefficients GrB.Vector[float64], err error) {
	defer GrB.CheckErrors(&err)
	try := func(f func() error) {
		GrB.OK(f())
	}
	A := G.A
	n, ncols, err := A.Size()
	GrB.OK(err)
	if n != ncols {
		err = errors.New("matrix must be square")
		return
	}

	undirected := G.Kind == AdjacencyUndirected ||
		(G.Kind == AdjacencyDirected &&
			G.IsSymmetricStructure == True)
	directed := !undirected

	S, err := GrB.MatrixNew[bool](n, n)
	GrB.OK(err)
	defer try(S.Free)
	GrB.OK(S.AssignConstant(A.AsMask(), nil, true, GrB.All(n), GrB.All(n), GrB.DescS))
	if G.NSelfEdges != 0 {
		GrB.OK(GrB.MatrixSelect(S, nil, nil, GrB.Offdiag[bool](), S, 0, nil))
	}

	T := S
	if directed {
		T, err = GrB.MatrixNew[bool](n, n)
		GrB.OK(err)
		defer try(T.Free)
		GrB.OK(T.EWiseAddBinaryOp(nil, nil, GrB.Oneb[bool](), S, S, GrB.DescT1))
	}

	Sp, Si, Sx, _, _, err := S.UnpackCSR(false, nil)
	GrB.OK(err)
	Sx.Free()
	defer func() {
		Sp.Free()
		Si.Free()
	}()
	Tp, Ti, Tx := Sp, Si, Sx
	if directed {
		Tp, Ti, Tx, _, _, err = T.UnpackCSR(false, nil)
		GrB.OK(err)
		Tx.Free()
		defer func() {
			Tp.Free()
			Ti.Free()
		}()
	}
	Sps := Sp.UnsafeSlice()
	Sis := Si.UnsafeSlice()
	Tps := Tp.UnsafeSlice()
	Tis := Ti.UnsafeSlice()

	coefficients, err = GrB.VectorNew[float64](n)
	GrB.OK(err)
	defer func() {
		if err != nil {
			_ = coefficients.Free()
		}
	}()

	vb := GrB.MakeSystemSlice[bool](n)
	vx := GrB.MakeSystemSlice[float64](n)
	defer func() {
		if err != nil {
			vb.Free()
			vx.Free()
		}
	}()
	vbs := vb.UnsafeSlice()
	vxs := vx.UnsafeSlice()

	nvals := parallel.RangeReduceSum(0, n, n, func(low, high int) (nvals int) {
		for i := low; i < high; i++ {
			neighbors := Tis[Tps[i]:Tps[i+1]]
			k := len(neighbors)
			if k < 2 {
				continue
			}
			esum := 0
			for _, e := range neighbors {
				links := Sis[Sps[e]:Sps[e+1]]
				if len(links) == 0 {
					continue
				}
				if undirected {
					links = links[:sort.SearchInts(links, e)]
				}
				esum += intersectionSize(neighbors, links)
			}
			if esum == 0 {
				continue
			}
			if undirected {
				esum *= 2
			}
			vbs[i] = true
			vxs[i] = float64(esum) / float64(k*(k-1))
			nvals++
		}
		return
	})

	GrB.OK(coefficients.PackBitmap(&vb, &vx, false, nvals, nil))
	return
}
