package smartio_test

import (
	"bytes"
	"errors"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/zhulik/d3/pkg/smartio"
)

var _ = Describe("RangedReader", func() {
	data := []byte("0123456789abcdefghij")

	Describe("NewRangedReader", func() {
		It("creates a reader positioned at start", func() {
			rr, err := smartio.NewRangedReader(bytes.NewReader(data), 5, 10)
			Expect(err).NotTo(HaveOccurred())
			Expect(rr).NotTo(BeNil())

			buf := make([]byte, 5)
			n, err := rr.Read(buf)
			Expect(err).To(Equal(io.EOF))
			Expect(n).To(Equal(5))
			Expect(string(buf)).To(Equal("56789"))
		})

		It("returns error if seek fails", func() {
			failingReader := &failingSeeker{reader: bytes.NewReader(data)}
			rr, err := smartio.NewRangedReader(failingReader, 5, 10)
			Expect(err).To(HaveOccurred())
			Expect(rr).To(BeNil())
		})

		Context("invalid range", func() {
			DescribeTable("returns ErrInvalidRange",
				func(start, end int64, description string) {
					rr, err := smartio.NewRangedReader(bytes.NewReader(data), start, end)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, smartio.ErrInvalidRange)).To(BeTrue())
					Expect(rr).To(BeNil())
				},
				Entry("when start is negative", int64(-1), int64(10), "start is negative"),
				Entry("when end is negative", int64(0), int64(-1), "end is negative"),
				Entry("when start is greater than end", int64(10), int64(5), "start > end"),
				Entry("when both start and end are negative", int64(-5), int64(-1), "both negative"),
				Entry("when start equals end but both are negative", int64(-5), int64(-5), "both negative and equal"),
			)
		})
	})

	Describe("Read", func() {
		Context("basic functionality", func() {
			DescribeTable("reads correctly",
				func(start, end int64, bufSize int, expectedN int, expectedErr error, expectedContent string) {
					rr, err := smartio.NewRangedReader(bytes.NewReader(data), start, end)
					Expect(err).NotTo(HaveOccurred())

					buf := make([]byte, bufSize)
					n, err := rr.Read(buf)
					if expectedErr == nil {
						Expect(err).To(BeNil())
					} else {
						Expect(err).To(Equal(expectedErr))
					}
					Expect(n).To(Equal(expectedN))
					if expectedN > 0 {
						Expect(string(buf[:n])).To(Equal(expectedContent))
					}
				},
				Entry("reads within range", int64(5), int64(10), 3, 3, nil, "567"),
				Entry("reads exactly to the end", int64(5), int64(10), 5, 5, io.EOF, "56789"),
				Entry("stops reading at end when buffer is larger", int64(5), int64(10), 10, 5, io.EOF, "56789"),
			)
		})

		Context("multiple reads", func() {
			It("handles multiple sequential reads", func() {
				rr, err := smartio.NewRangedReader(bytes.NewReader(data), 5, 10)
				Expect(err).NotTo(HaveOccurred())

				buf1 := make([]byte, 2)
				n1, err1 := rr.Read(buf1)
				Expect(err1).NotTo(HaveOccurred())
				Expect(n1).To(Equal(2))
				Expect(string(buf1)).To(Equal("56"))

				buf2 := make([]byte, 2)
				n2, err2 := rr.Read(buf2)
				Expect(err2).NotTo(HaveOccurred())
				Expect(n2).To(Equal(2))
				Expect(string(buf2)).To(Equal("78"))

				buf3 := make([]byte, 2)
				n3, err3 := rr.Read(buf3)
				Expect(err3).To(Equal(io.EOF))
				Expect(n3).To(Equal(1))
				Expect(string(buf3[:n3])).To(Equal("9"))
			})

			It("handles reads that span the boundary", func() {
				rr, err := smartio.NewRangedReader(bytes.NewReader(data), 5, 10)
				Expect(err).NotTo(HaveOccurred())

				// First read: within range
				buf1 := make([]byte, 3)
				n1, err1 := rr.Read(buf1)
				Expect(err1).NotTo(HaveOccurred())
				Expect(n1).To(Equal(3))
				Expect(string(buf1)).To(Equal("567"))

				// Second read: spans boundary (3 bytes remaining, but buffer is 5)
				buf2 := make([]byte, 5)
				n2, err2 := rr.Read(buf2)
				Expect(err2).To(Equal(io.EOF))
				Expect(n2).To(Equal(2))
				Expect(string(buf2[:n2])).To(Equal("89"))
			})
		})

		Context("edge cases", func() {
			DescribeTable("handles range boundaries",
				func(start, end int64, expectedN int, expectedContent string, description string) {
					rr, err := smartio.NewRangedReader(bytes.NewReader(data), start, end)
					Expect(err).NotTo(HaveOccurred())

					buf := make([]byte, 10)
					n, err := rr.Read(buf)
					Expect(err).To(Equal(io.EOF))
					Expect(n).To(Equal(expectedN))
					if expectedN > 0 {
						Expect(string(buf[:n])).To(Equal(expectedContent))
					}
				},
				Entry("empty range (start == end)", int64(5), int64(5), 0, "", "empty range"),
				Entry("single byte range", int64(5), int64(6), 1, "5", "single byte"),
				Entry("reading from start of file", int64(0), int64(5), 5, "01234", "start of file"),
				Entry("reading to end of file", int64(15), int64(20), 5, "fghij", "end of file"),
			)

			It("returns EOF on subsequent reads after reaching end", func() {
				rr, err := smartio.NewRangedReader(bytes.NewReader(data), 5, 10)
				Expect(err).NotTo(HaveOccurred())

				// Read to end
				buf1 := make([]byte, 10)
				n1, err1 := rr.Read(buf1)
				Expect(err1).To(Equal(io.EOF))
				Expect(n1).To(Equal(5))

				// Try to read again
				buf2 := make([]byte, 10)
				n2, err2 := rr.Read(buf2)
				Expect(err2).To(Equal(io.EOF))
				Expect(n2).To(Equal(0))
			})

			It("handles small buffer reads", func() {
				rr, err := smartio.NewRangedReader(bytes.NewReader(data), 5, 10)
				Expect(err).NotTo(HaveOccurred())

				// Read with small buffer multiple times
				var result []byte
				for range 10 {
					buf := make([]byte, 1)
					n, err := rr.Read(buf)
					if errors.Is(err, io.EOF) {
						result = append(result, buf[:n]...)

						break
					}
					Expect(err).NotTo(HaveOccurred())
					result = append(result, buf[:n]...)
				}
				Expect(string(result)).To(Equal("56789"))
			})
		})

		Context("error handling", func() {
			It("preserves underlying reader errors", func() {
				// Create a reader that will return an error after the first read
				errorReader := &errorReader{
					reader:    bytes.NewReader(data),
					failAfter: 1,
				}
				rr, err := smartio.NewRangedReader(errorReader, 5, 10)
				Expect(err).NotTo(HaveOccurred())

				buf1 := make([]byte, 1)
				n1, err1 := rr.Read(buf1)
				Expect(err1).NotTo(HaveOccurred())
				Expect(n1).To(Equal(1))

				buf2 := make([]byte, 1)
				n2, err2 := rr.Read(buf2)
				// The errorReader will fail on the second read
				Expect(err2).To(HaveOccurred())
				Expect(err2).NotTo(Equal(io.EOF))
				Expect(n2).To(Equal(0))
			})

			It("handles EOF from underlying reader correctly", func() {
				// Range extends beyond available data
				rr, err := smartio.NewRangedReader(bytes.NewReader(data), 15, 25)
				Expect(err).NotTo(HaveOccurred())

				buf := make([]byte, 10)
				n, err := rr.Read(buf)
				// Should get EOF (either from underlying reader or from range)
				// The underlying reader will return EOF when reading past end
				// but we should still return the data we read
				Expect(err).To(Equal(io.EOF))
				Expect(n).To(Equal(5))
				Expect(string(buf[:n])).To(Equal("fghij"))
			})
		})

		Context("boundary conditions", func() {
			DescribeTable("handles boundary positions",
				func(start, end int64, expectedN int, expectedContent string, description string) {
					rr, err := smartio.NewRangedReader(bytes.NewReader(data), start, end)
					Expect(err).NotTo(HaveOccurred())

					buf := make([]byte, 100)
					n, err := rr.Read(buf)
					Expect(err).To(Equal(io.EOF))
					Expect(n).To(Equal(expectedN))
					Expect(string(buf[:n])).To(Equal(expectedContent))
				},
				Entry("range at very start", int64(0), int64(1), 1, "0", "very start"),
				Entry("range at very end", int64(19), int64(20), 1, "j", "very end"),
				Entry("full range", int64(0), int64(len(data)), len(data), string(data), "full range"),
			)
		})

		Context("zero-length reads", func() {
			It("handles zero-length buffer", func() {
				rr, err := smartio.NewRangedReader(bytes.NewReader(data), 5, 10)
				Expect(err).NotTo(HaveOccurred())

				buf := make([]byte, 0)
				n, err := rr.Read(buf)
				Expect(err).NotTo(HaveOccurred())
				Expect(n).To(Equal(0))
			})
		})
	})
})

// Helper types for testing

type failingSeeker struct {
	reader io.ReadSeeker
}

func (f *failingSeeker) Read(p []byte) (int, error) {
	return f.reader.Read(p)
}

func (f *failingSeeker) Seek(_ int64, _ int) (int64, error) {
	return 0, io.ErrUnexpectedEOF
}

type errorReader struct {
	reader    io.ReadSeeker
	readCount int
	failAfter int
}

func (e *errorReader) Read(p []byte) (int, error) {
	e.readCount++
	if e.readCount > e.failAfter {
		return 0, io.ErrUnexpectedEOF
	}

	n, err := e.reader.Read(p)

	return n, err
}

func (e *errorReader) Seek(offset int64, whence int) (int64, error) {
	// Reset read count on seek to allow fresh reads
	e.readCount = 0

	return e.reader.Seek(offset, whence)
}
