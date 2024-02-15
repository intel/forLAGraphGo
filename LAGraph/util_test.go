package LAGraph_test

import "github.com/intel/forGraphBLASGo/GrB"

func checkVector(X GrB.Vector[int], n, missing int) ([]int, error) {
	x := make([]int, n)
	for i := range n {
		if t, ok, err := X.ExtractElement(i); err != nil {
			return nil, err
		} else if ok {
			x[i] = t
		} else {
			x[i] = missing
		}
	}
	return x, nil
}
