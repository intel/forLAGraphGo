package main

import (
	"flag"
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
	var cpuprofile, memprofile string
	flag.StringVar(&cpuprofile, "cpuprofile", "", "optional output file for a cpu profile")
	flag.StringVar(&memprofile, "memprofile", "", "optional output file for a mem profile")
	flag.Parse()

	ntrials := 3
	G, _, err := LAG.ReadProblem[int64](false, true, true, true, false, flag.Args())
	try(err)
	log.Println("G.PropertyRowDegree()")
	G.PropertyRowDegree()
	n, err := G.A.NRows()
	try(err)

	log.Println("Warmup.")
	presort := LAG.AutoSelectSort
	method := LAG.SandiaDot2
	tic := time.Now()
	ntriangles := LAG.TriangleCountMethods(G, method, &presort)
	toc := time.Now()
	log.Printf("Warmup: Triangles %v time %v\n", ntriangles, toc.Sub(tic))

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

	for _, method = range []LAG.TriangleCountMethod{LAG.Sandia, LAG.Sandia2, LAG.SandiaDot} {
		sorting := LAG.AutoSelectSort

		if n == 134217726 && method != LAG.SandiaDot {
			log.Printf("kron fails on method %v", method)
			continue
		}
		if n != 134217728 && method != LAG.SandiaDot {
			log.Printf("all but urand is slow with method %v: skipped", method)
			continue
		}

		for trial := 0; trial < ntrials; trial++ {
			log.Printf("starting method %v trial %v\n", method, trial)
			tic = time.Now()
			presort = sorting
			nt2 := LAG.TriangleCountMethods(G, method, &presort)
			toc = time.Now()
			log.Printf("method %v trial %v: duration: %v triangles: %v\n", method, trial, toc.Sub(tic), nt2)
		}
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
