package MatrixMarket

import (
	"bufio"
	"errors"
	"fmt"
	GrB "github.com/intel/forGraphBLASGo"
	"io"
	"strconv"
	"strings"
)

const (
	coordinateString    = "coordinate"
	arrayString         = "array"
	realString          = "real"
	complexString       = "complex"
	patternString       = "pattern"
	integerString       = "integer"
	generalString       = "general"
	hermitianString     = "hermitian"
	symmetricString     = "symmetric"
	skewSymmetricString = "skew-symmetric"
)

type (
	Format  int
	Type    int
	Storage int
)

const (
	Coordinate Format = iota
	Array
)

const (
	Real Type = iota
	Complex
	Pattern
	Integer
)

const (
	General Storage = iota
	Hermitian
	Symmetric
	SkewSymmetric
)

type Header struct {
	Format              Format
	Type                Type
	GrBType             GrB.Type
	Storage             Storage
	NRows, NCols, NVals int
}

func ReadHeader(r io.Reader) (header Header, scanner *bufio.Scanner, err error) {
	s := bufio.NewScanner(r)
	if !s.Scan() {
		err = errors.New("Matrix Market header line missing.")
		return
	}
	fields := strings.Fields(s.Text())
	if len(fields) != 5 {
		err = fmt.Errorf("Matrix Market incorrect number of header line elements; expected 5, got %v.", len(fields))
		return
	}
	if fields[0] != "%%MatrixMarket" {
		err = fmt.Errorf("Matrix Market header line prefix missing; expected %%MatrixMarket, got %v.", fields[0])
		return
	}
	if fields[1] != "matrix" {
		err = fmt.Errorf("Matrix Market header line second entry incorrect; expected matrix, got %v.", fields[1])
		return
	}
	formatString, typeString, storageString := strings.ToLower(fields[2]), strings.ToLower(fields[3]), strings.ToLower(fields[4])
	var format Format
	switch formatString {
	case coordinateString:
		format = Coordinate
	case arrayString:
		format = Array
	default:
		err = fmt.Errorf("Matrix Market header line format entry incorrect; expected (coordinate | array), got %v.", formatString)
		return
	}
	var typ Type
	switch typeString {
	case realString:
		typ = Real
	case complexString:
		typ = Complex
	case patternString:
		typ = Pattern
	case integerString:
		typ = Integer
	default:
		err = fmt.Errorf("Matrix Market header line type entry incorrect; expected (real | complex | pattern | integer), got %v.", typeString)
		return
	}
	var storage Storage
	switch storageString {
	case generalString:
		storage = General
	case hermitianString:
		storage = Hermitian
	case symmetricString:
		storage = Symmetric
	case skewSymmetricString:
		storage = SkewSymmetric
	default:
		err = fmt.Errorf("Matrix Market header line storage entry incorrect; expected (general | hermitian | symmetric | skew-symmetric), got %v.", storageString)
		return
	}
	if typ == Complex || storage == Hermitian {
		err = fmt.Errorf("Matrix Market complex type currently not supported, got %v %v.", typeString, storageString)
		return
	}

	var grbType GrB.Type

	switch typ {
	case Real:
		grbType = GrB.FP64
	case Integer:
		grbType = GrB.Int64
	case Pattern:
		grbType = GrB.Int8
	}

	if !s.Scan() {
		err = errors.New("Matrix Market second line missing.")
		return
	}

	sText := s.Text()
	var entryType string

	if strings.HasPrefix(sText, "%%GraphBLAS") {
		fields = strings.Fields(sText)
		if len(fields) != 2 {
			err = fmt.Errorf("Matrix Market GraphBLAS incorrect number of elements, expected 2, got %v.", len(fields))
			return
		}
		entryType = fields[1]
		switch entryType {
		case "GrB_BOOL":
			grbType = GrB.Int8
		case "GrB_INT8":
			grbType = GrB.Int8
		case "GrB_INT16":
			grbType = GrB.Int16
		case "GrB_INT32":
			grbType = GrB.Int32
		case "GrB_INT64":
			grbType = GrB.Int64
		case "GrB_UINT8":
			grbType = GrB.Uint8
		case "GrB_UINT16":
			grbType = GrB.Uint16
		case "GrB_UINT32":
			grbType = GrB.Uint32
		case "GrB_UINT64":
			grbType = GrB.Uint64
		case "GrB_FP32":
			grbType = GrB.FP32
		case "GrB_FP64":
			grbType = GrB.FP64
		default:
			err = fmt.Errorf("Matrix Market GraphBLAS type %v not supported or not known.", entryType)
			return
		}
		sText = ""
	}

	if sText == "" || strings.HasPrefix(sText, "%") {
		for s.Scan() {
			sText = s.Text()
			if strings.HasPrefix(sText, "%") {
				sText = ""
				continue
			}
			sText = strings.TrimSpace(sText)
			if sText == "" {
				continue
			}
			break
		}
	}

	if sText == "" {
		err = errors.New("Matrix Market format header line missing.")
		return
	}

	var nrows, nvals, ncols int64

	switch format {
	case Coordinate:
		fields = strings.Fields(sText)
		if len(fields) != 3 {
			err = fmt.Errorf("Matrix Market coordinate header line unexpected number of entries, expected 3, got %v.", len(fields))
			return
		}
		var e error
		nrows, e = strconv.ParseInt(fields[0], 10, 64)
		if e != nil {
			err = fmt.Errorf("Matrix Market coordinate header line nrows parse error %v, while parsing %v.", e, fields[0])
			return
		}
		ncols, e = strconv.ParseInt(fields[1], 10, 64)
		if e != nil {
			err = fmt.Errorf("Matrix Market coordinate header line ncols parse error %v, while parsing %v.", e, fields[1])
			return
		}
		nvals, e = strconv.ParseInt(fields[2], 10, 64)
		if e != nil {
			err = fmt.Errorf("Matrix Market coordinate header line nvals parse error %v, while parsing %v.", e, fields[2])
			return
		}
	case Array:
		fields = strings.Fields(sText)
		if len(fields) != 2 {
			err = fmt.Errorf("Matrix Market array header line unexpected number of entries, expected 2, got %v.", len(fields))
			return
		}
		var e error
		nrows, e = strconv.ParseInt(fields[0], 10, 64)
		if e != nil {
			err = fmt.Errorf("Matrix Market array header line nrows parse error %v, while parsing %v.", e, fields[0])
			return
		}
		ncols, e = strconv.ParseInt(fields[1], 10, 64)
		if e != nil {
			err = fmt.Errorf("Matrix Market array header line ncols parse error %v, while parsing %v.", e, fields[1])
			return
		}
		nvals = nrows * ncols
	}
	return Header{
		Format:  format,
		Type:    typ,
		GrBType: grbType,
		Storage: storage,
		NRows:   int(nrows),
		NCols:   int(ncols),
		NVals:   int(nvals),
	}, s, nil
}
