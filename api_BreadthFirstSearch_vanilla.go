package forLAGraphGo

import (
	"github.com/intel/forGoParallel/parallel"
	GrB "github.com/intel/forGraphBLASGo"
	"log"
)

func BreadthFirstSearchVanilla[T GrB.Number](G *Graph[T], src int, computeLevel, computeParent bool) (level, parent *GrB.Vector[int]) {
	G.Check()
	if !(computeLevel || computeParent) {
		return
	}
	A := G.A
	n, err := A.NRows()
	try(err)
	if src >= n {
		log.Panicf("invalid source node %v >= %v", src, n)
	}

	if computeParent {
		B, err := GrB.MatrixNew[int](n, n)
		try(err)
		try(GrB.MatrixApply(B, nil, nil, func(i T) int { return int(i) }, A, nil))

		lParent, err := GrB.VectorNew[int](n)
		try(err)
		semiring := GrB.MinFirstSemiring[int, int]
		frontier, err := GrB.VectorNew[int](n)
		try(err)
		try(frontier.SetElement(src, src))
		ramp := GrB.RowIndex[int, int]

		var lLevel *GrB.Vector[int]
		if computeLevel {
			lLevel, err = GrB.VectorNew[int](n)
			try(err)
		}

		currentLevel := 0

		for {
			if computeLevel {
				try(GrB.VectorAssignConstant(lLevel, frontier.AsMask(), nil, currentLevel, GrB.All(n), GrB.DescS))
				currentLevel++
			}

			try(GrB.VectorAssign(lParent, frontier.AsMask(), nil, frontier, GrB.All(n), GrB.DescS))
			try(GrB.VectorApplyIndexOp(frontier, nil, nil, ramp, frontier, 0, nil))
			try(lParent.Wait(GrB.Materialize))
			try(GrB.VxM(frontier, lParent.AsMask(), nil, semiring, frontier, B, GrB.DescRSC))

			try(frontier.Wait(GrB.Materialize))
			nvals, err := frontier.NVals()
			try(err)
			if nvals == 0 {
				break
			}
		}
		return lLevel, lParent
	}

	B, err := GrB.MatrixNew[bool](n, n)
	try(err)
	try(GrB.MatrixApply(B, nil, nil, func(i T) bool { return i != 0 }, A, nil))

	semiring := StructuralBool[bool, bool]
	frontier, err := GrB.VectorNew[bool](n)
	try(err)
	try(frontier.SetElement(true, src))

	lLevel, err := GrB.VectorNew[int](n)
	try(err)

	currentLevel := 0

	for {
		try(GrB.VectorAssignConstant(lLevel, frontier, nil, currentLevel, GrB.All(n), GrB.DescS))
		currentLevel++

		try(lLevel.Wait(GrB.Materialize))
		try(GrB.VxM(frontier, lLevel.AsMask(), nil, semiring, frontier, B, GrB.DescRSC))

		try(frontier.Wait(GrB.Materialize))
		nvals, err := frontier.NVals()
		try(err)
		if nvals == 0 {
			break
		}
	}
	level = lLevel
	return
}

func CheckVector[T GrB.Number](x []int, X GrB.Vector[T], n, missing int) {
	parallel.Range(0, n, func(low, high int) {
		for i := low; i < high; i++ {
			t, err := X.ExtractElement(i)
			if err == nil {
				x[i] = int(t)
			} else if err == GrB.NoValue {
				x[i] = missing
			} else {
				panic(err)
			}
		}
	})
}

func CheckBFS[T GrB.Number](level, parent *GrB.Vector[int], G *Graph[T], src int) {
	G.Check()
	n, ncols, err := G.A.Size()
	if err != nil {
		panic(err)
	}
	if n != ncols {
		panic("G.A must be square")
	}

	queue := make([]int, n)
	levelCheck := make([]int, n)

	var levelIn, parentIn []int

	parallel.Do(func() {
		if level != nil {
			levelIn = make([]int, n)
			CheckVector(levelIn, *level, n, -1)
		}
	}, func() {
		if parent != nil {
			parentIn = make([]int, n)
			CheckVector(parentIn, *parent, n, -1)
		}
	})

	queue[0] = src

	head := 0
	tail := 1

	visited := make([]bool, n)
	visited[src] = true

	for i := 0; i < n; i++ {
		levelCheck[i] = -1
	}
	levelCheck[src] = 0

	Row, err := GrB.VectorNew[bool](n)

	B, err := GrB.MatrixNew[bool](n, n)
	if err != nil {
		panic(err)
	}
	if err = GrB.MatrixApply(B, nil, nil, func(i T) bool { return i != 0 }, G.A, nil); err != nil {
		panic(err)
	}

	for head < tail {
		u := queue[head]
		head++
		if err = GrB.ColExtract(Row, nil, nil, B, GrB.All(n), u, GrB.DescT0); err != nil {
			panic(err)
		}
		nodeUAdjacencyList, _, err := Row.ExtractTuples()
		if err != nil {
			panic(err)
		}
		degree := len(nodeUAdjacencyList)

		for k := 0; k < degree; k++ {
			v := nodeUAdjacencyList[k]
			if !visited[v] {
				visited[v] = true
				levelCheck[v] = levelCheck[u] + 1
				queue[tail] = v
				tail++
			}
		}
	}

	parallel.Do(func() {
		if levelIn != nil {
			parallel.Range(0, n, func(low, high int) {
				for i := low; i < high; i++ {
					ok := levelIn[i] == levelCheck[i]
					if !ok {
						panic("incorrect levels")
					}
				}
			})
		}
	}, func() {
		if parentIn != nil {
			parallel.Range(0, n, func(low, high int) {
				for i := low; i < high; i++ {
					if i == src {
						ok := parentIn[src] == src && visited[src]
						if !ok {
							panic("incorrect parents")
						}
					} else if visited[i] {
						pi := parentIn[i]
						ok := pi >= 0 && pi < n && visited[pi]
						if !ok {
							panic("incorrect parents")
						}
						if _, err := G.A.ExtractElement(pi, i); err != nil {
							panic(err)
						}
						ok = levelCheck[i] == levelCheck[pi]+1
						if !ok {
							panic("invalid parent")
						}
					}
				}
			})
		}
	})
}
