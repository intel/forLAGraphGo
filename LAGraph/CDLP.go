package LAGraph

import (
	"errors"
	"github.com/intel/forGoParallel/parallel"
	"github.com/intel/forGraphBLASGo/GrB"
)

func countsReduce(counts map[int]int) (entry, count int) {
	entry = GrB.IndexMax + 1
	count = 0
	for e2, c2 := range counts {
		if count > c2 {
			continue
		}
		if c2 > count {
			entry = e2
			count = c2
			continue
		}
		if entry < e2 {
			continue
		}
		entry = e2
		count = c2
	}
	return
}

func (G *Graph[D]) CDLP(itermax int) (result GrB.Vector[int], err error) {
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

	S, err := GrB.MatrixNew[int](n, n)
	GrB.OK(err)
	defer try(S.Free)
	GrB.OK(GrB.MatrixApplyBinaryOp2nd(S, nil, nil, GrB.Oneb[int](), GrB.MatrixView[int, D](A), 0, nil))

	var Tps, Tis []int
	if G.Kind == AdjacencyDirected {
		AT, e := GrB.MatrixNew[int](n, n)
		GrB.OK(e)
		defer try(AT.Free)
		GrB.OK(GrB.Transpose(AT, nil, nil, S, nil))
		Tp, Ti, Tx, _, _, e := AT.UnpackCSR(true, nil)
		GrB.OK(e)
		Tx.Free()
		defer func() {
			Tp.Free()
			Ti.Free()
		}()
		Tps = Tp.UnsafeSlice()
		Tis = Ti.UnsafeSlice()
		GrB.OK(AT.Free())
	}

	var Sps, Sis []int
	{
		Sp, Si, Sx, _, _, e := S.UnpackCSR(true, nil)
		GrB.OK(e)
		Sx.Free()
		defer func() {
			Sp.Free()
			Si.Free()
		}()
		Sps = Sp.UnsafeSlice()
		Sis = Si.UnsafeSlice()
		GrB.OK(S.Free())
	}

	L := make([]int, n)
	for i := range L {
		L[i] = i
	}
	Lnext := make([]int, n)

	for iteration := 0; iteration < itermax; iteration++ {
		parallel.Range(0, n, n, func(low, high int) {
			counts := make(map[int]int)
			for i := low; i < high; i++ {
				clear(counts)
				neighbors := Sis[Sps[i]:Sps[i+1]]
				for _, neighbor := range neighbors {
					counts[L[neighbor]]++
				}
				if G.Kind == AdjacencyDirected {
					neighbors = Tis[Tps[i]:Tps[i+1]]
					for _, neighbor := range neighbors {
						counts[L[neighbor]]++
					}
				}
				bestLabel, _ := countsReduce(counts)
				Lnext[i] = bestLabel
			}
		})
		L, Lnext = Lnext, L
		changed := false
		for i := 0; i < n; i++ {
			if L[i] != Lnext[i] {
				changed = true
				break
			}
		}
		if !changed {
			break
		}
	}

	result, err = GrB.VectorNew[int](n)
	GrB.OK(err)
	defer func() {
		if err != nil {
			_ = result.Free()
		}
	}()
	for i, label := range L {
		GrB.OK(result.SetElement(label, i))
	}

	return
}
