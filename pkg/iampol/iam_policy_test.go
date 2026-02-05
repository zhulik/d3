package iampol_test

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"

	"github.com/zhulik/d3/pkg/iampol"
	"github.com/zhulik/d3/pkg/json"
)

var _ = Describe("Parse", func() {
	// load test cases from testdata/policies.json
	p := filepath.Join("testdata", "policies.json")
	data, err := os.ReadFile(p)
	if err != nil {
		panic(fmt.Sprintf("failed to read test data file %s: %v", p, err))
	}

	type testCase struct {
		Policy json.RawMessage `json:"policy"`
		Error  *string         `json:"error"`
	}

	cases := lo.Must(json.Unmarshal[[]testCase](data))

	entries := lo.Map(cases, func(item testCase, index int) any {
		return Entry(fmt.Sprintf("case %d", index), item.Policy, item.Error)
	})

	entries = slices.Concat([]any{func(policy json.RawMessage, expectedErr *string) {
		_, err := iampol.Parse(policy)
		if expectedErr == nil {
			Expect(err).ToNot(HaveOccurred())
		} else {
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal(*expectedErr))
		}
	}}, entries)

	DescribeTable("table-driven Parse tests", entries...)
})
