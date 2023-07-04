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
	check        = false
	panicOnError = false
	ntrials      = 3
)

func main() {
	flag.BoolVar(&burble, "burble", false, "enable burble")
	flag.BoolVar(&check, "check", false, "check the result once")
	flag.BoolVar(&panicOnError, "panic", false, "panic on error")
	flag.IntVar(&ntrials, "ntrials", 3, "number of trials")
	flag.Parse()
	try(LAGraph.DemoInit())
	defer func() {
		try(LAGraph.Finalize())
	}()
	try(GrB.GlobalSetBurble(burble))
	try(GrB.GlobalSetPanicOnError(panicOnError))

	log.Println("Burble:", burble)
	log.Println("Check:", check)
	log.Println("Panic on error:", panicOnError)
	log.Println("Number of trials:", ntrials)

	var matrixName string
	if len(flag.Args()) > 0 {
		matrixName = flag.Args()[0]
	} else {
		matrixName = "stdin"
	}
	G, _, err := LAGraph.ReadProblem[int64](false, false, true, true, false, false, flag.Args())
	try(err)
	defer func() {
		try(G.Delete())
	}()

	try(G.CachedIsSymmetricStructure())

	var cgood GrB.Vector[int]

	if check {
		tt := time.Now()
		var e error
		cgood, e = G.CDLPCheck(100)
		try(e)
		defer func() {
			try(cgood.Free())
		}()
		try(cgood.Wait(GrB.Materialize))
		log.Printf("compute check time %v\n", time.Since(tt))
	}

	tt := time.Now()
	c, err := G.CDLP(100)
	try(err)
	defer func() {
		try(c.Free())
	}()
	log.Printf("warmup time %v\n", time.Since(tt))

	if check {
		try(c.Wait(GrB.Materialize))
		ok, err := LAGraph.VectorIsEqual(cgood, c)
		try(err)
		if !ok {
			panic("incorrect result")
		}
		log.Println("correct result")
		try(cgood.Free())
	}

	try(c.Free())

	var ttot time.Duration
	for trial := 0; trial < ntrials; trial++ {
		tt = time.Now()
		c, err = G.CDLP(100)
		try(err)
		try(c.Free())
		ttrial := time.Since(tt)
		ttot += ttrial
		log.Printf("trial %v: %v\n", trial, ttrial)
	}
	ttot = ttot / time.Duration(ntrials)
	log.Printf("Avg: CDLP time: %v matrix: %v\n", ttot, matrixName)
}
