package smartio

import (
	"errors"
	"io"
)

var (
	ErrInvalidRange = errors.New("invalid range")
)

type RangedReader struct {
	reader io.ReadSeeker
	start  int64
	end    int64

	current int64
}

func NewRangedReader(reader io.ReadSeeker, start int64, end int64) (*RangedReader, error) {
	if start < 0 || end < 0 || start > end {
		return nil, ErrInvalidRange
	}

	_, err := reader.Seek(start, io.SeekStart)
	if err != nil {
		return nil, err
	}

	return &RangedReader{
		reader:  reader,
		start:   start,
		end:     end,
		current: start,
	}, nil
}

func (r *RangedReader) Read(p []byte) (int, error) {
	// Empty range (start == end) returns EOF immediately
	if r.start == r.end && r.current >= r.end {
		return 0, io.EOF
	}

	if r.current > r.end {
		return 0, io.EOF
	}

	// Limit read size to remaining bytes in range (inclusive of end)
	maxRead := r.end - r.current + 1
	if int64(len(p)) > maxRead {
		p = p[:maxRead]
	}

	n, err := r.reader.Read(p)
	r.current += int64(n)

	// If we've reached the end of the range, return EOF
	if r.current > r.end {
		if err == nil {
			err = io.EOF
		}
	} else if err == nil && n < len(p) {
		// If underlying reader returned nil but read fewer bytes than requested,
		// we've hit the end of the underlying data, so return EOF
		err = io.EOF
	}

	return n, err
}
