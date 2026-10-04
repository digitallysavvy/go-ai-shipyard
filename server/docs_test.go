package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The docs are a map of the code: AGENTS.md and docs/*.md name files and
// symbols so agents can jump straight to them. This test keeps that map true.

var (
	backtickRe = regexp.MustCompile("`([^`\\s]+)`")
	mdLinkRe   = regexp.MustCompile(`\]\(([^)#\s]+)(#[^)]*)?\)`)
	// A repo path: has a slash or a known root file, and a file extension.
	repoPathRe = regexp.MustCompile(`^(\.?[A-Za-z0-9_\-]+/)*[A-Za-z0-9_\-.]+\.(go|ts|tsx|css|md|mod|yml|json|mjs|example)$`)
	identRe    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

func docFiles(t *testing.T, root string) []string {
	t.Helper()
	files := []string{filepath.Join(root, "AGENTS.md")}
	matches, err := filepath.Glob(filepath.Join(root, "docs", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	return append(files, matches...)
}

func TestDocsReferToRealFilesAndSymbols(t *testing.T) {
	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}
	for _, doc := range docFiles(t, root) {
		raw, err := os.ReadFile(doc)
		if err != nil {
			t.Fatal(err)
		}
		rel, _ := filepath.Rel(root, doc)
		text := stripCodeFences(string(raw))

		// Relative markdown links must resolve.
		for _, m := range mdLinkRe.FindAllStringSubmatch(text, -1) {
			target := m[1]
			if strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(doc), target)); err != nil {
				t.Errorf("%s: link to %s does not resolve", rel, target)
			}
		}

		// Backticked repo paths must exist.
		for _, m := range backtickRe.FindAllStringSubmatch(text, -1) {
			p := strings.TrimSuffix(m[1], "/")
			if !repoPathRe.MatchString(p) || !strings.Contains(p, "/") {
				continue
			}
			if _, err := os.Stat(filepath.Join(root, p)); err != nil {
				t.Errorf("%s: %s does not exist", rel, p)
			}
		}

		// Table rows of the form | `path/to/file` | `Symbol`, `Other` | ...
		for _, line := range strings.Split(text, "\n") {
			cells := strings.Split(line, "|")
			if len(cells) < 4 {
				continue
			}
			first := backtickRe.FindStringSubmatch(cells[1])
			if first == nil || !repoPathRe.MatchString(first[1]) {
				continue
			}
			file := filepath.Join(root, first[1])
			var symbols []string
			for _, m := range backtickRe.FindAllStringSubmatch(cells[2], -1) {
				if identRe.MatchString(m[1]) {
					symbols = append(symbols, m[1])
				}
			}
			for _, sym := range symbols {
				if !declares(t, file, sym) {
					t.Errorf("%s: %s does not declare %s", rel, first[1], sym)
				}
			}
		}
	}
}

// stripCodeFences drops fenced code blocks, whose contents are examples.
func stripCodeFences(s string) string {
	var out strings.Builder
	inFence := false
	for _, line := range strings.SplitAfter(s, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if !inFence {
			out.WriteString(line)
		}
	}
	return out.String()
}

// declares reports whether file declares sym at top level (Go) or as a
// function, constant, type or interface (TypeScript). For CSS files it
// accepts any name, since the table lists token groups rather than symbols.
func declares(t *testing.T, file, sym string) bool {
	t.Helper()
	switch filepath.Ext(file) {
	case ".go":
		// A test file's row names the code it covers, so look across the
		// whole package for _test.go files and in the file itself otherwise.
		files := []string{file}
		if strings.HasSuffix(file, "_test.go") {
			files, _ = filepath.Glob(filepath.Join(filepath.Dir(file), "*.go"))
		}
		for _, name := range files {
			if goFileDeclares(t, name, sym) {
				return true
			}
		}
		return false
	case ".ts", ".tsx":
		src, err := os.ReadFile(file)
		if err != nil {
			return false
		}
		return regexp.MustCompile(`\b(function|const|let|type|interface|class)\s+` + regexp.QuoteMeta(sym) + `\b`).Match(src)
	default:
		return true
	}
}

// goFileDeclares reports whether the Go file declares sym at top level,
// including methods.
func goFileDeclares(t *testing.T, file, sym string) bool {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name.Name == sym {
				return true
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.Name == sym {
						return true
					}
				case *ast.ValueSpec:
					for _, n := range s.Names {
						if n.Name == sym {
							return true
						}
					}
				}
			}
		}
	}
	return false
}
