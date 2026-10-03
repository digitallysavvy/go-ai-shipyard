package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const maxFileBytes = 256 << 10

// workspace is the project the agents work on: a copy of workspace-template
// under .data/workspace that the chat tools read and test, and that the
// coding agent's changes are synced back into.
type workspace struct {
	template string
	dir      string
	mu       sync.Mutex // serializes test runs, syncs and resets
}

func newWorkspace(template, dir string) (*workspace, error) {
	if _, err := os.Stat(template); err != nil {
		return nil, err
	}
	return &workspace{template: template, dir: dir}, nil
}

// Reset restores the workspace to the template (the failing state).
func (w *workspace) Reset() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := os.RemoveAll(w.dir); err != nil {
		return err
	}
	files, err := readTree(w.template)
	if err != nil {
		return err
	}
	return writeTree(w.dir, files)
}

// Files returns every file in the workspace keyed by slash-separated path.
func (w *workspace) Files() (map[string]string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return readTree(w.dir)
}

// ReadFile returns one file, rejecting paths that escape the workspace.
func (w *workspace) ReadFile(rel string) (string, error) {
	p, err := w.resolve(rel)
	if err != nil {
		return "", err
	}
	b, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("%s does not exist", rel)
	}
	if err != nil {
		return "", err
	}
	if len(b) > maxFileBytes {
		return "", fmt.Errorf("%s is larger than %d bytes", rel, maxFileBytes)
	}
	return string(b), nil
}

// Apply writes changed files back into the workspace.
func (w *workspace) Apply(changed map[string]string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	for rel, content := range changed {
		p, err := w.resolve(rel)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

type testRun struct {
	Passed     bool   `json:"passed"`
	Output     string `json:"output"`
	DurationMs int64  `json:"durationMs"`
}

// RunTests runs `go test ./...` in the workspace.
func (w *workspace) RunTests(ctx context.Context) testRun {
	w.mu.Lock()
	defer w.mu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	start := time.Now()
	cmd := exec.CommandContext(ctx, "go", "test", "-count=1", "./...")
	cmd.Dir = w.dir
	out, err := cmd.CombinedOutput()
	return testRun{
		Passed:     err == nil,
		Output:     strings.TrimSpace(string(out)),
		DurationMs: time.Since(start).Milliseconds(),
	}
}

func (w *workspace) resolve(rel string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(rel))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q is outside the workspace", rel)
	}
	return filepath.Join(w.dir, clean), nil
}

func readTree(dir string) (map[string]string, error) {
	files := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() > maxFileBytes {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		files[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	return files, err
}

func writeTree(dir string, files map[string]string) error {
	for rel, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
