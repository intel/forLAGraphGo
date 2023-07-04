package LAGraph

import (
	"github.com/intel/forGoParallel/parallel"
	"github.com/intel/forGoParallel/psort"
	"github.com/intel/forGraphBLASGo/GrB"
	"sort"
)

type twoSlicesSorter[D GrB.Number] struct {
	s1, s2 []D
}

func (s twoSlicesSorter[D]) Assign(source psort.StableSorter) func(i, j, len int) {
	src := source.(twoSlicesSorter[D])
	return func(i, j, len int) {
		parallel.Do(func() {
			copy(s.s1[i:i+len], src.s1[j:j+len])
		}, func() {
			copy(s.s2[i:i+len], src.s2[j:j+len])
		})
	}
}

func (s twoSlicesSorter[D]) Len() int {
	return len(s.s1)
}

func (s twoSlicesSorter[D]) Less(i, j int) bool {
	si := s.s1[i]
	sj := s.s1[j]
	if si < sj {
		return true
	}
	if si > sj {
		return false
	}
	return s.s2[i] < s.s2[j]
}

func (s twoSlicesSorter[D]) NewTemp() psort.StableSorter {
	return twoSlicesSorter[D]{
		s1: make([]D, len(s.s1)),
		s2: make([]D, len(s.s2)),
	}
}

func (s twoSlicesSorter[D]) SequentialSort(i, j int) {
	sort.Stable(twoSlicesSorter[D]{
		s1: s.s1[i:j],
		s2: s.s2[i:j],
	})
}

func (s twoSlicesSorter[D]) Swap(i, j int) {
	s.s1[i], s.s1[j] = s.s1[j], s.s1[i]
	s.s2[i], s.s2[j] = s.s2[j], s.s2[i]
}

func twoSliceSort[D GrB.Number](s1, s2 []D) {
	psort.StableSort(twoSlicesSorter[D]{s1: s1, s2: s2})
}
