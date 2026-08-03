package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestEveryRouteHasHandler fails when the generated ServerInterface declares an
// operation that no hand-written handler implements.
//
// Handlers embeds Unimplemented so that regenerating the spec never breaks the
// build. The cost is that a missing or renamed handler is invisible to the
// compiler: the route silently answers 501 at runtime, and an old handler with
// the previous name still compiles as dead code. This test restores the signal.
//
// It reads the source rather than using reflection because a promoted method
// and a real one are indistinguishable at runtime — both report the same
// (*Handlers).Name wrapper.
func TestEveryRouteHasHandler(t *testing.T) {
	implemented := handlerMethodsInPackage(t)

	iface := reflect.TypeOf((*ServerInterface)(nil)).Elem()
	for i := range iface.NumMethod() {
		name := iface.Method(i).Name
		if !implemented[name] {
			t.Errorf("ServerInterface.%s has no handler on *Handlers; the route would return 501 at runtime", name)
		}
	}
}

// TestNoOrphanedHandlers fails when a handler exists that the spec no longer
// declares — the residue of a renamed operation.
func TestNoOrphanedHandlers(t *testing.T) {
	iface := reflect.TypeOf((*ServerInterface)(nil)).Elem()
	declared := make(map[string]bool, iface.NumMethod())
	for i := range iface.NumMethod() {
		declared[iface.Method(i).Name] = true
	}

	for name := range handlerMethodsInPackage(t) {
		if !declared[name] {
			t.Errorf("*Handlers.%s implements no operation in ServerInterface; it is dead code left by a rename", name)
		}
	}
}

// handlerMethodsInPackage returns the names of methods declared on *Handlers in
// the hand-written sources, excluding the generated file and tests.
func handlerMethodsInPackage(t *testing.T) map[string]bool {
	t.Helper()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}

	methods := make(map[string]bool)
	fset := token.NewFileSet()

	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") || name == "api.gen.go" {
			continue
		}

		file, err := parser.ParseFile(fset, filepath.Join(".", name), nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || len(fn.Recv.List) != 1 {
				continue
			}
			if receiverTypeName(fn.Recv.List[0].Type) == "Handlers" && fn.Name.IsExported() {
				methods[fn.Name.Name] = true
			}
		}
	}

	if len(methods) == 0 {
		t.Fatal("found no handler methods; the source scan is broken")
	}
	return methods
}

func receiverTypeName(expr ast.Expr) string {
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = star.X
	}
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return ""
}
