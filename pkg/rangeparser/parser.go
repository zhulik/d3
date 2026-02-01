package rangeparser

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrInvalidRange = errors.New("invalid range")
)

type Range struct {
	Start int64
	End   int64
}

func Parse(rangeHeader string, contentLength int64) (*Range, error) {
	var (
		r   Range
		err error
	)
	// Remove "bytes=" prefix

	if !strings.HasPrefix(rangeHeader, "bytes=") {
		return nil, fmt.Errorf("%w: invalid range header format", ErrInvalidRange)
	}

	rangeSpec := strings.TrimPrefix(rangeHeader, "bytes=")

	parts := strings.Split(rangeSpec, "-")
	if len(parts) != 2 {
		return nil, fmt.Errorf("%w: invalid range specification", ErrInvalidRange)
	}

	switch {
	case parts[0] == "":
		// Suffix range: bytes=-500 (last 500 bytes)
		suffix, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || suffix <= 0 {
			return nil, fmt.Errorf("%w: invalid suffix range", ErrInvalidRange)
		}

		r.Start = max(contentLength-suffix, 0)

		r.End = contentLength - 1
	case parts[1] == "":
		// Open-ended range: bytes=500- (from 500 to end)
		r.Start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil || r.Start < 0 {
			return nil, fmt.Errorf("%w: invalid start position", ErrInvalidRange)
		}

		if r.Start >= contentLength {
			return nil, fmt.Errorf("%w: start position beyond content length", ErrInvalidRange)
		}

		r.End = contentLength - 1
	default:
		// Normal range: bytes=0-499
		r.Start, err = strconv.ParseInt(parts[0], 10, 64)
		if err != nil || r.Start < 0 {
			return nil, fmt.Errorf("%w: invalid start position", ErrInvalidRange)
		}

		r.End, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil || r.End < r.Start {
			return nil, fmt.Errorf("%w: invalid end position", ErrInvalidRange)
		}
		// Clamp end to content length
		if r.End >= contentLength {
			r.End = contentLength - 1
		}

		if r.Start >= contentLength {
			return nil, fmt.Errorf("%w: start position beyond content length", ErrInvalidRange)
		}
	}

	return &r, nil
}
