package scaffold

import (
	"testing"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

// goLikeMap is a TypeMap that resembles the Go target. Multiple type tests use it.
var goLikeMap = TypeMap{
	Simple:   map[string]string{"String": "string", "Integer": "int"},
	Array:    "[]%s",
	Optional: "*%s",
	Union:    "any", // Go has no unions. This verb-less format ignores the arguments.
	Named:    "%s",
}

func TestRenderSimpleKnown(t *testing.T) {
	got := goLikeMap.Render(&ast.SimpleType{Name: "String"})
	if got != "string" {
		t.Fatalf("Render(String): got %q, want %q", got, "string")
	}
}

func TestRenderSimpleUnknownPassesThrough(t *testing.T) {
	got := goLikeMap.Render(&ast.SimpleType{Name: "Widget"})
	if got != "Widget" {
		t.Fatalf("Render(Widget): got %q, want %q", got, "Widget")
	}
}

func TestRenderNamedUsesSimpleName(t *testing.T) {
	got := goLikeMap.Render(&ast.NamedType{Name: qname("Domain", "User")})
	if got != "User" {
		t.Fatalf("Render(Domain/User): got %q, want %q", got, "User")
	}
}

func TestRenderNestedArrayOptionalNamed(t *testing.T) {
	// Array[?Domain/User]
	typ := &ast.ArrayType{Elem: &ast.OptionalType{Inner: &ast.NamedType{Name: qname("Domain", "User")}}}
	got := goLikeMap.Render(typ)
	if got != "[]*User" {
		t.Fatalf("Render(Array[?Domain/User]): got %q, want %q", got, "[]*User")
	}
}

func TestRenderUnionVerbless(t *testing.T) {
	typ := &ast.UnionType{
		Left:  &ast.SimpleType{Name: "String"},
		Right: &ast.SimpleType{Name: "Integer"},
	}
	if got := goLikeMap.Render(typ); got != "any" {
		t.Fatalf("Render(union) with verb-less format: got %q, want %q", got, "any")
	}
}

func TestRenderUnionWithVerbs(t *testing.T) {
	tsLike := TypeMap{
		Simple: map[string]string{"String": "string", "Integer": "number"},
		Union:  "%s | %s",
		Named:  "%s",
	}
	typ := &ast.UnionType{
		Left:  &ast.SimpleType{Name: "String"},
		Right: &ast.SimpleType{Name: "Integer"},
	}
	if got := tsLike.Render(typ); got != "string | number" {
		t.Fatalf("Render(union): got %q, want %q", got, "string | number")
	}
}

func TestRenderNil(t *testing.T) {
	if got := goLikeMap.Render(nil); got != "" {
		t.Fatalf("Render(nil): got %q, want empty", got)
	}
}
