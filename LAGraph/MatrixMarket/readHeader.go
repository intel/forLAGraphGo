package MatrixMarket

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/intel/forGraphBLASGo/GrB"
	"io"
	"math"
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
	format  int
	typ     int
	storage int
)

const (
	coordinate format = iota
	array
)

const (
	treal typ = iota
	tcomplex
	tpattern
	tinteger
)

const (
	general storage = iota
	hermitian
	symmetric
	skewSymmetric
)

type header struct {
	format              format
	typ                 typ
	grbType             GrB.Type
	storage             storage
	nrows, ncols, nvals int
}

func readHeader(r io.Reader) (hdr header, scanner *bufio.Scanner, err error) {
	s := bufio.NewScanner(r)
	if !s.Scan() {
		err = errors.New("MatrixMarket header line missing")
		return
	}
	fields := strings.Fields(s.Text())
	if len(fields) != 5 {
		err = fmt.Errorf("MatrixMarket incorrect number of header line elements; expected 5, got %v", len(fields))
		return
	}
	if fields[0] != "%%MatrixMarket" {
		err = fmt.Errorf("MatrixMarket header line prefix missing; expected %%MatrixMarket, got %v", fields[0])
		return
	}
	if fields[1] != "matrix" {
		err = fmt.Errorf("MatrixMarket header line second entry incorrect; expected matrix, got %v", fields[1])
		return
	}
	formatString, typeString, storageString := strings.ToLower(fields[2]), strings.ToLower(fields[3]), strings.ToLower(fields[4])
	var mmFormat format
	switch formatString {
	case coordinateString:
		mmFormat = coordinate
	case arrayString:
		mmFormat = array
	default:
		err = fmt.Errorf("MatrixMarket header line format entry incorrect; expected (coordinate | array), got %v", formatString)
		return
	}
	var mmType typ
	switch typeString {
	case realString:
		mmType = treal
	case complexString:
		mmType = tcomplex
	case patternString:
		mmType = tpattern
	case integerString:
		mmType = tinteger
	default:
		err = fmt.Errorf("MatrixMarket header line type entry incorrect; expected (real | complex | pattern | integer), got %v", typeString)
		return
	}
	var mmStorage storage
	switch storageString {
	case generalString:
		mmStorage = general
	case hermitianString:
		mmStorage = hermitian
	case symmetricString:
		mmStorage = symmetric
	case skewSymmetricString:
		mmStorage = skewSymmetric
	default:
		err = fmt.Errorf("MatrixMarket header line storage entry incorrect; expected (general | hermitian | symmetric | skew-symmetric), got %v", storageString)
		return
	}
	if mmType == tcomplex || mmStorage == hermitian {
		err = fmt.Errorf("MatrixMarket complex type currently not supported, got %v %v", typeString, storageString)
		return
	}

	var grbType GrB.Type

	switch mmType {
	case treal:
		grbType = GrB.Float64
	case tinteger:
		grbType = GrB.Int64
	case tpattern:
		grbType = GrB.Int8
	}

	if !s.Scan() {
		err = errors.New("MatrixMarket second line missing")
		return
	}

	sText := s.Text()
	var entryType string

	if strings.HasPrefix(sText, "%%GraphBLAS") {
		fields = strings.Fields(sText)
		if len(fields) != 2 {
			err = fmt.Errorf("MatrixMarket GraphBLAS incorrect number of elements, expected 2, got %v", len(fields))
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
			grbType = GrB.Float32
		case "GrB_FP64":
			grbType = GrB.Float64
		default:
			err = fmt.Errorf("MatrixMarket GraphBLAS type %v not supported or not known", entryType)
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
		err = errors.New("MatrixMarket format header line missing")
		return
	}

	var nrows, nvals, ncols int64

	switch mmFormat {
	case coordinate:
		fields = strings.Fields(sText)
		if len(fields) != 3 {
			err = fmt.Errorf("MatrixMarket coordinate header line unexpected number of entries, expected 3, got %v", len(fields))
			return
		}
		var e error
		nrows, e = strconv.ParseInt(fields[0], 10, 64)
		if e != nil {
			err = fmt.Errorf("MatrixMarket coordinate header line nrows parse error %v, while parsing %v", e, fields[0])
			return
		}
		ncols, e = strconv.ParseInt(fields[1], 10, 64)
		if e != nil {
			err = fmt.Errorf("MatrixMarket coordinate header line ncols parse error %v, while parsing %v", e, fields[1])
			return
		}
		nvals, e = strconv.ParseInt(fields[2], 10, 64)
		if e != nil {
			err = fmt.Errorf("MatrixMarket coordinate header line nvals parse error %v, while parsing %v", e, fields[2])
			return
		}
	case array:
		fields = strings.Fields(sText)
		if len(fields) != 2 {
			err = fmt.Errorf("MatrixMarket array header line unexpected number of entries, expected 2, got %v", len(fields))
			return
		}
		var e error
		nrows, e = strconv.ParseInt(fields[0], 10, 64)
		if e != nil {
			err = fmt.Errorf("MatrixMarket array header line nrows parse error %v, while parsing %v", e, fields[0])
			return
		}
		ncols, e = strconv.ParseInt(fields[1], 10, 64)
		if e != nil {
			err = fmt.Errorf("MatrixMarket array header line ncols parse error %v, while parsing %v", e, fields[1])
			return
		}
		nvals = nrows * ncols
	}
	if nrows < 1 || nrows > math.MaxInt {
		err = fmt.Errorf("MatrixMarket header line nrows out of range %v", nrows)
		return
	}
	if ncols < 1 || ncols > math.MaxInt {
		err = fmt.Errorf("MatrixMarket header line ncols out of range %v", ncols)
		return
	}
	if nvals > math.MaxInt {
		err = fmt.Errorf("MatrixMarket header line nvals out of range %v", nvals)
	}
	return header{
		format:  mmFormat,
		typ:     mmType,
		grbType: grbType,
		storage: mmStorage,
		nrows:   int(nrows),
		ncols:   int(ncols),
		nvals:   int(nvals),
	}, s, nil
}
