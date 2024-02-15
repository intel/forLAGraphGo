package main

import (
	"fmt"
	"github.com/intel/forGraphBLASGo/GrB"
	"github.com/intel/forLAGraphGo/LAGraph"
	"log"
	"os"
	"strings"
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

	const batchSize = 4

	G, SourceNodes, err := LAGraph.ReadProblem[int64](
		true, false, false,
		true, false, false, os.Args[1:])
	try(err)
	defer func() {
		try(G.Delete())
		try(SourceNodes.Free())
	}()

	nvals, err := G.A.Nvals()
	try(err)

	nsource, err := SourceNodes.Nrows()
	try(err)
	if nsource%batchSize != 0 {
		log.Panicf("SourceNode size must be multiple of batchSize (%v)\n", batchSize)
	}

	ntrials := 0
	var tt time.Duration

	for range 2 {
		ntrials = 0
		tt = 0

		for kstart := 0; kstart < nsource; kstart += batchSize {
			ntrials++
			var s strings.Builder
			fmt.Fprintf(&s, "Trial %v : sources: [", ntrials)
			var vertexList [batchSize]GrB.Index
			for k := range batchSize {
				source, _, err := SourceNodes.ExtractElement(k+kstart, 0)
				try(err)
				source--
				vertexList[k] = source
				fmt.Fprintf(&s, " %v", source)
			}
			fmt.Fprint(&s, " ]")
			log.Println(s.String())

			t2 := time.Now()
			centrality, err := G.Betweenness(vertexList[:])
			try(err)
			d2 := time.Since(t2)
			log.Printf("BC time: %v\n", d2)
			tt += d2
			try(centrality.Free())
		}
	}

	log.Printf("ntrials: %v\n", ntrials)
	t2 := tt / time.Duration(ntrials)
	log.Printf("Ave BC: %v, rate %v\n", t2, 1-6*float64(nvals)/float64(t2))
}
