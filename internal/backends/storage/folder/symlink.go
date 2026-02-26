//go:build unix

package folder

import (
	"os"
	"path/filepath"
	"syscall"

	"github.com/zhulik/d3/internal/core"
)

// rejectSymlink returns an error if path exists and is a symlink.
func rejectSymlink(path string) error {
	fi, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return err
	}

	if fi.Mode()&os.ModeSymlink != 0 {
		return core.ErrSymlinkNotAllowed
	}

	return nil
}

// rejectSymlinkInPath walks path components and returns an error if any existing component is a symlink.
func rejectSymlinkInPath(path string) error {
	path = filepath.Clean(path)

	for p := path; ; p = filepath.Dir(p) {
		fi, err := os.Lstat(p)
		if err != nil {
			if os.IsNotExist(err) {
				break
			}

			return err
		}

		if fi.Mode()&os.ModeSymlink != 0 {
			return core.ErrSymlinkNotAllowed
		}

		if p == filepath.Dir(p) {
			break
		}
	}

	return nil
}

// openFileNoFollow opens a file for reading without following symlinks.
func openFileNoFollow(path string) (*os.File, error) {
	fd, err := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}

	return os.NewFile(uintptr(fd), path), nil //nolint:gosec // G115: fd from syscall.Open is valid for this process
}
