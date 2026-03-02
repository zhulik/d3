package s3_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/zhulik/d3/internal/apis/s3"
	"github.com/zhulik/d3/internal/core"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	maxTagCount = 10
	maxTagKey   = 128
	maxTagVal   = 256
)

func TestS3(t *testing.T) {
	t.Parallel()
	RegisterFailHandler(Fail)
	RunSpecs(t, "S3 API Suite")
}

var _ = Describe("ValidateTags", func() {
	When("tags are nil or empty", func() {
		It("returns nil", func() {
			Expect(s3.ValidateTags(nil)).To(Succeed())
			Expect(s3.ValidateTags(map[string]string{})).To(Succeed())
		})
	})

	When("tag count is within limit", func() {
		It("accepts up to 10 tags", func() {
			tags := make(map[string]string, maxTagCount)
			for i := range maxTagCount {
				tags[string(rune('a'+i))] = "v"
			}

			Expect(s3.ValidateTags(tags)).To(Succeed())
		})
	})

	When("more than 10 tags", func() {
		It("returns ErrInvalidTag", func() {
			tags := make(map[string]string, 11)
			for i := range 11 {
				tags[string(rune('a'+i))] = "v"
			}

			err := s3.ValidateTags(tags)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, core.ErrInvalidTag)).To(BeTrue())
		})
	})

	When("tag key exceeds max length", func() {
		It("returns ErrInvalidTag", func() {
			tags := map[string]string{strings.Repeat("k", maxTagKey+1): "v"}
			err := s3.ValidateTags(tags)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, core.ErrInvalidTag)).To(BeTrue())
		})
	})

	When("tag value exceeds max length", func() {
		It("returns ErrInvalidTag", func() {
			tags := map[string]string{"k": strings.Repeat("v", maxTagVal+1)}
			err := s3.ValidateTags(tags)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, core.ErrInvalidTag)).To(BeTrue())
		})
	})

	When("key and value at exact max length", func() {
		It("accepts them", func() {
			tags := map[string]string{
				strings.Repeat("k", maxTagKey): strings.Repeat("v", maxTagVal),
			}
			Expect(s3.ValidateTags(tags)).To(Succeed())
		})
	})
})
