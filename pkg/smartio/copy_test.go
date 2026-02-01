package smartio_test

import (
	"bytes"
	"context"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/zhulik/d3/pkg/smartio"
)

var _ = Describe("Copy", func() {
	It("copies the contents of the reader to the writer", func(ctx context.Context) {
		reader := strings.NewReader("hello world")
		writer := bytes.NewBuffer(nil)
		n, sha256sum, err := smartio.Copy(ctx, writer, reader)
		Expect(err).NotTo(HaveOccurred())
		Expect(n).To(Equal(int64(11)))
		Expect(sha256sum).To(Equal("b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"))
		Expect(writer.String()).To(Equal("hello world"))
	})
})
