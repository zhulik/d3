package wld_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/zhulik/d3/pkg/wld"
)

func TestWld(t *testing.T) {
	t.Parallel()

	RegisterFailHandler(Fail)
	RunSpecs(t, "Wld Suite")
}

var _ = Describe("Match", func() {
	DescribeTable("wildcard patterns",
		func(p, s string, want bool) {
			Expect(wld.Match(p, s)).To(Equal(want))
		},
		Entry("* matches anything", "*", "anything", true),
		Entry("? matches single char", "?", "a", true),
		Entry("? does not match two chars", "?", "ab", false),
		Entry("a?c matches abc", "a?c", "abc", true),
		Entry("a*c matches abbbbbc", "a*c", "abbbbbc", true),
		Entry("a*c matches ac", "a*c", "ac", true),
		Entry("a*?c matches abbbc", "a*?c", "abbbc", true),
		Entry("a*?c does not match ac", "a*?c", "ac", false),
		Entry("*a matches ba", "*a", "ba", true),
		Entry("*a does not match b", "*a", "b", false),
		Entry("ab*cd?e matches abxxxxcdXe", "ab*cd?e", "abxxxxcdXe", true),
		Entry("ab*cd?e matches abxcdXe", "ab*cd?e", "abxcdXe", true),
		Entry("ab*cd?e matches abxcdYe", "ab*cd?e", "abxcdYe", true),
		Entry("empty pattern matches empty string", "", "", true),
		Entry("empty pattern does not match non-empty", "", "a", false),
		Entry("* matches empty", "*", "", true),
		Entry("a*b*c matches axbyc", "a*b*c", "axbyc", true),
		Entry("a*b*c matches abc", "a*b*c", "abc", true),
		Entry("*?*?* matches ab", "*?*?*", "ab", true),
		Entry("*?*?* does not match a", "*?*?*", "a", false),
		// Add a descriptive entry for debugging
		Entry(func() string { return fmt.Sprintf("pattern=%q str=%q", "*", "anything") }(), "*", "anything", true),
	)
})
