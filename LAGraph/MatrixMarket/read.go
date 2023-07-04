package MatrixMarket

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/intel/forGraphBLASGo/GrB"
	"io"
	"strconv"
	"strings"
	"unsafe"
)

type matrixConstructor[D GrB.Number] struct {
	parseValue   func(string) D
	nrows, ncols int
	rows, cols   []int
	vals         []D
}

type numberKindType int

const (
	_ numberKindType = iota
	signed
	unsigned
	float
)

var (
	numberKind = map[GrB.Type]numberKindType{
		GrB.Int:     signed,
		GrB.Int8:    signed,
		GrB.Int16:   signed,
		GrB.Int32:   signed,
		GrB.Int64:   signed,
		GrB.Uint:    unsigned,
		GrB.Uint8:   unsigned,
		GrB.Uint16:  unsigned,
		GrB.Uint32:  unsigned,
		GrB.Uint64:  unsigned,
		GrB.Float32: float,
		GrB.Float64: float,
	}

	bitSize = map[GrB.Type]int{
		GrB.Int:     64,
		GrB.Int8:    8,
		GrB.Int16:   16,
		GrB.Int32:   32,
		GrB.Int64:   64,
		GrB.Uint:    64,
		GrB.Uint8:   8,
		GrB.Uint16:  16,
		GrB.Uint32:  32,
		GrB.Uint64:  64,
		GrB.Float32: 32,
		GrB.Float64: 64,
	}
)

func init() {
	if unsafe.Sizeof(0) == 4 {
		bitSize[GrB.Int] = 32
		bitSize[GrB.Uint] = 32
	}
}

func makeParseValue[D GrB.Number](kind GrB.Type) func(string) D {
	numKind := numberKind[kind]
	bits := bitSize[kind]
	switch numKind {
	case signed:
		return func(s string) D {
			if v, err := strconv.ParseInt(s, 10, bits); err != nil {
				panic(fmt.Errorf("MatrixMarket value parse error %w while parsing %v", err, s))
			} else {
				return D(v)
			}
		}
	case unsigned:
		return func(s string) D {
			if v, err := strconv.ParseUint(s, 10, bits); err != nil {
				panic(fmt.Errorf("MatrixMarket value parse error %w while parsing %v", err, s))
			} else {
				return D(v)
			}
		}
	case float:
		return func(s string) D {
			if v, err := strconv.ParseFloat(s, bits); err != nil {
				panic(fmt.Errorf("MatrixMarket value parse error %w while parsing %v", err, s))
			} else {
				return D(v)
			}
		}
	default:
		panic("unreachable code")
	}
}

func makeMatrixConstructor[D GrB.Number](kind GrB.Type, nrows, ncols, nvals int, storage storage) *matrixConstructor[D] {
	switch storage {
	case symmetric, skewSymmetric:
		nvals *= 2
	}
	return &matrixConstructor[D]{
		parseValue: makeParseValue[D](kind),
		nrows:      nrows,
		ncols:      ncols,
		rows:       make([]int, 0, nvals),
		cols:       make([]int, 0, nvals),
		vals:       make([]D, 0, nvals),
	}
}

func (m *matrixConstructor[D]) addGeneral(row, col int, val string) {
	m.rows = append(m.rows, row)
	m.cols = append(m.cols, col)
	m.vals = append(m.vals, m.parseValue(val))
}

func (m *matrixConstructor[D]) addSymmetric(row, col int, val string) {
	m.rows = append(m.rows, row, col)
	m.cols = append(m.cols, col, row)
	v := m.parseValue(val)
	m.vals = append(m.vals, v, v)
}

func (m *matrixConstructor[D]) addSkewSymmetric(row, col int, val string) {
	m.rows = append(m.rows, row, col)
	m.cols = append(m.cols, col, row)
	v := m.parseValue(val)
	m.vals = append(m.vals, v, -v)
}

func (m *matrixConstructor[D]) constructMatrix() (A GrB.Matrix[D], err error) {
	A, err = GrB.MatrixNew[D](m.nrows, m.ncols)
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			_ = A.Free()
		}
	}()
	dup := GrB.First[D, D]()
	err = A.Build(m.rows, m.cols, m.vals, &dup)
	return
}

func Read[D GrB.Number](r io.Reader) (matrix GrB.Matrix[D], err error) {
	hdr, s, err := readHeader(r)
	if err != nil {
		return
	}
	switch hdr.grbType {
	case GrB.Int8:
		return readDispatch[D, int8](hdr, s)
	case GrB.Int16:
		return readDispatch[D, int16](hdr, s)
	case GrB.Int32:
		return readDispatch[D, int32](hdr, s)
	case GrB.Int64:
		return readDispatch[D, int64](hdr, s)
	case GrB.Uint8:
		return readDispatch[D, uint8](hdr, s)
	case GrB.Uint16:
		return readDispatch[D, uint16](hdr, s)
	case GrB.Uint32:
		return readDispatch[D, uint32](hdr, s)
	case GrB.Uint64:
		return readDispatch[D, uint64](hdr, s)
	case GrB.Float32:
		return readDispatch[D, float32](hdr, s)
	case GrB.Float64:
		return readDispatch[D, float64](hdr, s)
	default:
		panic("unreachable code")
	}
}

