package LAGraph

import (
	"errors"
	"github.com/intel/forGraphBLASGo/GrB"
	"math"
	"math/rand"
	"runtime"
	"sync"
)

func (G *Graph[D]) ConnectedComponents() (component GrB.Vector[int], err error) {
	defer GrB.CheckErrors(&err)

	GrB.OK(G.Check())
	if !(G.Kind == AdjacencyUndirected || (G.Kind == AdjacencyDirected && G.IsSymmetricStructure == True)) {
		err = errors.New("G.A must be known to be symmetric")
		return
	}

	A := G.A
	n, err := A.Nrows()
	GrB.OK(err)
	nvals, err := A.Nvals()
	GrB.OK(err)
	if n > math.MaxInt32 {
		component = connectedComponents[D, int64, uint64](A, n, nvals)
	} else {
		component = connectedComponents[D, int32, uint32](A, n, nvals)
	}
	return
}

func fastsv[Uint uint32 | uint64](
	A GrB.Matrix[bool],
	parent, mngp GrB.Vector[Uint],
	gp, gpNew *GrB.Vector[Uint],
	t GrB.Vector[bool],
	eq GrB.BinaryOp[bool, Uint, Uint],
	min GrB.BinaryOp[Uint, Uint, Uint],
	min2nd GrB.Semiring[Uint, Uint, Uint],
	C GrB.Matrix[bool],
	cp, px *GrB.SystemSlice[int],
) {
	cx := GrB.MakeSystemSlice[bool](1)
	defer cx.Free()
	iso := true
	jumbled := false
	for {
		GrB.OK(GrB.MxV(mngp, nil, &min, min2nd, GrB.MatrixView[Uint, bool](A), *gp, nil))
		GrB.OK(C.PackCSC(cp, px, &cx, iso, jumbled, nil))
		GrB.OK(GrB.MxV(parent, nil, &min, min2nd, GrB.MatrixView[Uint, bool](C), mngp, nil))
		var err error
		*cp, *px, cx, iso, jumbled, err = C.UnpackCSC(true, nil)
		GrB.OK(err)

		GrB.OK(GrB.VectorEWiseAddBinaryOp(parent, nil, &min, min, mngp, *gp, nil))
		pxs := (*px).UnsafeSlice()[:0]
		GrB.OK(GrB.VectorView[int, Uint](parent).ExtractTuples(nil, &pxs))
		GrB.OK(GrB.VectorExtract(*gpNew, nil, nil, parent, pxs, nil))
		GrB.OK(GrB.VectorEWiseMultBinaryOp(t, nil, nil, eq, *gpNew, *gp, nil))
		done, err := GrB.VectorReduce(GrB.LandMonoidBool, t, nil)
		GrB.OK(err)
		if done {
			break
		}
		*gp, *gpNew = *gpNew, *gp
	}
}

