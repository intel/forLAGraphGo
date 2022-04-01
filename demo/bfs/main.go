package main

import (
	"flag"
	"github.com/intel/forGoParallel/parallel"
	GrB "github.com/intel/forGraphBLASGo"
	LAG "github.com/intel/forLAGraphGo"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

func try(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	var checkResult bool
	flag.BoolVar(&checkResult, "check", false, "check result of breadth-first search algorithm")
	var cpuprofile, memprofile string
	flag.StringVar(&cpuprofile, "cpuprofile", "", "optional output file for a cpu profile")
	flag.StringVar(&memprofile, "memprofile", "", "optional output file for a mem profile")
	flag.Parse()
	G, SourceNodes, err := LAG.ReadProblem[int64](true, false, false, true, false, flag.Args())
	try(err)

	G.PropertyRowDegree()
	ntrials, err := SourceNodes.NRows()
	try(err)

	log.Println("Warmup.")
	src, err := SourceNodes.ExtractElement(0, 0)
	try(err)
	level, parent := LAG.BreadthFirstSearchVanilla(G, src, false, true)
	try(parent.Wait(GrB.Materialize))

	if cpuprofile != "" {
		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err = pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	for trial := 0; trial < ntrials; trial++ {
		src, err := SourceNodes.ExtractElement(trial, 0)
		try(err)
		src--
		log.Printf("starting parent only trial: %v src %v\n", trial, src)
		tic := time.Now()
		_, parent = LAG.BreadthFirstSearchVanilla(G, src, false, true)
		try(parent.Wait(GrB.Materialize))
		toc := time.Now()
		log.Printf("parent only trial: %v src %v duration: %v\n", trial, src, toc.Sub(tic))

		if checkResult && trial == 0 {
			log.Println("checking...")
			tic = time.Now()
			LAG.CheckBFS(nil, parent, G, src)
			toc = time.Now()
			log.Printf("check: %v\n", toc.Sub(tic))
		}

		log.Printf("starting parent+level tial: %v src %v\n", trial, src)
		tic = time.Now()
		level, parent = LAG.BreadthFirstSearchVanilla(G, src, true, true)
		parallel.Do(func() {
			level.Wait(GrB.Materialize)
		}, func() {
			parent.Wait(GrB.Materialize)
		})
		toc = time.Now()
		log.Printf("parent+level trial: %v src %v duration %v\n", trial, src, toc.Sub(tic))
	}

	if memprofile != "" {
		f, err := os.Create(memprofile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		defer f.Close()
		runtime.GC()
		if err = pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
	}
}
