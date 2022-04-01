package forLAGraphGo

const (
	Random15Max = 32767
	Random60Max = (1 << 60) - 1
)

func Random15(seed *uint64) uint64 {
	*seed = *seed*1103515245 + 12345
	return (*seed / 65536) % (Random15Max + 1)
}

func Random60(seed *uint64) uint64 {
	i := Random15(seed)
	i = Random15(seed) + Random15Max*i
	i = Random15(seed) + Random15Max*i
	i = Random15(seed) + Random15Max*i
	i = i % (Random60Max + 1)
	return i
}