func connectedComponents[D GrB.Predefined, Int int32 | int64, Uint uint32 | uint64](A GrB.Matrix[D], n, nvals int) GrB.Vector[int] {
	try := func(f func() error) {
		GrB.OK(f())
	}
	ramp := GrB.RowIndex[Int, Int]()
	umin := GrB.Min[Uint]()
	imin := GrB.Min[Int]()
	eq := GrB.Eq[Uint]()
	min2nd := GrB.MinSecondSemiring[Uint]()
	min2ndi := GrB.MinSecondi[Int]()

	const fastsvSamples = 4
	sampling := nvals > n*fastsvSamples*2 && n > 1024

	nthreads := runtime.GOMAXPROCS(0)
	nthreads = min(nthreads, n/16)
	nthreads = max(nthreads, 1)

	c, err := GrB.MatrixNew[bool](n, n)
	GrB.OK(err)
	defer try(c.Free)
	var cp GrB.SystemSlice[int]
	defer cp.Free()
	{
		t, e := GrB.VectorNew[int](n + 1)
		GrB.OK(e)
		defer try(t.Free)

		GrB.OK(GrB.VectorAssignConstant(t, nil, nil, 0, GrB.All(n+1), nil))
		GrB.OK(GrB.VectorApplyIndexOp(t, nil, nil, GrB.RowIndex[int, int](), t, 0, nil))
		cp, _, err = t.UnpackFull(nil)
		GrB.OK(err)
		GrB.OK(t.Free())
	}

	var y GrB.Vector[Int]
	{
		t, e := GrB.VectorNew[Int](n)
		GrB.OK(e)
		defer try(t.Free)

		GrB.OK(GrB.VectorAssignConstant(t, nil, nil, 0, GrB.All(n), nil))

		y, err = GrB.VectorNew[Int](n)
		GrB.OK(err)
		defer try(y.Free)

		GrB.OK(GrB.VectorAssignConstant(y, nil, nil, 0, GrB.All(n), nil))
		GrB.OK(GrB.VectorApplyIndexOp(y, nil, nil, ramp, y, 0, nil))
		GrB.OK(GrB.MxV(y, nil, &imin, min2ndi, GrB.MatrixView[Int, D](A), t, nil))

		GrB.OK(t.Free())
	}

	parent, err := GrB.VectorNew[Uint](n)
	GrB.OK(err)
	GrB.OK(GrB.VectorAssign(parent, nil, nil, GrB.VectorView[Uint, Int](y), GrB.All(n), nil))
	GrB.OK(y.Free())

	px := GrB.MakeSystemSlice[int](n)
	pxs := px.UnsafeSlice()[:0]
	GrB.OK(GrB.VectorView[int, Uint](parent).ExtractTuples(nil, &pxs))

	gp, err := parent.Dup()
	GrB.OK(err)
	defer try(gp.Free)

	mngp, err := parent.Dup()
	GrB.OK(err)
	defer try(mngp.Free)

	gpNew, err := GrB.VectorNew[Uint](n)
	GrB.OK(err)
	defer try(gpNew.Free)

	t, err := GrB.VectorNew[bool](n)
	GrB.OK(err)
	defer try(t.Free)

	if sampling {
		ap, aj, ax, aiso, ajumbled, e := A.UnpackCSR(true, nil)
		GrB.OK(e)

		tp := GrB.MakeSystemSlice[int](n + 1)
		tj := GrB.MakeSystemSlice[int](nvals)
		tx := GrB.MakeSystemSlice[bool](1)
		ranges := make([]int, nthreads+1)
		counts := make([]int, nthreads+1)

		for tid := range nthreads + 1 {
			ranges[tid] = (n*tid + nthreads - 1) / nthreads
		}

		aps := ap.UnsafeSlice()

		var wg sync.WaitGroup
		wg.Add(nthreads)
		for tid := range nthreads {
			go func(tid int) {
				defer wg.Done()
				for i := ranges[tid]; i < ranges[tid+1]; i++ {
					deg := aps[i+1] - aps[i]
					counts[tid+1] += min(fastsvSamples, deg)
				}
			}(tid)
		}
		wg.Wait()

		for tid := range nthreads {
			counts[tid+1] += counts[tid]
		}

		ajs := aj.UnsafeSlice()
		tps := tp.UnsafeSlice()
		tjs := tj.UnsafeSlice()

		wg.Add(nthreads)
		for tid := range nthreads {
			go func(tid int) {
				defer wg.Done()
				p := counts[tid]
				tps[ranges[tid]] = p
				for i := ranges[tid]; i < ranges[tid+1]; i++ {
					for j := range fastsvSamples {
						if aps[i]+j >= aps[i+1] {
							break
						}
						tjs[p] = ajs[aps[i]+j]
						p++
					}
					tps[i+1] = p
				}
			}(tid)
		}
		wg.Wait()

		T, e := GrB.MatrixNew[bool](n, n)
		GrB.OK(e)
		defer try(T.Free)

		GrB.OK(T.PackCSR(&tp, &tj, &tx, true, ajumbled, nil))

		fastsv[Uint](T, parent, mngp, &gp, &gpNew, t, eq, umin, min2nd, c, &cp, &px)

		const hashSamples = 864
		htCount := make(map[int]int32)
		rnd := rand.New(rand.NewSource(rand.Int63()))
		key := -1
		maxCount := int32(0)
		for range hashSamples {
			x := pxs[rnd.Intn(n)]
			htCount[x]++
			if htCount[x] > maxCount {
				key = x
				maxCount = htCount[x]
			}
		}

		var tiso bool
		tp, tj, tx, tiso, _, err = T.UnpackCSR(true, nil)
		GrB.OK(err)

		wg.Add(nthreads)
		for tid := range nthreads {
			go func(tid int) {
				defer wg.Done()
				p := aps[ranges[tid]]
				for i := ranges[tid]; i < ranges[tid+1]; i++ {
					pi := pxs[i]
					tps[i] = p
					if pi != key {
						for ps := aps[i]; ps < aps[i+1]; ps++ {
							j := ajs[ps]
							if pxs[j] != key {
								tjs[p] = j
								p++
							}
						}
						if p-tps[i] < aps[i+1]-aps[i] {
							tjs[p] = key
							p++
						}
					}
				}
				counts[tid] = p - tps[ranges[tid]]
			}(tid)
		}
		wg.Wait()

		nvals = 0
		for tid := range nthreads {
			copy(tjs[nvals:nvals+counts[tid]], tjs[tps[ranges[tid]]:])
			nvals += counts[tid]
			counts[tid] = nvals - counts[tid]
		}

		wg.Add(nthreads)
		for tid := range nthreads {
			go func(tid int) {
				defer wg.Done()
				p := tps[ranges[tid]]
				for i := ranges[tid]; i < ranges[tid+1]; i++ {
					tps[i] -= p - counts[tid]
				}
			}(tid)
		}
		wg.Wait()

		tps[n] = nvals

		GrB.OK(T.PackCSR(&tp, &tj, &tx, tiso, true, nil))
		GrB.OK(A.PackCSR(&ap, &aj, &ax, aiso, ajumbled, nil))

		A = GrB.MatrixView[D, bool](T)
	}

	if nvals == 0 {
		return GrB.VectorView[int, Uint](parent)
	}

	fastsv[Uint](GrB.MatrixView[bool, D](A), parent, mngp, &gp, &gpNew, t, eq, umin, min2nd, c, &cp, &px)

	return GrB.VectorView[int, Uint](parent)
}
