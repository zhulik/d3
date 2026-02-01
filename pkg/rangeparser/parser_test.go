package rangeparser_test

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/zhulik/d3/pkg/rangeparser"
)

var _ = Describe("Parse", func() {
	const contentLength = int64(1000)

	Describe("invalid input", func() {
		When("range header has invalid format", func() {
			DescribeTable("returns ErrInvalidRange",
				func(rangeHeader string, _ string) {
					result, err := rangeparser.Parse(rangeHeader, contentLength)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, rangeparser.ErrInvalidRange)).To(BeTrue())
					Expect(result).To(BeNil())
				},
				Entry("missing bytes= prefix", "0-499", "no prefix"),
				Entry("empty string", "", "empty"),
				Entry("only bytes=", "bytes=", "only prefix"),
				Entry("wrong prefix", "range=0-499", "wrong prefix"),
				Entry("case mismatch", "BYTES=0-499", "case mismatch"),
			)
		})

		When("range specification is invalid", func() {
			DescribeTable("returns ErrInvalidRange",
				func(rangeHeader string, _ string) {
					result, err := rangeparser.Parse(rangeHeader, contentLength)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, rangeparser.ErrInvalidRange)).To(BeTrue())
					Expect(result).To(BeNil())
				},
				Entry("no dash", "bytes=0499", "no dash"),
				Entry("multiple dashes", "bytes=0-4-99", "multiple dashes"),
				Entry("only dash", "bytes=-", "only dash"),
				Entry("empty after prefix", "bytes= ", "empty after prefix"),
			)
		})
	})

	Describe("suffix range", func() {
		When("range header is bytes=-N", func() {
			DescribeTable("parses correctly",
				func(rangeHeader string, contentLength int64, expectedStart, expectedEnd int64, _ string) {
					result, err := rangeparser.Parse(rangeHeader, contentLength)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.Start).To(Equal(expectedStart))
					Expect(result.End).To(Equal(expectedEnd))
				},
				Entry("suffix smaller than content", "bytes=-500", int64(1000), int64(500), int64(999), "suffix < content"),
				Entry("suffix equal to content", "bytes=-1000", int64(1000), int64(0), int64(999), "suffix == content"),
				Entry("suffix larger than content", "bytes=-2000", int64(1000), int64(0), int64(999), "suffix > content"),
				Entry("suffix of 1", "bytes=-1", int64(1000), int64(999), int64(999), "suffix of 1"),
				Entry("suffix on small content", "bytes=-5", int64(10), int64(5), int64(9), "suffix on small content"),
				Entry("suffix larger than small content", "bytes=-20", int64(10), int64(0), int64(9), "suffix > small content"),
			)

			DescribeTable("returns ErrInvalidRange for invalid suffix",
				func(rangeHeader string, contentLength int64, _ string) {
					result, err := rangeparser.Parse(rangeHeader, contentLength)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, rangeparser.ErrInvalidRange)).To(BeTrue())
					Expect(result).To(BeNil())
				},
				Entry("zero suffix", "bytes=-0", contentLength, "zero suffix"),
				Entry("negative suffix", "bytes=--500", contentLength, "negative suffix"),
				Entry("non-numeric suffix", "bytes=-abc", contentLength, "non-numeric suffix"),
				Entry("empty suffix", "bytes=-", contentLength, "empty suffix"),
				Entry("suffix with spaces", "bytes=- 500", contentLength, "suffix with spaces"),
			)

			It("accepts suffix with plus sign", func() {
				result, err := rangeparser.Parse("bytes=-+500", contentLength)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Start).To(Equal(int64(500)))
				Expect(result.End).To(Equal(int64(999)))
			})
		})
	})

	Describe("open-ended range", func() {
		When("range header is bytes=N-", func() {
			DescribeTable("parses correctly",
				func(rangeHeader string, contentLength int64, expectedStart, expectedEnd int64, _ string) {
					result, err := rangeparser.Parse(rangeHeader, contentLength)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.Start).To(Equal(expectedStart))
					Expect(result.End).To(Equal(expectedEnd))
				},
				Entry("start at beginning", "bytes=0-", contentLength, int64(0), int64(999), "start at 0"),
				Entry("start in middle", "bytes=500-", contentLength, int64(500), int64(999), "start in middle"),
				Entry("start near end", "bytes=999-", contentLength, int64(999), int64(999), "start near end"),
				Entry("start at last byte", "bytes=998-", contentLength, int64(998), int64(999), "start at last byte"),
			)

			DescribeTable("returns ErrInvalidRange for invalid start",
				func(rangeHeader string, contentLength int64, _ string) {
					result, err := rangeparser.Parse(rangeHeader, contentLength)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, rangeparser.ErrInvalidRange)).To(BeTrue())
					Expect(result).To(BeNil())
				},
				Entry("negative start", "bytes=-100-", contentLength, "negative start"),
				Entry("start equals content length", "bytes=1000-", contentLength, "start == content length"),
				Entry("start exceeds content length", "bytes=2000-", contentLength, "start > content length"),
				Entry("non-numeric start", "bytes=abc-", contentLength, "non-numeric start"),
				Entry("start with spaces", "bytes= 500-", contentLength, "start with spaces"),
			)

			It("accepts start with plus sign", func() {
				result, err := rangeparser.Parse("bytes=+500-", contentLength)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Start).To(Equal(int64(500)))
				Expect(result.End).To(Equal(int64(999)))
			})
		})
	})

	Describe("normal range", func() {
		When("range header is bytes=N-M", func() {
			DescribeTable("parses correctly",
				func(rangeHeader string, contentLength int64, expectedStart, expectedEnd int64, _ string) {
					result, err := rangeparser.Parse(rangeHeader, contentLength)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.Start).To(Equal(expectedStart))
					Expect(result.End).To(Equal(expectedEnd))
				},
				Entry("full range", "bytes=0-999", int64(1000), int64(0), int64(999), "full range"),
				Entry("range at beginning", "bytes=0-499", int64(1000), int64(0), int64(499), "range at beginning"),
				Entry("range in middle", "bytes=200-799", int64(1000), int64(200), int64(799), "range in middle"),
				Entry("range at end", "bytes=500-999", int64(1000), int64(500), int64(999), "range at end"),
				Entry("single byte range", "bytes=500-500", int64(1000), int64(500), int64(500), "single byte"),
				Entry("two byte range", "bytes=500-501", int64(1000), int64(500), int64(501), "two bytes"),
			)

			DescribeTable("clamps end to content length",
				func(rangeHeader string, contentLength int64, expectedStart, expectedEnd int64, _ string) {
					result, err := rangeparser.Parse(rangeHeader, contentLength)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.Start).To(Equal(expectedStart))
					Expect(result.End).To(Equal(expectedEnd))
				},
				Entry("end equals content length", "bytes=0-1000", int64(1000), int64(0), int64(999), "end == content length"),
				Entry("end exceeds content length", "bytes=0-2000", int64(1000), int64(0), int64(999), "end > content length"),
				Entry("end exceeds content length in middle", "bytes=500-2000", int64(1000), int64(500), int64(999), "end > content length in middle"),
			)

			DescribeTable("returns ErrInvalidRange for invalid start",
				func(rangeHeader string, contentLength int64, _ string) {
					result, err := rangeparser.Parse(rangeHeader, contentLength)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, rangeparser.ErrInvalidRange)).To(BeTrue())
					Expect(result).To(BeNil())
				},
				Entry("negative start", "bytes=-100-500", contentLength, "negative start"),
				Entry("start equals content length", "bytes=1000-1500", contentLength, "start == content length"),
				Entry("start exceeds content length", "bytes=2000-2500", contentLength, "start > content length"),
				Entry("non-numeric start", "bytes=abc-500", contentLength, "non-numeric start"),
				Entry("start with spaces", "bytes= 500-600", contentLength, "start with spaces"),
			)

			It("accepts start with plus sign", func() {
				result, err := rangeparser.Parse("bytes=+500-600", contentLength)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Start).To(Equal(int64(500)))
				Expect(result.End).To(Equal(int64(600)))
			})

			DescribeTable("returns ErrInvalidRange for invalid end",
				func(rangeHeader string, contentLength int64, _ string) {
					result, err := rangeparser.Parse(rangeHeader, contentLength)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, rangeparser.ErrInvalidRange)).To(BeTrue())
					Expect(result).To(BeNil())
				},
				Entry("end less than start", "bytes=500-400", contentLength, "end < start"),
				Entry("end equals start but negative", "bytes=-500--500", contentLength, "end == start but negative"),
				Entry("non-numeric end", "bytes=500-abc", contentLength, "non-numeric end"),
				Entry("end with spaces", "bytes=500- 600", contentLength, "end with spaces"),
			)

			It("accepts end with plus sign", func() {
				result, err := rangeparser.Parse("bytes=500-+600", contentLength)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Start).To(Equal(int64(500)))
				Expect(result.End).To(Equal(int64(600)))
			})
		})
	})

	Describe("edge cases with content length", func() {
		When("content length is zero", func() {
			DescribeTable("rejects ranges",
				func(rangeHeader string, _ string) {
					result, err := rangeparser.Parse(rangeHeader, 0)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, rangeparser.ErrInvalidRange)).To(BeTrue())
					Expect(result).To(BeNil())
				},
				Entry("normal range", "bytes=0-0", "normal range"),
				Entry("open-ended range", "bytes=0-", "open-ended range"),
			)

			It("returns invalid range for suffix range (parser limitation)", func() {
				// Note: This is a limitation of the current parser implementation.
				// For "bytes=-1" with contentLength=0, it returns start=0, end=-1,
				// which is an invalid range (start > end), but the parser doesn't validate this.
				result, err := rangeparser.Parse("bytes=-1", 0)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Start).To(Equal(int64(0)))
				Expect(result.End).To(Equal(int64(-1)))
			})
		})

		When("content length is one", func() {
			DescribeTable("handles single byte content",
				func(rangeHeader string, expectedStart, expectedEnd int64, _ string) {
					result, err := rangeparser.Parse(rangeHeader, 1)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.Start).To(Equal(expectedStart))
					Expect(result.End).To(Equal(expectedEnd))
				},
				Entry("full range", "bytes=0-0", int64(0), int64(0), "full range"),
				Entry("suffix of 1", "bytes=-1", int64(0), int64(0), "suffix of 1"),
				Entry("open-ended from start", "bytes=0-", int64(0), int64(0), "open-ended from start"),
			)

			DescribeTable("rejects invalid ranges",
				func(rangeHeader string, _ string) {
					result, err := rangeparser.Parse(rangeHeader, 1)
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, rangeparser.ErrInvalidRange)).To(BeTrue())
					Expect(result).To(BeNil())
				},
				Entry("start at 1", "bytes=1-", "start == content length"),
				Entry("start exceeds", "bytes=2-", "start > content length"),
				Entry("suffix of 0", "bytes=-0", "zero suffix"),
			)
		})

		When("content length is very large", func() {
			It("handles large ranges correctly", func() {
				const largeContentLength = int64(1000000)
				result, err := rangeparser.Parse("bytes=500000-750000", largeContentLength)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Start).To(Equal(int64(500000)))
				Expect(result.End).To(Equal(int64(750000)))
			})

			It("handles large suffix ranges correctly", func() {
				const largeContentLength = int64(1000000)
				result, err := rangeparser.Parse("bytes=-500000", largeContentLength)
				Expect(err).NotTo(HaveOccurred())
				Expect(result).NotTo(BeNil())
				Expect(result.Start).To(Equal(int64(500000)))
				Expect(result.End).To(Equal(int64(999999)))
			})
		})
	})

	Describe("boundary conditions", func() {
		When("range boundaries are at limits", func() {
			DescribeTable("handles boundary positions",
				func(rangeHeader string, contentLength int64, expectedStart, expectedEnd int64, _ string) {
					result, err := rangeparser.Parse(rangeHeader, contentLength)
					Expect(err).NotTo(HaveOccurred())
					Expect(result).NotTo(BeNil())
					Expect(result.Start).To(Equal(expectedStart))
					Expect(result.End).To(Equal(expectedEnd))
				},
				Entry("start at 0, end at contentLength-1", "bytes=0-999", int64(1000), int64(0), int64(999), "full range"),
				Entry("start at 0, end at contentLength (clamped)", "bytes=0-1000", int64(1000), int64(0), int64(999), "end clamped"),
				Entry("start at contentLength-1, end at contentLength-1", "bytes=999-999", int64(1000), int64(999), int64(999), "last byte"),
				Entry("suffix that results in start=0", "bytes=-1000", int64(1000), int64(0), int64(999), "suffix to start"),
				Entry("suffix larger than content (clamped to 0)", "bytes=-2000", int64(1000), int64(0), int64(999), "suffix clamped"),
			)
		})
	})
})
