package main

import (
	"flag"
	"github.com/intel/forGraphBLASGo/GrB"
	"github.com/intel/forLAGraphGo/LAGraph"
	"log"
	"time"
)

func try(err error) {
	if err != nil {
		panic(err)
	}
}

var (
	burble       = false
	panicOnError = false
	ntrials      = 16
)

func main() {
	flag.BoolVar(&burble, "burble", false, "enable burble")
	flag.BoolVar(&panicOnError, "panic", false, "panic on error")
	flag.IntVar(&ntrials, "ntrials", 16, "number of trials")
	flag.Parse()
	try(LAGraph.DemoInit())
	defer func() {
		try(LAGraph.Finalize())
	}()
	try(GrB.GlobalSetBurble(burble))
	try(GrB.GlobalSetPanicOnError(panicOnError))

	log.Println("Burble:", burble)
	log.Println("Panic on error:", panicOnError)
	log.Println("Number of trials:", ntrials)

	var matrixName string
	if len(flag.Args()) > 0 {
		matrixName = flag.Args()[0]
	} else {
		matrixName = "stdin"
	}
	G, _, err := LAGraph.ReadProblem[int64](false, false, false, true, false, false, flag.Args())
	try(err)
	defer func() {
		try(G.Delete())
	}()

	n, _, err := G.A.Size()
	try(err)

	try(G.CachedOutDegree())

	nvals, err := G.OutDegree.Nvals()
	try(err)
	log.Printf("nsinks: %v\n", n-nvals)

	const (
		damping = 0.85
		tol     = 1e-4
		itermax = 100
	)

	tt := time.Now()
	c, _, err := G.PageRankGAP(damping, tol, itermax)
	try(err)
	var indices []int
	var values []float32
	try(c.ExtractTuples(&indices, &values))
	var sum float32
	for _, v := range values {
		sum += v
	}
	log.Println("result:", len(indices), sum, sum/float32(len(indices)))
	try(c.Free())
	log.Printf("warmup time %v\n", time.Since(tt))

	var ttot time.Duration
	for trial := 0; trial < ntrials; trial++ {
		tt = time.Now()
		c, _, err = G.PageRankGAP(damping, tol, itermax)
		try(err)
		try(c.Free())
		ttrial := time.Since(tt)
		ttot += ttrial
		log.Printf("trial %v: %v\n", trial, ttrial)
	}
	ttot = ttot / time.Duration(ntrials)
	log.Printf("Avg: PR GAP time: %v matrix: %v\n", ttot, matrixName)
}
