package forLAGraphGo

import (
	"github.com/intel/forGoParallel/parallel"
	"github.com/intel/forGoParallel/psort"
	"sort"
)

type twoSlicesSorter struct {
	s1, s2 []int
}

func (s twoSlicesSorter) Assign(source psort.StableSorter) func(i, j, len int) {
	src := source.(twoSlicesSorter)
	return func(i, j, len int) {
		parallel.Do(func() {
			copy(s.s1[i:i+len], src.s1[j:j+len])
		}, func() {
			copy(s.s2[i:i+len], src.s2[j:j+len])
		})
	}
}

func (s twoSlicesSorter) Len() int {
	return len(s.s1)
}

func (s twoSlicesSorter) Less(i, j int) bool {
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

func (s twoSlicesSorter) NewTemp() psort.StableSorter {
	return twoSlicesSorter{
		s1: make([]int, len(s.s1)),
		s2: make([]int, len(s.s2)),
	}
}

func (s twoSlicesSorter) SequentialSort(i, j int) {
	sort.Stable(twoSlicesSorter{
		s1: s.s1[i:j],
		s2: s.s2[i:j],
	})
}

func (s twoSlicesSorter) Swap(i, j int) {
	s.s1[i], s.s1[j] = s.s1[j], s.s1[i]
	s.s2[i], s.s2[j] = s.s2[j], s.s2[i]
}

func twoSliceSort(s1, s2 []int) {
	psort.StableSort(twoSlicesSorter{s1: s1, s2: s2})
}
