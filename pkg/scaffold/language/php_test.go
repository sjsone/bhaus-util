package language

import (
	"strings"
	"testing"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

func TestPHPClassWithMethod(t *testing.T) {
	decl := &ast.ClassDecl{
		Name:    qname("UserController"),
		Extends: qname("Base", "Controller"),
		Members: []ast.ClassMember{
			&ast.PropertyMember{Name: &ast.Ident{Name: "repo"}, Vis: ast.VisibilityPrivate, Type: &ast.NamedType{Name: qname("Domain", "Repository")}},
			&ast.MethodMember{
				Name:       &ast.Ident{Name: "index"},
				Vis:        ast.VisibilityPublic,
				ReturnType: &ast.SimpleType{Name: "String"},
				Intents:    []*ast.FunctionalIntent{{Text: "render index"}},
			},
		},
	}
	out := render(t, mustGet(t, "php"), decl)
	for _, want := range []string{
		"<?php",
		"class UserController extends Controller {",
		"private Repository $repo;",
		"public function index(): string {",
		"// TODO: render index",
		"throw new \\Exception('not implemented');",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
}

func TestPHPInterfaceWithOptionalParam(t *testing.T) {
	decl := &ast.ProtocolDecl{
		Name: qname("Repository"),
		Members: []ast.ProtocolMember{
			&ast.MethodMember{
				Name:       &ast.Ident{Name: "findById"},
				Vis:        ast.VisibilityPublic,
				Params:     []*ast.Parameter{{Type: &ast.SimpleType{Name: "Integer"}}},
				ReturnType: &ast.OptionalType{Inner: &ast.NamedType{Name: qname("Domain", "User")}},
			},
		},
	}
	out := render(t, mustGet(t, "php"), decl)
	if !strings.Contains(out, "interface Repository {") {
		t.Errorf("missing interface\n%s", out)
	}
	if !strings.Contains(out, "public function findById(int $arg0): ?User;") {
		t.Errorf("missing method signature\n%s", out)
	}
}

func TestPHPPSR4NamespacePathAndUse(t *testing.T) {
	decl := &ast.ClassDecl{
		Name: qname("Controller", "UserController"),
		Members: []ast.ClassMember{
			&ast.PropertyMember{Name: &ast.Ident{Name: "repo"}, Vis: ast.VisibilityPrivate, Type: &ast.NamedType{Name: qname("Domain", "Repository")}},
			&ast.MethodMember{
				Name:       &ast.Ident{Name: "find"},
				Vis:        ast.VisibilityPublic,
				ReturnType: &ast.OptionalType{Inner: &ast.NamedType{Name: qname("Domain", "User")}},
			},
		},
	}
	files, err := mustGet(t, "php").Scaffold([]ast.Decl{decl})
	if err != nil {
		t.Fatalf("Scaffold: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	f := files[0]
	if f.Path != "Controller/UserController.php" {
		t.Errorf("path: got %q, want Controller/UserController.php", f.Path)
	}
	for _, want := range []string{
		"namespace Controller;",
		"use Domain\\Repository;",
		"use Domain\\User;",
		"class UserController {",
		"private Repository $repo;",
		"public function find(): ?User {",
	} {
		if !strings.Contains(f.Content, want) {
			t.Errorf("content missing %q\n---\n%s", want, f.Content)
		}
	}
}

func TestPHPUseForExtendsAndImplements(t *testing.T) {
	// A class in the global namespace imports types that it extends or
	// implements from other namespaces.
	decl := &ast.ClassDecl{
		Name:       qname("Model"),
		Extends:    qname("Base", "AbstractModel"),
		Implements: []*ast.QualifiedName{qname("Base", "Entity")},
	}
	files, _ := mustGet(t, "php").Scaffold([]ast.Decl{decl})
	for _, want := range []string{
		"use Base\\AbstractModel;",
		"use Base\\Entity;",
		"class Model extends AbstractModel implements Entity {",
	} {
		if !strings.Contains(files[0].Content, want) {
			t.Errorf("content missing %q\n---\n%s", want, files[0].Content)
		}
	}
}

func TestPHPNoUseForSameNamespace(t *testing.T) {
	// A class in Domain can reference another Domain type without a use statement.
	decl := &ast.ClassDecl{
		Name: qname("Domain", "UserService"),
		Members: []ast.ClassMember{
			&ast.PropertyMember{Name: &ast.Ident{Name: "repo"}, Vis: ast.VisibilityPrivate, Type: &ast.NamedType{Name: qname("Domain", "Repository")}},
		},
	}
	files, _ := mustGet(t, "php").Scaffold([]ast.Decl{decl})
	if strings.Contains(files[0].Content, "use ") {
		t.Errorf("same-namespace reference should not emit a use statement\n%s", files[0].Content)
	}
}

func TestPHPMultipleDeclsProduceMultipleFiles(t *testing.T) {
	decls := []ast.Decl{
		&ast.ClassDecl{Name: qname("Controller", "UserController")},
		&ast.StructDecl{Name: qname("Domain", "User")},
	}
	files, _ := mustGet(t, "php").Scaffold(decls)
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2", len(files))
	}
	paths := map[string]bool{files[0].Path: true, files[1].Path: true}
	for _, want := range []string{"Controller/UserController.php", "Domain/User.php"} {
		if !paths[want] {
			t.Errorf("missing file %q; got %v", want, paths)
		}
	}
}

func TestPHPStructAsClassAndArray(t *testing.T) {
	decl := &ast.StructDecl{
		Name: qname("Bag"),
		Members: []ast.StructMember{
			&ast.PropertyMember{Name: &ast.Ident{Name: "items"}, Vis: ast.VisibilityPublic, Type: &ast.ArrayType{Elem: &ast.SimpleType{Name: "String"}}},
		},
	}
	out := render(t, mustGet(t, "php"), decl)
	if !strings.Contains(out, "class Bag {") || !strings.Contains(out, "public array $items;") {
		t.Errorf("missing struct-as-class output\n%s", out)
	}
}
