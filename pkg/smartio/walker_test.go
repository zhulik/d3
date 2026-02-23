package smartio_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/zhulik/d3/pkg/smartio"
)

// makeWalkerTree builds the test tree. Structure:
//
//	root/
//	  file_at_root.txt
//	  first/second/third/{file.txt, another_file.txt}
//	  first/second/yet_another_file.txt
//	  first/second/second_extra/x.txt
//	  first/first_other/y.txt
//	  foo/bar.txt
//	  baz/que.txt
//	  a/b/c/deep.txt
//	  empty_dir/
//	  prefix/sub/file.txt
func makeWalkerTree(root string) {
	lo.Must0(os.WriteFile(filepath.Join(root, "file_at_root.txt"), nil, 0o644))
	lo.Must0(os.MkdirAll(filepath.Join(root, "first", "second", "third"), 0o755))
	lo.Must0(os.WriteFile(filepath.Join(root, "first", "second", "third", "file.txt"), nil, 0o644))
	lo.Must0(os.WriteFile(filepath.Join(root, "first", "second", "third", "another_file.txt"), nil, 0o644))
	lo.Must0(os.WriteFile(filepath.Join(root, "first", "second", "yet_another_file.txt"), nil, 0o644))
	lo.Must0(os.MkdirAll(filepath.Join(root, "first", "second", "second_extra"), 0o755))
	lo.Must0(os.WriteFile(filepath.Join(root, "first", "second", "second_extra", "x.txt"), nil, 0o644))
	lo.Must0(os.MkdirAll(filepath.Join(root, "first", "first_other"), 0o755))
	lo.Must0(os.WriteFile(filepath.Join(root, "first", "first_other", "y.txt"), nil, 0o644))
	lo.Must0(os.MkdirAll(filepath.Join(root, "foo"), 0o755))
	lo.Must0(os.WriteFile(filepath.Join(root, "foo", "bar.txt"), nil, 0o644))
	lo.Must0(os.MkdirAll(filepath.Join(root, "baz"), 0o755))
	lo.Must0(os.WriteFile(filepath.Join(root, "baz", "que.txt"), nil, 0o644))
	lo.Must0(os.MkdirAll(filepath.Join(root, "a", "b", "c"), 0o755))
	lo.Must0(os.WriteFile(filepath.Join(root, "a", "b", "c", "deep.txt"), nil, 0o644))
	lo.Must0(os.MkdirAll(filepath.Join(root, "empty_dir"), 0o755))
	lo.Must0(os.MkdirAll(filepath.Join(root, "prefix", "sub"), 0o755))
	lo.Must0(os.WriteFile(filepath.Join(root, "prefix", "sub", "file.txt"), nil, 0o644))
}

func joinRoot(root string, rel string) string {
	if rel == "" {
		return root
	}

	return filepath.Join(root, rel)
}

var errStopWalk = errors.New("stop")

