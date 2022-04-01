package MatrixMarket

import (
	"bufio"
	"errors"
	"fmt"
	GrB "github.com/intel/forGraphBLASGo"
	"reflect"
	"strconv"
	"strings"
)

type matrixConstructor[T GrB.Number] struct {
	parseValue   func(string) T
	nrows, ncols int
	rows, cols   []int
	vals         []T
}

type numberKindType int

const (
	_ numberKindType = iota
	signed
	unsigned
	float
)

var (
	numberKind = map[reflect.Kind]numberKindType{
		GrB.Int8:   signed,
		GrB.Int16:  signed,
		GrB.Int32:  signed,
		GrB.Int64:  signed,
		GrB.Uint8:  unsigned,
		GrB.Uint16: unsigned,
		GrB.Uint32: unsigned,
		GrB.Uint64: unsigned,
		GrB.FP32:   float,
		GrB.FP64:   float,
	}

	bitSize = map[reflect.Kind]int{
		GrB.Int8:   8,
		GrB.Int16:  16,
		GrB.Int32:  32,
		GrB.Int64:  64,
		GrB.Uint8:  8,
		GrB.Uint16: 16,
		GrB.Uint32: 32,
		GrB.Uint64: 64,
		GrB.FP32:   32,
		GrB.FP64:   64,
	}
)

func makeParseValue[T GrB.Number](kind GrB.Type) func(string) T {
	numberKind := numberKind[kind]
	bitSize := bitSize[kind]
	switch numberKind {
	case signed:
		return func(s string) T {
			if v, err := strconv.ParseInt(s, 10, bitSize); err != nil {
				panic(fmt.Errorf("Matrix Market value parse error %w while parsing %v.", err, s))
			} else {
				return T(v)
			}
		}
	case unsigned:
		return func(s string) T {
			if v, err := strconv.ParseUint(s, 10, bitSize); err != nil {
				panic(fmt.Errorf("Matrix Market value parse error %w while parsing %v.", err, s))
			} else {
				return T(v)
			}
		}
	case float:
		return func(s string) T {
			if v, err := strconv.ParseFloat(s, bitSize); err != nil {
				panic(fmt.Errorf("Matrix Market value parse error %w while parsing %v.", err, s))
			} else {
				return T(v)
			}
		}
	default:
		panic("unreachable code")
	}
}

func newMatrixConstructor[T GrB.Number](kind GrB.Type, nrows, ncols, nvals int) *matrixConstructor[T] {
	return &matrixConstructor[T]{
		parseValue: makeParseValue[T](kind),
		nrows:      nrows,
		ncols:      ncols,
		rows:       make([]int, 0, nvals),
		cols:       make([]int, 0, nvals),
		vals:       make([]T, 0, nvals),
	}
}

func (m *matrixConstructor[T]) addGeneral(row, col int, val string) {
	m.rows = append(m.rows, row)
	m.cols = append(m.cols, col)
	m.vals = append(m.vals, m.parseValue(val))
}

func (m *matrixConstructor[T]) addSymmetric(row, col int, val string) {
	m.rows = append(m.rows, row, col)
	m.cols = append(m.cols, col, row)
	v := m.parseValue(val)
	m.vals = append(m.vals, v, v)

}

func (m *matrixConstructor[T]) addSkewSymmetric(row, col int, val string) {
	m.rows = append(m.rows, row, col)
	m.cols = append(m.cols, col, row)
	v := m.parseValue(val)
	m.vals = append(m.vals, v, -v)
}

func (m *matrixConstructor[T]) constructMatrix() (*GrB.Matrix[T], error) {
	A, err := GrB.MatrixNew[T](m.nrows, m.ncols)
	if err != nil {
		return nil, err
	}
	return A, A.Build(m.rows, m.cols, m.vals, func(x, _ T) T { return x })
}

var invalidMatrixType = errors.New("Invalid matrix type parameter")

