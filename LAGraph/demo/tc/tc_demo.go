package main

import (
	"github.com/intel/forLAGraphGo/LAGraph"
	"log"
	"os"
	"time"
)

func try(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	try(LAGraph.DemoInit())
	defer func() {
		try(LAGraph.Finalize())
	}()
	ntrials := 3
	G, _, err := LAGraph.ReadProblem[int64](false, true, true, true, false, false, os.Args[1:])
	try(err)
	defer func() {
		try(G.Delete())
	}()
	try(G.CachedOutDegree())
	n, err := G.A.Nrows()
	try(err)

	ttot := time.Now()
	ntriangles, _, _, err := G.TriangleCountMethods(LAGraph.TriangleCountSandiaULT, LAGraph.TriangleCountAutoSort)
	try(err)
	log.Printf("warmup: # of triangles %v time %v\n", ntriangles, time.Since(ttot))

	for _, method := range []LAGraph.TriangleCountMethod{LAGraph.TriangleCountSandiaLL, LAGraph.TriangleCountSandiaUU, LAGraph.TriangleCountSandiaLUT} {
		if n == 134217726 && method != LAGraph.TriangleCountSandiaLUT {
			log.Printf("kron fails on method %v", method)
			continue
		}
		if n != 134217728 && method != LAGraph.TriangleCountSandiaLUT {
			log.Printf("all but urand is slow with method %v: skipped", method)
			continue
		}

		for trial := 0; trial < ntrials; trial++ {
			tt := time.Now()
			nt2, _, _, err := G.TriangleCountMethods(method, LAGraph.TriangleCountAutoSort)
			try(err)
			log.Printf("method %v trial %v: duration: %v triangles: %v\n", method, trial, time.Since(tt), nt2)
		}
	}
}