func readDispatch[To, From GrB.Number](hdr header, s *bufio.Scanner) (matrix GrB.Matrix[To], err error) {
	m, err := read[From](hdr, s)
	if err != nil {
		return
	}
	matrix = GrB.MatrixView[To, From](m)
	return
}

func read[D GrB.Number](hdr header, s *bufio.Scanner) (matrix GrB.Matrix[D], err error) {
	switch hdr.format {
	case coordinate:
		switch hdr.typ {
		case treal, tinteger:
			mc := makeMatrixConstructor[D](hdr.grbType, hdr.nrows, hdr.ncols, hdr.nvals, hdr.storage)
			var addValue func(int, int, string)
			switch hdr.storage {
			case general:
				addValue = mc.addGeneral
			case symmetric:
				addValue = mc.addSymmetric
			case skewSymmetric:
				addValue = mc.addSkewSymmetric
			}
			nvals := hdr.nvals
			for s.Scan() {
				sText := s.Text()
				if strings.HasPrefix(sText, "%") {
					continue
				}
				sText = strings.TrimSpace(sText)
				if sText == "" {
					continue
				}
				fields := strings.Fields(sText)
				if len(fields) != 3 {
					err = fmt.Errorf("MatrixMarket coordinate line unexpected number of elements, expected 3, got %v", len(fields))
					return
				}
				row, e := strconv.ParseInt(fields[0], 10, 64)
				if e != nil {
					err = fmt.Errorf("MatrixMarket coordinate line row parse error %w, while parsing %v", err, fields[0])
					return
				}
				col, e := strconv.ParseInt(fields[1], 10, 64)
				if e != nil {
					err = fmt.Errorf("MatrixMarket coordinate line col parse error %w, while parsing %v", err, fields[1])
					return
				}
				if nvals == 0 {
					err = errors.New("MatrixMarket too many coordinate lines")
					return
				}
				addValue(int(row-1), int(col-1), fields[2])
				nvals--
			}
			if nvals > 0 {
				err = errors.New("MatrixMarket too few coordinate lines")
				return
			}
			return mc.constructMatrix()
		case tpattern:
			mc := makeMatrixConstructor[D](hdr.grbType, hdr.nrows, hdr.ncols, hdr.nvals, hdr.storage)
			var addValue func(int, int, string)
			switch hdr.storage {
			case general:
				addValue = mc.addGeneral
			case symmetric:
				addValue = mc.addSymmetric
			case skewSymmetric:
				addValue = mc.addSkewSymmetric
			}
			nvals := hdr.nvals
			for s.Scan() {
				sText := s.Text()
				if strings.HasPrefix(sText, "%") {
					continue
				}
				sText = strings.TrimSpace(sText)
				if sText == "" {
					continue
				}
				fields := strings.Fields(sText)
				if len(fields) != 2 {
					err = fmt.Errorf("MatrixMarket coordinate line unexpected number of elements, expected 2, got %v", len(fields))
					return
				}
				row, e := strconv.ParseInt(fields[0], 10, 64)
				if e != nil {
					err = fmt.Errorf("MatrixMarket coordinate line row parse error %w, while parsing %v", err, fields[0])
					return
				}
				col, e := strconv.ParseInt(fields[1], 10, 64)
				if e != nil {
					err = fmt.Errorf("MatrixMarket coordinate line col parse error %w, while parsing %v", err, fields[1])
					return
				}
				if nvals == 0 {
					err = errors.New("MatrixMarket too many coordinate lines")
					return
				}
				addValue(int(row-1), int(col-1), "1")
				nvals--
			}
			if nvals > 0 {
				err = errors.New("MatrixMarket too few coordinate lines")
				return
			}
			return mc.constructMatrix()
		}

	case array:
		switch hdr.typ {
		case treal, tinteger:
			mc := makeMatrixConstructor[D](hdr.grbType, hdr.nrows, hdr.ncols, hdr.nvals, hdr.storage)
			var addValue func(int, int, string)
			var row, col int
			var resetRow func()
			switch hdr.storage {
			case general:
				addValue = mc.addGeneral
				resetRow = func() { row = 0 }
			case symmetric:
				addValue = mc.addSymmetric
				resetRow = func() { row = col }
			case skewSymmetric:
				addValue = mc.addSkewSymmetric
				resetRow = func() { row = col + 1 }
			}
			resetRow()
			nrows := hdr.nrows
			ncols := hdr.ncols
			for s.Scan() {
				sText := s.Text()
				if strings.HasPrefix(sText, "%") {
					continue
				}
				sText = strings.TrimSpace(sText)
				if sText == "" {
					continue
				}
				fields := strings.Fields(sText)
				if len(fields) != 1 {
					err = fmt.Errorf("MatrixMarket array line unexpected number of elements, expected 1, got %v", len(fields))
					return
				}
				if row >= nrows || col >= ncols {
					err = errors.New("MatrixMarket too many array lines")
					return
				}
				addValue(row, col, fields[0])
				if row++; row == nrows {
					col++
					resetRow()
				}
			}
			return mc.constructMatrix()
		case tpattern:
			err = errors.New("MatrixMarket array format not supported for pattern type")
			return
		}
	}
	panic("unreachable code")
}
