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
	G, SourceNodes, err := LAGraph.ReadProblem[int64](true, false, false, true, false, false, os.Args[1:])
	try(err)
	defer func() {
		try(G.Delete())
		try(SourceNodes.Free())
	}()

	try(G.CachedOutDegree())
	ntrials, err := SourceNodes.Nrows()
	try(err)

	src, _, err := SourceNodes.ExtractElement(0, 0)
	try(err)
	twarmup := time.Now()
	_, parent, err := G.BreadthFirstSearch(src, false, true)
	try(parent.Free())
	try(err)
	log.Println("warmup: parent only, pushpull:", time.Since(twarmup))

	for trial := range ntrials {
		src, _, err := SourceNodes.ExtractElement(trial, 0)
		try(err)
		src--
		ttrial := time.Now()
		_, parent, err := G.BreadthFirstSearch(src, false, true)
		try(err)
		log.Printf("parent only pushpull trial: %v src %v duration: %v\n", trial, src, time.Since(ttrial))

		try(parent.Free())

		ttrial = time.Now()
		level, parent, err := G.BreadthFirstSearch(src, true, true)
		try(err)
		log.Printf("parent+level pushpull trial: %v src %v duration %v\n", trial, src, time.Since(ttrial))

		try(level.Free())
		try(parent.Free())
	}
}
