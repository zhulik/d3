package smartio

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Sentinel errors for WalkDir.
var (
	ErrStartFromNotExist    = errors.New("startFrom path does not exist")
	ErrStartFromBadPrefix   = errors.New("startFrom path does not match prefix")
	ErrStartFromOutsideRoot = errors.New("startFrom path is not under root")
)

// WalkDir walks the file tree rooted at root, calling fn for each file or directory.
// It is similar to filepath.WalkDir but supports prefix filtering and an optional startFrom path.
//
// prefix filters which paths are visited: only paths whose relative path from root
// has prefix as a prefix are visited. Empty prefix means no filtering.
// Prefix is interpreted with forward slashes (e.g. "first/second").
//
// If startFrom is non-nil, iteration starts from that path: all entries before it in
// depth-first order are skipped. startFrom must exist and must match the prefix.
// Walk order is depth-first: directory first, then its children in lexical order.
func WalkDir(
	ctx context.Context,
	root string,
	prefix string,
	startFrom *string,
	fn func(path string) error,
) error {
	root = filepath.Clean(root)
	prefix = filepath.ToSlash(strings.Trim(prefix, "/"))

	if startFrom != nil {
		if err := validateStartFrom(root, prefix, *startFrom); err != nil {
			return err
		}
	}

	started := startFrom == nil

	return walkDir(ctx, root, "", prefix, startFrom, &started, fn)
}

func validateStartFrom(root, prefix, startFrom string) error {
	path := filepath.Clean(startFrom)

	_, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ErrStartFromNotExist
		}

		return err
	}

	rel, err := filepath.Rel(root, path)
	if err != nil {
		return err
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ErrStartFromOutsideRoot
	}

	rel = filepath.ToSlash(rel)
	if prefix != "" && !strings.HasPrefix(rel, prefix) {
		return ErrStartFromBadPrefix
	}

	return nil
}

func walkDir(
	ctx context.Context,
	path string,
	relPath string,
	prefix string,
	startFrom *string,
	started *bool,
	fn func(path string) error,
) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if startFrom != nil && !*started {
		cleanStart := filepath.Clean(*startFrom)
		if path == cleanStart {
			*started = true
		}
	}

	if *started && strings.HasPrefix(relPath, prefix) {
		if err := fn(path); err != nil {
			return err
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, e := range entries {
		name := e.Name()
		childPath := filepath.Join(path, name)
		childRel := filepath.ToSlash(filepath.Join(relPath, name))

		if err := walkDir(ctx, childPath, childRel, prefix, startFrom, started, fn); err != nil {
			return err
		}
	}

	return nil
}
