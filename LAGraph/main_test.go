package LAGraph_test

import (
	"github.com/intel/forGraphBLASGo/GrB"
	"github.com/intel/forLAGraphGo/LAGraph"
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if err := LAGraph.Init(GrB.NonBlocking); err != nil {
		panic(err)
	}
	defer func() {
		if err := LAGraph.Finalize(); err != nil {
			panic(err)
		}
	}()
	log.Print(
		GrB.SuiteSparseImplementationName, " ",
		GrB.SuiteSparseImplementationMajor, ".",
		GrB.SuiteSparseImplementationMinor, ".",
		GrB.SuiteSparseImplementationSub,
	)
	if omp, err := GrB.GlobalGetOpenMP(); err != nil {
		panic(err)
	} else {
		log.Println("OpenMP:", omp)
	}
	os.Exit(m.Run())
}