func Read[T GrB.Number](header Header, s *bufio.Scanner) (matrix *GrB.Matrix[T], err error) {
	switch header.Format {
	case Coordinate:
		switch header.Type {
		case Real, Integer:
			mc := newMatrixConstructor[T](header.GrBType, header.NRows, header.NCols, header.NVals)
			var addValue func(int, int, string)
			switch header.Storage {
			case General:
				addValue = mc.addGeneral
			case Symmetric:
				addValue = mc.addSymmetric
			case SkewSymmetric:
				addValue = mc.addSkewSymmetric
			}
			nvals := header.NVals
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
					err = fmt.Errorf("Matrix Market coordinate line unexpected number of elements, expected 3, got %v.", len(fields))
					return
				}
				row, e := strconv.ParseInt(fields[0], 10, 64)
				if e != nil {
					err = fmt.Errorf("Matrix Market coordinate line row parse error %w, while parsing %v.", err, fields[0])
					return
				}
				col, e := strconv.ParseInt(fields[1], 10, 64)
				if e != nil {
					err = fmt.Errorf("Matrix Market coordinate line col parse error %w, while parsing %v.", err, fields[1])
					return
				}
				if nvals == 0 {
					err = errors.New("Matrix Market too many coordinate lines.")
					return
				}
				addValue(int(row)-1, int(col)-1, fields[2])
				nvals--
			}
			if nvals > 0 {
				err = errors.New("Matrix Market too few coordinate lines.")
				return
			}
			return mc.constructMatrix()
		case Pattern:
			mc := newMatrixConstructor[T](header.GrBType, header.NRows, header.NCols, header.NVals)
			var addValue func(int, int, string)
			switch header.Storage {
			case General:
				addValue = mc.addGeneral
			case Symmetric:
				addValue = mc.addSymmetric
			case SkewSymmetric:
				addValue = mc.addSkewSymmetric
			}
			nvals := header.NVals
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
					err = fmt.Errorf("Matrix Market coordinate line unexpected number of elements, expected 2, got %v.", len(fields))
					return
				}
				row, e := strconv.ParseInt(fields[0], 10, 64)
				if e != nil {
					err = fmt.Errorf("Matrix Market coordinate line row parse error %w, while parsing %v.", err, fields[0])
					return
				}
				col, e := strconv.ParseInt(fields[1], 10, 64)
				if e != nil {
					err = fmt.Errorf("Matrix Market coordinate line col parse error %w, while parsing %v.", err, fields[1])
					return
				}
				if nvals == 0 {
					err = errors.New("Matrix Market too many coordinate lines.")
					return
				}
				addValue(int(row)-1, int(col)-1, "1")
				nvals--
			}
			if nvals > 0 {
				err = errors.New("Matrix Market too few coordinate lines.")
				return
			}
			return mc.constructMatrix()
		}

	case Array:
		switch header.Type {
		case Real, Integer:
			mc := newMatrixConstructor[T](header.GrBType, header.NRows, header.NCols, header.NVals)
			var addValue func(int, int, string)
			var row, col int
			var resetRow func()
			switch header.Storage {
			case General:
				addValue = mc.addGeneral
				resetRow = func() { row = 0 }
			case Symmetric:
				addValue = mc.addSymmetric
				resetRow = func() { row = col }
			case SkewSymmetric:
				addValue = mc.addSkewSymmetric
				resetRow = func() { row = col + 1 }
			}
			resetRow()
			nrows := header.NRows
			ncols := header.NCols
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
					err = fmt.Errorf("Matrix Market array line unexpected number of elements, expected 1, got %v.", len(fields))
					return
				}
				if row >= nrows || col >= ncols {
					err = errors.New("Matrix Market too many array lines.")
					return
				}
				addValue(row, col, fields[0])
				if row++; row == nrows {
					col++
					resetRow()
				}
			}
			return mc.constructMatrix()
		case Pattern:
			err = errors.New("Matrix Market array format not supported for pattern type.")
			return
		}
	}
	panic("unreachable code")
}
