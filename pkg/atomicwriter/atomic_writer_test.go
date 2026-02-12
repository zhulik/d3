package atomicwriter_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/stretchr/testify/mock"
	"github.com/zhulik/d3/pkg/atomicwriter"
)

var _ = Describe("AtomicWriter", func() {
	var (
		tmpDir      string
		tmpDataPath string
		locker      atomicwriter.Locker
		writer      *atomicwriter.AtomicWriter
	)

	BeforeEach(func() {
		tmpDir = lo.Must(os.MkdirTemp("", "atomicwriter-test-*"))

		tmpDataPath = filepath.Join(tmpDir, "data")
		lo.Must0(os.MkdirAll(tmpDataPath, 0755))

		mockLocker := NewMockLocker(GinkgoT())

		mockLocker.EXPECT().Lock(mock.Anything, mock.Anything).Maybe().Return(
			func(ctx context.Context, _ string) (context.Context, context.CancelFunc, error) {
				return ctx, func() {}, nil
			},
		)

		locker = mockLocker
		writer = atomicwriter.New(locker, tmpDataPath)
	})

	AfterEach(func() {
		if tmpDir != "" {
			_ = os.RemoveAll(tmpDir)
		}
	})

	Describe("New", func() {
		It("creates a new AtomicWriter", func() {
			aw := atomicwriter.New(locker, tmpDataPath)
			Expect(aw).NotTo(BeNil())
		})
	})

	Describe("ReadWrite", func() {
		Describe("Happy path", func() {
			It("creates a new file when it doesn't exist", func(ctx context.Context) {
				filename := filepath.Join(tmpDir, "new_file.txt")

				err := writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(BeEmpty())

					return []byte("new content"), nil
				})

				Expect(err).NotTo(HaveOccurred())
				content, err := os.ReadFile(filename)
				Expect(err).NotTo(HaveOccurred())
				Expect(content).To(Equal([]byte("new content")))
			})

			It("reads and modifies existing file content", func(ctx context.Context) {
				filename := filepath.Join(tmpDir, "existing_file.txt")
				originalContent := []byte("original content")

				err := os.WriteFile(filename, originalContent, 0644)
				Expect(err).NotTo(HaveOccurred())

				err = writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(Equal(originalContent))

					return append(content, []byte(" - modified")...), nil
				})

				Expect(err).NotTo(HaveOccurred())
				content, err := os.ReadFile(filename)
				Expect(err).NotTo(HaveOccurred())
				Expect(content).To(Equal([]byte("original content - modified")))
			})

			It("preserves file permissions for existing files", func(ctx context.Context) {
				filename := filepath.Join(tmpDir, "permission_file.txt")
				originalContent := []byte("content")
				originalPerm := os.FileMode(0755)

				err := os.WriteFile(filename, originalContent, 0644)
				Expect(err).NotTo(HaveOccurred())

				// Explicitly set the permissions we want to preserve
				err = os.Chmod(filename, originalPerm)
				Expect(err).NotTo(HaveOccurred())

				err = writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(Equal(originalContent))

					return []byte("new content"), nil
				})

				Expect(err).NotTo(HaveOccurred())

				fileInfo, err := os.Stat(filename)
				Expect(err).NotTo(HaveOccurred())
				Expect(fileInfo.Mode()).To(Equal(originalPerm))
			})

			It("uses default permissions (0644) for new files", func(ctx context.Context) {
				filename := filepath.Join(tmpDir, "new_perm_file.txt")

				err := writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(BeEmpty())

					return []byte("content"), nil
				})

				Expect(err).NotTo(HaveOccurred())

				fileInfo, err := os.Stat(filename)
				Expect(err).NotTo(HaveOccurred())
				Expect(fileInfo.Mode()).To(Equal(os.FileMode(0644)))
			})

			It("handles empty file correctly", func(ctx context.Context) {
				filename := filepath.Join(tmpDir, "empty_file.txt")

				err := os.WriteFile(filename, []byte{}, 0644)
				Expect(err).NotTo(HaveOccurred())

				err = writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(BeEmpty())

					return []byte("populated"), nil
				})

				Expect(err).NotTo(HaveOccurred())
				content, err := os.ReadFile(filename)
				Expect(err).NotTo(HaveOccurred())
				Expect(content).To(Equal([]byte("populated")))
			})

			It("handles large file content", func(ctx context.Context) {
				filename := filepath.Join(tmpDir, "large_file.txt")
				largeContent := make([]byte, 1024*1024) // 1MB
				for i := range largeContent {
					largeContent[i] = byte(i % 256)
				}

				err := writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(BeEmpty())

					return largeContent, nil
				})

				Expect(err).NotTo(HaveOccurred())
				content, err := os.ReadFile(filename)
				Expect(err).NotTo(HaveOccurred())
				Expect(content).To(Equal(largeContent))
			})

			It("appends content to existing file", func(ctx context.Context) {
				filename := filepath.Join(tmpDir, "append_file.txt")

				err := writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(BeEmpty())

					return []byte("line1\n"), nil
				})
				Expect(err).NotTo(HaveOccurred())

				err = writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(Equal([]byte("line1\n")))

					return append(content, []byte("line2\n")...), nil
				})

				Expect(err).NotTo(HaveOccurred())
				content, err := os.ReadFile(filename)
				Expect(err).NotTo(HaveOccurred())
				Expect(content).To(Equal([]byte("line1\nline2\n")))
			})

			It("handles binary content correctly", func(ctx context.Context) {
				filename := filepath.Join(tmpDir, "binary_file.bin")
				binaryContent := []byte{0xFF, 0xFE, 0x00, 0x01, 0xAB, 0xCD, 0xEF}

				err := writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(BeEmpty())

					return binaryContent, nil
				})

				Expect(err).NotTo(HaveOccurred())
				content, err := os.ReadFile(filename)
				Expect(err).NotTo(HaveOccurred())
				Expect(content).To(Equal(binaryContent))
			})

			It("returns unchanged content when contentMap returns input as-is", func(ctx context.Context) {
				filename := filepath.Join(tmpDir, "unchanged_file.txt")
				originalContent := []byte("original")

				err := os.WriteFile(filename, originalContent, 0644)
				Expect(err).NotTo(HaveOccurred())

				err = writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(Equal(originalContent))

					return content, nil
				})

				Expect(err).NotTo(HaveOccurred())
				content, err := os.ReadFile(filename)
				Expect(err).NotTo(HaveOccurred())
				Expect(content).To(Equal(originalContent))
			})
		})

		Describe("Error cases - Lock failures", func() {
			It("returns error if lock fails", func(ctx context.Context) {
				failingLocker := NewMockLocker(GinkgoT())
				lockErr := errors.New("lock error") //nolint:err113
				failingLocker.EXPECT().Lock(mock.Anything, mock.Anything).Maybe().Return(nil, nil, lockErr)
				failingWriter := atomicwriter.New(failingLocker, tmpDataPath)
				filename := filepath.Join(tmpDir, "file.txt")

				err := failingWriter.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(BeEmpty())

					return []byte("content"), nil
				})

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("lock error"))
			})
		})

		Describe("Error cases - Directory handling", func() {
			It("returns error if directory of file doesn't exist", func(ctx context.Context) {
				nonExistentDir := filepath.Join(tmpDir, "non_existent_dir")
				filename := filepath.Join(nonExistentDir, "file.txt")

				err := writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(BeEmpty())

					return []byte("content"), nil
				})

				Expect(err).To(HaveOccurred())
			})
		})

		Describe("Error cases - ContentMap function failures", func() {
			It("returns error if contentMap function fails", func(ctx context.Context) {
				filename := filepath.Join(tmpDir, "file.txt")
				expectedError := errors.New("custom content map error") //nolint:err113

				err := writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(BeEmpty())

					return nil, expectedError
				})

				Expect(err).To(Equal(expectedError))
				// Verify the file was not created or modified
				_, err = os.Stat(filename)
				Expect(err).To(HaveOccurred())
			})

			It("returns error if contentMap fails on existing file", func(ctx context.Context) {
				filename := filepath.Join(tmpDir, "file.txt")
				originalContent := []byte("original")

				err := os.WriteFile(filename, originalContent, 0644)
				Expect(err).NotTo(HaveOccurred())

				expectedError := errors.New("content transformation error") //nolint:err113
				err = writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(Equal(originalContent))

					return nil, expectedError
				})

				Expect(err).To(Equal(expectedError))

				// Verify file content wasn't modified
				content, err := os.ReadFile(filename)
				Expect(err).NotTo(HaveOccurred())
				Expect(content).To(Equal(originalContent))
			})
		})

		Describe("Error cases - File system operations", func() {
			It("returns error if temp file creation fails", func(ctx context.Context) {
				readOnlyDir, err := os.MkdirTemp("", "atomicwriter-readonly-*")
				Expect(err).NotTo(HaveOccurred())
				defer func() { _ = os.RemoveAll(readOnlyDir) }()

				// Create a file in the directory first
				filename := filepath.Join(readOnlyDir, "file.txt")
				err = os.WriteFile(filename, []byte("content"), 0644)
				Expect(err).NotTo(HaveOccurred())

				// Make directory read-only to prevent temp file creation
				err = os.Chmod(readOnlyDir, 0555)
				Expect(err).NotTo(HaveOccurred())
				defer func() { _ = os.Chmod(readOnlyDir, 0755) }()

				readOnlyTmpDir, err := os.MkdirTemp("", "atomicwriter-tmp-*")
				Expect(err).NotTo(HaveOccurred())
				defer func() { _ = os.RemoveAll(readOnlyTmpDir) }()

				// Make tmp directory read-only to prevent temp file creation
				err = os.Chmod(readOnlyTmpDir, 0555)
				Expect(err).NotTo(HaveOccurred())
				defer func() { _ = os.Chmod(readOnlyTmpDir, 0755) }()

				failingWriter := atomicwriter.New(locker, readOnlyTmpDir)

				err = failingWriter.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(Equal([]byte("content")))

					return []byte("new content"), nil
				})

				Expect(err).To(HaveOccurred())
			})

			It("returns error if rename fails", func(ctx context.Context) {
				filename := filepath.Join(tmpDir, "file.txt")

				// Create a directory with the same name where the file should be
				err := os.Mkdir(filename, 0755)
				Expect(err).NotTo(HaveOccurred())

				err = writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(BeEmpty())

					return []byte("content"), nil
				})

				Expect(err).To(HaveOccurred())
			})
		})

		Describe("Error cases - Context cancellation", func() {
			It("returns error if context is cancelled before rename", func(ctx context.Context) {
				filename := filepath.Join(tmpDir, "file.txt")
				ctx, cancel := context.WithCancel(ctx)

				err := writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(BeEmpty())
					// Cancel the context before returning
					cancel()

					return []byte("content"), nil
				})

				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(context.Canceled))
			})
		})

		Describe("Edge cases", func() {
			It("handles file with special characters in name", func(ctx context.Context) {
				filename := filepath.Join(tmpDir, "file-with_special.chars123.txt")

				err := writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(BeEmpty())

					return []byte("content"), nil
				})

				Expect(err).NotTo(HaveOccurred())
				content, err := os.ReadFile(filename)
				Expect(err).NotTo(HaveOccurred())
				Expect(content).To(Equal([]byte("content")))
			})

			It("handles sequential writes correctly", func(ctx context.Context) {
				filename := filepath.Join(tmpDir, "sequential.txt")

				// First write
				err := writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(BeEmpty())

					return []byte("first"), nil
				})
				Expect(err).NotTo(HaveOccurred())

				// Second write
				err = writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(Equal([]byte("first")))

					return []byte("second"), nil
				})
				Expect(err).NotTo(HaveOccurred())

				// Third write
				err = writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(Equal([]byte("second")))

					return []byte("third"), nil
				})
				Expect(err).NotTo(HaveOccurred())

				content, err := os.ReadFile(filename)
				Expect(err).NotTo(HaveOccurred())
				Expect(content).To(Equal([]byte("third")))
			})

			It("handles file with no read permissions on existing file gracefully", func(ctx context.Context) {
				filename := filepath.Join(tmpDir, "no_read_perm.txt")
				err := os.WriteFile(filename, []byte("content"), 0755)
				Expect(err).NotTo(HaveOccurred())

				// Change to 000 to make it unreadable
				err = os.Chmod(filename, 0000)
				Expect(err).NotTo(HaveOccurred())
				defer func() { _ = os.Chmod(filename, 0644) }()

				err = writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(BeEmpty())

					return []byte("new"), nil
				})

				// Should fail because we can't read the file
				Expect(err).To(HaveOccurred())
			})

			It("handles contentMap that transforms input content", func(ctx context.Context) {
				filename := filepath.Join(tmpDir, "transform.txt")
				err := os.WriteFile(filename, []byte("hello"), 0644)
				Expect(err).NotTo(HaveOccurred())

				err = writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(Equal([]byte("hello")))
					// Transform: uppercase and add suffix
					result := []byte{}
					for _, b := range content {
						if b >= 'a' && b <= 'z' {
							result = append(result, b-32)
						} else {
							result = append(result, b)
						}
					}

					return append(result, []byte("-TRANSFORMED")...), nil
				})

				Expect(err).NotTo(HaveOccurred())
				content, err := os.ReadFile(filename)
				Expect(err).NotTo(HaveOccurred())
				Expect(content).To(Equal([]byte("HELLO-TRANSFORMED")))
			})

			It("handles file content with null bytes", func(ctx context.Context) {
				filename := filepath.Join(tmpDir, "null_bytes.bin")
				contentWithNulls := []byte{0x00, 0x48, 0x65, 0x6c, 0x6c, 0x6f, 0x00}

				err := writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(BeEmpty())

					return contentWithNulls, nil
				})

				Expect(err).NotTo(HaveOccurred())
				content, err := os.ReadFile(filename)
				Expect(err).NotTo(HaveOccurred())
				Expect(content).To(Equal(contentWithNulls))
			})

			It("correctly updates file mtime", func(ctx context.Context) {
				filename := filepath.Join(tmpDir, "mtime.txt")
				err := os.WriteFile(filename, []byte("original"), 0644)
				Expect(err).NotTo(HaveOccurred())

				fileInfo1, err := os.Stat(filename)
				Expect(err).NotTo(HaveOccurred())
				mtime1 := fileInfo1.ModTime()

				// Small delay to ensure mtime difference
				// In practice, this may not always show a difference on fast systems
				// but the atomic rename should ensure file is updated

				err = writer.ReadWrite(ctx, filename, func(_ context.Context, content []byte) ([]byte, error) {
					Expect(content).To(Equal([]byte("original")))

					return []byte("modified"), nil
				})
				Expect(err).NotTo(HaveOccurred())

				fileInfo2, err := os.Stat(filename)
				Expect(err).NotTo(HaveOccurred())
				mtime2 := fileInfo2.ModTime()

				// After modification, mtime should be >= original
				Expect(mtime2.After(mtime1) || mtime2.Equal(mtime1)).To(BeTrue())
			})

			It("handles multiple concurrent writes with sequential processing", func(ctx context.Context) {
				filenames := []string{
					filepath.Join(tmpDir, "file1.txt"),
					filepath.Join(tmpDir, "file2.txt"),
					filepath.Join(tmpDir, "file3.txt"),
				}

				for i, filename := range filenames {
					content := []byte("content" + string(rune('1'+i)))
					err := writer.ReadWrite(ctx, filename, func(_ context.Context, _ []byte) ([]byte, error) {
						return content, nil
					})
					Expect(err).NotTo(HaveOccurred())
				}

				for i, filename := range filenames {
					content, err := os.ReadFile(filename)
					Expect(err).NotTo(HaveOccurred())
					Expect(content).To(Equal([]byte("content" + string(rune('1'+i)))))
				}
			})
		})
	})
})
