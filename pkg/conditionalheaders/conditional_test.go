package conditionalheaders_test

import (
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/zhulik/d3/pkg/conditionalheaders"
)

var _ = Describe("Parse", func() {
	When("headers are empty", func() {
		It("returns zero conditionals", func() {
			c := conditionalheaders.Parse(http.Header{})
			Expect(c.IfMatch).To(BeEmpty())
			Expect(c.IfNoneMatch).To(BeEmpty())
			Expect(c.IfModifiedSince).To(BeNil())
			Expect(c.IfUnmodifiedSince).To(BeNil())
		})
	})

	When("conditional headers are set", func() {
		It("parses If-Match and If-None-Match", func() {
			h := http.Header{}
			h.Set("If-Match", "\"abc123\"")
			h.Set("If-None-Match", "*")
			c := conditionalheaders.Parse(h)
			Expect(c.IfMatch).To(Equal("\"abc123\""))
			Expect(c.IfNoneMatch).To(Equal("*"))
			Expect(c.IfModifiedSince).To(BeNil())
			Expect(c.IfUnmodifiedSince).To(BeNil())
		})

		It("parses valid If-Modified-Since", func() {
			h := http.Header{}
			h.Set("If-Modified-Since", "Wed, 21 Oct 2015 07:28:00 GMT")
			c := conditionalheaders.Parse(h)
			Expect(c.IfModifiedSince).NotTo(BeNil())
			Expect(c.IfModifiedSince.UTC().Year()).To(Equal(2015))
			Expect(c.IfModifiedSince.UTC().Month()).To(Equal(time.October))
			Expect(c.IfModifiedSince.UTC().Day()).To(Equal(21))
		})

		It("parses valid If-Unmodified-Since", func() {
			h := http.Header{}
			h.Set("If-Unmodified-Since", "Mon, 02 Jan 2006 15:04:05 GMT")
			c := conditionalheaders.Parse(h)
			Expect(c.IfUnmodifiedSince).NotTo(BeNil())
			Expect(c.IfUnmodifiedSince.UTC().Year()).To(Equal(2006))
		})

		When("date is invalid", func() {
			It("leaves If-Modified-Since nil", func() {
				h := http.Header{}
				h.Set("If-Modified-Since", "not-a-date")
				c := conditionalheaders.Parse(h)
				Expect(c.IfModifiedSince).To(BeNil())
			})

			It("leaves If-Unmodified-Since nil", func() {
				h := http.Header{}
				h.Set("If-Unmodified-Since", "invalid")
				c := conditionalheaders.Parse(h)
				Expect(c.IfUnmodifiedSince).To(BeNil())
			})
		})
	})
})

var _ = Describe("NormalizeETag", func() {
	When("ETag has quotes", func() {
		It("strips surrounding quotes", func() {
			Expect(conditionalheaders.NormalizeETag(`"abc123"`)).To(Equal("abc123"))
		})
	})

	When("ETag has W/ prefix", func() {
		It("strips W/ and quotes", func() {
			Expect(conditionalheaders.NormalizeETag(`W/"abc123"`)).To(Equal("abc123"))
		})
	})

	When("ETag is plain", func() {
		It("returns trimmed value", func() {
			Expect(conditionalheaders.NormalizeETag("  abc123  ")).To(Equal("abc123"))
		})
	})
})

var _ = Describe("ParseHTTPDate", func() {
	When("string is valid HTTP-date", func() {
		It("returns time and true", func() {
			t, ok := conditionalheaders.ParseHTTPDate("Wed, 21 Oct 2015 07:28:00 GMT")
			Expect(ok).To(BeTrue())
			Expect(t.UTC().Year()).To(Equal(2015))
		})
	})

	When("string is invalid", func() {
		It("returns zero time and false", func() {
			t, ok := conditionalheaders.ParseHTTPDate("not-a-date")
			Expect(ok).To(BeFalse())
			Expect(t.IsZero()).To(BeTrue())
		})
	})
})

var _ = Describe("ETagMatches", func() {
	When("client and object ETags match after normalization", func() {
		It("returns true", func() {
			Expect(conditionalheaders.ETagMatches(`"abc123"`, "abc123")).To(BeTrue())
			Expect(conditionalheaders.ETagMatches("abc123", `"abc123"`)).To(BeTrue())
		})
	})

	When("ETags differ", func() {
		It("returns false", func() {
			Expect(conditionalheaders.ETagMatches("abc123", "def456")).To(BeFalse())
		})
	})
})

var _ = Describe("Check", func() {
	const objectETag = "abc123"

	lastModified := time.Date(2025, 3, 9, 12, 0, 0, 0, time.UTC)

	When("no conditionals are set", func() {
		It("returns 200", func() {
			c := conditionalheaders.Conditionals{}
			Expect(c.Check(objectETag, lastModified)).To(Equal(http.StatusOK))
		})
	})

	When("If-Match", func() {
		It("returns 200 when ETag matches", func() {
			c := conditionalheaders.Conditionals{IfMatch: objectETag}
			Expect(c.Check(objectETag, lastModified)).To(Equal(http.StatusOK))
		})

		It("returns 200 when ETag matches with quotes", func() {
			c := conditionalheaders.Conditionals{IfMatch: `"` + objectETag + `"`}
			Expect(c.Check(objectETag, lastModified)).To(Equal(http.StatusOK))
		})

		It("returns 412 when ETag does not match", func() {
			c := conditionalheaders.Conditionals{IfMatch: "wrong-etag"}
			Expect(c.Check(objectETag, lastModified)).To(Equal(http.StatusPreconditionFailed))
		})
	})

	When("If-None-Match", func() {
		It("returns 200 when ETag does not match", func() {
			c := conditionalheaders.Conditionals{IfNoneMatch: "other-etag"}
			Expect(c.Check(objectETag, lastModified)).To(Equal(http.StatusOK))
		})

		It("returns 304 when ETag matches", func() {
			c := conditionalheaders.Conditionals{IfNoneMatch: objectETag}
			Expect(c.Check(objectETag, lastModified)).To(Equal(http.StatusNotModified))
		})

		It("returns 304 when value is *", func() {
			c := conditionalheaders.Conditionals{IfNoneMatch: "*"}
			Expect(c.Check(objectETag, lastModified)).To(Equal(http.StatusNotModified))
		})
	})

	When("If-Modified-Since", func() {
		It("returns 200 when object is modified after the date", func() {
			past := lastModified.Add(-time.Hour)
			c := conditionalheaders.Conditionals{IfModifiedSince: &past}
			Expect(c.Check(objectETag, lastModified)).To(Equal(http.StatusOK))
		})

		It("returns 304 when object is not modified after the date", func() {
			future := lastModified.Add(time.Hour)
			c := conditionalheaders.Conditionals{IfModifiedSince: &future}
			Expect(c.Check(objectETag, lastModified)).To(Equal(http.StatusNotModified))
		})
	})

	When("If-Unmodified-Since", func() {
		It("returns 200 when object is not modified after the date", func() {
			future := lastModified.Add(time.Hour)
			c := conditionalheaders.Conditionals{IfUnmodifiedSince: &future}
			Expect(c.Check(objectETag, lastModified)).To(Equal(http.StatusOK))
		})

		It("returns 412 when object is modified after the date", func() {
			past := lastModified.Add(-time.Hour)
			c := conditionalheaders.Conditionals{IfUnmodifiedSince: &past}
			Expect(c.Check(objectETag, lastModified)).To(Equal(http.StatusPreconditionFailed))
		})
	})
})
