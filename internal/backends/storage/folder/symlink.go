//go:build unix

package folder

import (
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"

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

// openDirNoFollow opens a directory without following symlinks in any path component.
// It walks the path component-by-component using openat with O_NOFOLLOW to avoid TOCTOU races.
func openDirNoFollow(path string) (*os.File, error) { //nolint:funlen
	path = filepath.Clean(path)
	if path == "" || path == "." {
		fd, err := unix.Openat(unix.AT_FDCWD, ".", unix.O_RDONLY|unix.O_NOFOLLOW, 0)
		if err != nil {
			return nil, err
		}

		return os.NewFile(uintptr(fd), "."), nil //nolint:gosec // G115: fd from unix.Openat is valid
	}

	dirFd := unix.AT_FDCWD
	if filepath.IsAbs(path) {
		fd, err := unix.Openat(unix.AT_FDCWD, "/", unix.O_RDONLY|unix.O_NOFOLLOW, 0)
		if err != nil {
			return nil, err
		}

		dirFd = fd
	}

	components := splitPath(path)

	for i, name := range components {
		if name == "" || name == "." {
			continue
		}

		if name == ".." {
			return nil, core.ErrSymlinkNotAllowed
		}

		isLast := i == len(components)-1
		if isLast {
			fd, err := unix.Openat(dirFd, name, unix.O_RDONLY|unix.O_NOFOLLOW, 0)
			if err != nil {
				return nil, err
			}

			if dirFd != unix.AT_FDCWD {
				_ = unix.Close(dirFd)
			}

			return os.NewFile(uintptr(fd), path), nil //nolint:gosec // G115: fd from unix.Openat is valid
		}

		nextFd, err := unix.Openat(dirFd, name, unix.O_RDONLY|unix.O_NOFOLLOW, 0)
		if err != nil {
			if dirFd != unix.AT_FDCWD {
				_ = unix.Close(dirFd)
			}

			return nil, err
		}

		if dirFd != unix.AT_FDCWD {
			_ = unix.Close(dirFd)
		}

		dirFd = nextFd
	}

	if dirFd != unix.AT_FDCWD {
		_ = unix.Close(dirFd)
	}

	fd, err := unix.Openat(unix.AT_FDCWD, ".", unix.O_RDONLY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}

	return os.NewFile(uintptr(fd), path), nil //nolint:gosec // G115: fd from unix.Openat is valid
}

func splitPath(path string) []string {
	var parts []string

	for p := path; p != ""; p = filepath.Dir(p) {
		parent := filepath.Dir(p)
		if p == parent {
			break
		}

		parts = append([]string{filepath.Base(p)}, parts...)
	}

	return parts
}

// createFileNoFollow creates a new file at path without following symlinks (avoids TOCTOU).
func createFileNoFollow(path string, mode uint32) (*os.File, error) {
	path = filepath.Clean(path)
	parentDir := filepath.Dir(path)
	base := filepath.Base(path)

	parent, err := openDirNoFollow(parentDir)
	if err != nil {
		return nil, err
	}
	defer parent.Close()

	flags := unix.O_CREAT | unix.O_EXCL | unix.O_WRONLY | unix.O_NOFOLLOW

	fd, err := unix.Openat(int(parent.Fd()), base, flags, mode) //nolint:gosec // G115: fd valid
	if err != nil {
		return nil, err
	}

	return os.NewFile(uintptr(fd), path), nil //nolint:gosec // G115: fd from unix.Openat is valid
}

// mkdirAllNoFollow creates a directory and all parents without following symlinks (avoids TOCTOU).
func mkdirAllNoFollow(path string, mode os.FileMode) error { //nolint:unparam // mode kept for future flexibility
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return nil
	}

	components := splitPath(path)

	dirFd := unix.AT_FDCWD
	if filepath.IsAbs(path) {
		fd, err := unix.Openat(unix.AT_FDCWD, "/", unix.O_RDONLY|unix.O_NOFOLLOW, 0)
		if err != nil {
			return err
		}

		dirFd = fd
	}

	defer func() {
		if dirFd != unix.AT_FDCWD {
			_ = unix.Close(dirFd)
		}
	}()

	for i, name := range components {
		if name == "" || name == "." {
			continue
		}

		if name == ".." {
			return core.ErrSymlinkNotAllowed
		}

		err := unix.Mkdirat(dirFd, name, uint32(mode))
		if err != nil && !os.IsExist(err) {
			return err
		}

		if i < len(components)-1 {
			nextFd, err := unix.Openat(dirFd, name, unix.O_RDONLY|unix.O_NOFOLLOW, 0)
			if err != nil {
				return err
			}

			if dirFd != unix.AT_FDCWD {
				_ = unix.Close(dirFd)
			}

			dirFd = nextFd
		}
	}

	return nil
}

// renameNoFollow renames oldPath to newPath without following symlinks (avoids TOCTOU).
func renameNoFollow(oldPath, newPath string) error {
	oldPath = filepath.Clean(oldPath)
	newPath = filepath.Clean(newPath)
	oldParent := filepath.Dir(oldPath)
	newParent := filepath.Dir(newPath)
	oldBase := filepath.Base(oldPath)
	newBase := filepath.Base(newPath)

	oldDir, err := openDirNoFollow(oldParent)
	if err != nil {
		return err
	}
	defer oldDir.Close()

	newDir, err := openDirNoFollow(newParent)
	if err != nil {
		return err
	}
	defer newDir.Close()

	oldFd := int(oldDir.Fd()) //nolint:gosec // G115: fd valid
	newFd := int(newDir.Fd()) //nolint:gosec // G115: fd valid

	return unix.Renameat(oldFd, oldBase, newFd, newBase)
}