var _ = Describe("WalkDir", func() {
	var root, outsideRoot string

	BeforeEach(func() {
		root = lo.Must(os.MkdirTemp("", "walkdir-*"))

		DeferCleanup(func() { _ = os.RemoveAll(root) })
		makeWalkerTree(root)

		outsideRoot = lo.Must(os.MkdirTemp("", "walkdir-outside-*"))

		DeferCleanup(func() { _ = os.RemoveAll(outsideRoot) })
		lo.Must0(os.WriteFile(filepath.Join(outsideRoot, "x"), nil, 0o644))
	})

	DescribeTable("walk and error cases",
		func(ctx context.Context, prefix, startFromRel string, expectedExactRel []string, expectErr error) {
			walkCtx := ctx
			if expectErr != nil && errors.Is(expectErr, context.Canceled) {
				cancelled, cancel := context.WithCancel(ctx)
				cancel()

				walkCtx = cancelled
			}

			var startFrom *string

			switch startFromRel {
			case "":
				// no startFrom
			case "__outside__":
				startFrom = lo.ToPtr(filepath.Join(outsideRoot, "x"))
			case "__root__":
				startFrom = lo.ToPtr(root)
			default:
				startFrom = lo.ToPtr(filepath.Join(root, startFromRel))
			}

			var got []string

			err := smartio.WalkDir(walkCtx, root, prefix, startFrom, func(path string) error {
				got = append(got, path)

				return nil
			})
			if expectErr != nil {
				Expect(err).To(MatchError(expectErr))

				return
			}

			Expect(err).NotTo(HaveOccurred())

			expectedExact := lo.Map(expectedExactRel, func(rel string, _ int) string {
				return joinRoot(root, rel)
			})
			Expect(got).To(HaveLen(len(expectedExact)))

			if len(expectedExact) > 0 {
				Expect(got).To(Equal(expectedExact))
			}
		},
		Entry("invokes fn for every file and directory when prefix is empty and startFrom is nil",
			"", "",
			[]string{"", "a", "a/b", "a/b/c", "a/b/c/deep.txt", "baz", "baz/que.txt", "empty_dir", "file_at_root.txt", "first", "first/first_other", "first/first_other/y.txt", "first/second", "first/second/second_extra", "first/second/second_extra/x.txt", "first/second/third", "first/second/third/another_file.txt", "first/second/third/file.txt", "first/second/yet_another_file.txt", "foo", "foo/bar.txt", "prefix", "prefix/sub", "prefix/sub/file.txt"},
			nil,
		),
		Entry("invokes fn only for paths under first and foo when prefix is 'f'",
			"f", "",
			[]string{"file_at_root.txt", "first", "first/first_other", "first/first_other/y.txt", "first/second", "first/second/second_extra", "first/second/second_extra/x.txt", "first/second/third", "first/second/third/another_file.txt", "first/second/third/file.txt", "first/second/yet_another_file.txt", "foo", "foo/bar.txt"},
			nil,
		),
		Entry("invokes fn only for paths under first/second when prefix is 'first/second'",
			"first/second", "",
			[]string{"first/second", "first/second/second_extra", "first/second/second_extra/x.txt", "first/second/third", "first/second/third/another_file.txt", "first/second/third/file.txt", "first/second/yet_another_file.txt"},
			nil,
		),
		Entry("invokes fn only for startFrom and paths after it in walk order when startFrom is set",
			"first", "first/second/third/another_file.txt",
			[]string{"first/second/third/another_file.txt", "first/second/third/file.txt", "first/second/yet_another_file.txt"},
			nil,
		),
		Entry("returns ErrStartFromNotExist when startFrom path does not exist",
			"", "nonexistent", nil, smartio.ErrStartFromNotExist,
		),
		Entry("returns ErrStartFromBadPrefix when startFrom path does not match prefix",
			"first", "baz/que.txt", nil, smartio.ErrStartFromBadPrefix,
		),
		Entry("returns context error when context is cancelled",
			"", "", nil, context.Canceled,
		),
		Entry("startFrom directory includes that directory and all descendants",
			"first", "first/second",
			[]string{"first/second", "first/second/second_extra", "first/second/second_extra/x.txt", "first/second/third", "first/second/third/another_file.txt", "first/second/third/file.txt", "first/second/yet_another_file.txt"},
			nil,
		),
		Entry("startFrom last path in full walk returns only that file",
			"", "prefix/sub/file.txt",
			[]string{"prefix/sub/file.txt"},
			nil,
		),
		Entry("startFrom root with empty prefix returns full walk",
			"", "__root__",
			[]string{"", "a", "a/b", "a/b/c", "a/b/c/deep.txt", "baz", "baz/que.txt", "empty_dir", "file_at_root.txt", "first", "first/first_other", "first/first_other/y.txt", "first/second", "first/second/second_extra", "first/second/second_extra/x.txt", "first/second/third", "first/second/third/another_file.txt", "first/second/third/file.txt", "first/second/yet_another_file.txt", "foo", "foo/bar.txt", "prefix", "prefix/sub", "prefix/sub/file.txt"},
			nil,
		),
		Entry("startFrom deep file with prefix 'a' returns only that file and nothing after in subtree",
			"a", "a/b/c/deep.txt",
			[]string{"a/b/c/deep.txt"},
			nil,
		),
		Entry("prefix 'a' returns only a and below",
			"a", "",
			[]string{"a", "a/b", "a/b/c", "a/b/c/deep.txt"},
			nil,
		),
		Entry("prefix 'a/b/c' returns only a/b/c and deep.txt",
			"a/b/c", "",
			[]string{"a/b/c", "a/b/c/deep.txt"},
			nil,
		),
		Entry("prefix 'z' matches nothing and returns empty slice",
			"z", "",
			[]string{},
			nil,
		),
		Entry("prefix 'prefix' visits directory named prefix and its subtree",
			"prefix", "",
			[]string{"prefix", "prefix/sub", "prefix/sub/file.txt"},
			nil,
		),
		Entry("prefix matching a single file visits only that file",
			"file_at_root", "",
			[]string{"file_at_root.txt"},
			nil,
		),
		Entry("returns ErrStartFromOutsideRoot when startFrom is not under root",
			"", "__outside__",
			nil, smartio.ErrStartFromOutsideRoot,
		),
	)

	Context("edge cases", func() {
		It("propagates error from callback and stops walk", func(ctx context.Context) {
			var callCount int

			err := smartio.WalkDir(ctx, root, "", nil, func(path string) error {
				callCount++

				if filepath.Base(path) == "third" {
					return errStopWalk
				}

				return nil
			})
			Expect(err).To(Equal(errStopWalk))
			Expect(callCount).To(BeNumerically("<", 25))
		})

		It("invokes fn for root when root is the only entry and prefix is empty", func(ctx context.Context) {
			emptyRoot := lo.Must(os.MkdirTemp("", "walkdir-single-*"))

			DeferCleanup(func() { _ = os.RemoveAll(emptyRoot) })

			var got []string

			err := smartio.WalkDir(ctx, emptyRoot, "", nil, func(path string) error {
				got = append(got, path)

				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal([]string{emptyRoot}))
		})

		It("prefix with trailing slash is trimmed and matches same as without", func(ctx context.Context) {
			var got []string

			err := smartio.WalkDir(ctx, root, "first/", nil, func(path string) error {
				got = append(got, path)

				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(ContainElement(filepath.Join(root, "first")))
			Expect(got).To(ContainElement(filepath.Join(root, "first", "second", "third", "file.txt")))
			Expect(got).NotTo(ContainElement(filepath.Join(root, "foo")))
		})
	})
})
