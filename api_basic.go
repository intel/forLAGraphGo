package forLAGraphGo

type Kind int

const (
	AdjacencyUndirected Kind = iota
	AdjacencyDirected
)

type Boolean int

const (
	False Boolean = iota
	True
	Unknown
)

func try(err error) {
	if err != nil {
		panic(err)
	}
}
