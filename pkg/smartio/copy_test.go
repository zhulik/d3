package smartio_test

import (
	"bytes"
	"context"
	"errors"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/zhulik/d3/pkg/smartio"
)

// infiniteReader is a reader that never returns EOF, used to test context cancellation.
type infiniteReader struct{}

func (infiniteReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}

	return len(p), nil
}

var _ = Describe("Copy", func() {
	When("reader has finite content", func() {
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

	When("context is cancelled during copy", func() {
		It("returns context.Canceled", func(ctx context.Context) {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			writer := bytes.NewBuffer(nil)
			done := make(chan struct{})

			var copyErr error

			go func() {
				_, _, copyErr = smartio.Copy(ctx, writer, infiniteReader{})

				close(done)
			}()

			cancel()
			<-done

			Expect(copyErr).To(HaveOccurred())
			Expect(errors.Is(copyErr, context.Canceled)).To(BeTrue())
		})
	})

	When("context has deadline exceeded", func() {
		It("returns context.DeadlineExceeded", func(ctx context.Context) {
			ctx, cancel := context.WithTimeout(ctx, 0)
			defer cancel()

			writer := bytes.NewBuffer(nil)
			_, _, err := smartio.Copy(ctx, writer, infiniteReader{})
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, context.DeadlineExceeded)).To(BeTrue())
		})
	})
})
