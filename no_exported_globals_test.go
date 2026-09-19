package xerr_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// TestNoExportedMutableGlobals is a structural lock, not a behavioral
// one: it fails if the xerr package ever gains an exported
// package-level var whose type is a map or a slice.
//
// v2 shipped CodesKind/CodesHttpStatus as exported map vars, which let
// any caller bypass RegisterCode's validation and locking by writing
// directly into the registry (`xerr.CodesKind[x] = y`). v3 closed that
// backdoor by unexporting both maps and making RegisterCode the only
// way in. This test is the guardrail that keeps it closed: it parses
// every non-test .go file in this package and fails on sight of a new
// exported map or slice var, so the backdoor can't be silently
// reopened by a future change.
//
// A slice is flagged for the same reason a map is: like a map, a slice
// value shares its backing array, so `var Foo = []T{...}` lets an
// outside caller mutate shared package state through the exported
// variable, same as a map would.
func TestNoExportedMutableGlobals(t *testing.T) {
	fset := token.NewFileSet()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("ReadDir(.) error = %v", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatalf("ParseFile(%s) error = %v", name, err)
		}

		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || genDecl.Tok != token.VAR {
				continue
			}
			for _, spec := range genDecl.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, ident := range valueSpec.Names {
					if !ident.IsExported() {
						continue
					}
					if isMutableGlobalType(valueSpec, i) {
						t.Errorf("%s: exported package-level var %q has a map/slice type — "+
							"this reopens the direct-registry-mutation backdoor RegisterCode "+
							"closed; unexport it or route writes through a validated function instead",
							name, ident.Name)
					}
				}
			}
		}
	}
}

// isMutableGlobalType reports whether the i-th name in spec is declared
// (explicitly, or via its initializer's composite literal) as a map or
// slice type.
func isMutableGlobalType(spec *ast.ValueSpec, i int) bool {
	if spec.Type != nil {
		return isMapOrSliceType(spec.Type)
	}
	if i < len(spec.Values) {
		if lit, ok := spec.Values[i].(*ast.CompositeLit); ok && lit.Type != nil {
			return isMapOrSliceType(lit.Type)
		}
	}
	return false
}

func isMapOrSliceType(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.MapType:
		return true
	case *ast.ArrayType:
		return t.Len == nil // nil Len means slice, not fixed-size array
	default:
		return false
	}
}
