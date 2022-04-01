package forLAGraphGo

func (G *Graph[T]) Check() {
	A := G.A
	kind := G.Kind
	nrows, ncols, err := A.Size()
	try(err)
	if kind == AdjacencyUndirected || kind == AdjacencyDirected {
		if nrows != ncols {
			panic("adjancency matrix must be square")
		}
	}
	if AT := G.AT; AT != nil {
		nrows2, ncols2, err := AT.Size()
		try(err)
		if nrows != ncols2 || ncols != nrows2 {
			panic("G.AT matrix has the wrong dimensions")
		}
	}
	if rowDegree := G.RowDegree; rowDegree != nil {
		m, err := rowDegree.Size()
		try(err)
		if m != nrows {
			panic("rowdegree invalid size")
		}
	}
	if colDegree := G.ColDegree; colDegree != nil {
		m, err := colDegree.Size()
		try(err)
		if m != ncols {
			panic("coldegree invalid size")
		}
	}
}
