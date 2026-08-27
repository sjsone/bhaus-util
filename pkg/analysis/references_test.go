package analysis

import (
	"testing"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

func zeroPt() ast.Pos { return ast.Pos{} }

func namedType(name string) *ast.NamedType {
	return &ast.NamedType{
		Name: &ast.QualifiedName{
			Segments: []*ast.Ident{{Name: name}},
		},
	}
}

func TestResolveSite_NamedType(t *testing.T) {
	file := &ast.File{
		URI: "test.bhaus",
		Decls: []ast.Decl{
			&ast.StructDecl{
				Name: &ast.QualifiedName{
					Segments: []*ast.Ident{{Name: "User"}},
				},
				Members: []ast.StructMember{
					&ast.PropertyMember{
						Name: &ast.Ident{Name: "name"},
						Vis:  ast.VisibilityPublic,
						Type: &ast.SimpleType{Name: "String"},
					},
					&ast.PropertyMember{
						Name: &ast.Ident{Name: "owner"},
						Vis:  ast.VisibilityPublic,
						Type: namedType("User"),
					},
				},
			},
		},
	}

	table := BuildSymbolTable(map[string]*ast.File{"test.bhaus": file})
	refs := CollectReferences(map[string]*ast.File{"test.bhaus": file}, table)

	if len(refs) != 1 {
		t.Fatalf("expected 1 reference, got %d", len(refs))
	}
	r := refs[0]
	if r.Kind != RefType {
		t.Errorf("expected RefType, got %v", r.Kind)
	}
	if r.Sym == nil {
		t.Fatal("expected resolved symbol, got nil")
	}
	if r.Sym.Name != "User" {
		t.Errorf("expected symbol 'User', got %q", r.Sym.Name)
	}
}

func TestReferencesTo_ExcludesDefinition(t *testing.T) {
	file := &ast.File{
		URI: "test.bhaus",
		Decls: []ast.Decl{
			&ast.StructDecl{
				Name: &ast.QualifiedName{
					Segments: []*ast.Ident{{Name: "User"}},
				},
				Members: []ast.StructMember{
					&ast.PropertyMember{
						Name: &ast.Ident{Name: "id"},
						Vis:  ast.VisibilityPublic,
						Type: &ast.SimpleType{Name: "Integer"},
					},
				},
			},
			&ast.StructDecl{
				Name: &ast.QualifiedName{
					Segments: []*ast.Ident{{Name: "Post"}},
				},
				Members: []ast.StructMember{
					&ast.PropertyMember{
						Name: &ast.Ident{Name: "author"},
						Vis:  ast.VisibilityPublic,
						Type: namedType("User"),
					},
				},
			},
		},
	}

	table := BuildSymbolTable(map[string]*ast.File{"test.bhaus": file})
	refs := CollectReferences(map[string]*ast.File{"test.bhaus": file}, table)
	userSym := table.LookupSimple("User")[0]
	userRefs := ReferencesTo(userSym, refs)
	if len(userRefs) != 1 {
		t.Fatalf("expected 1 reference to User (excluding definition), got %d", len(userRefs))
	}
}

func TestReferencesTo_NoFalsePositive(t *testing.T) {
	file := &ast.File{
		URI: "test.bhaus",
		Decls: []ast.Decl{
			&ast.StructDecl{
				Name: &ast.QualifiedName{
					Segments: []*ast.Ident{{Name: "User"}},
				},
			},
			&ast.StructDecl{
				Name: &ast.QualifiedName{
					Segments: []*ast.Ident{{Name: "UserController"}},
				},
				Members: []ast.StructMember{
					&ast.PropertyMember{
						Name: &ast.Ident{Name: "owner"},
						Vis:  ast.VisibilityPublic,
						Type: namedType("User"),
					},
				},
			},
		},
	}

	table := BuildSymbolTable(map[string]*ast.File{"test.bhaus": file})
	refs := CollectReferences(map[string]*ast.File{"test.bhaus": file}, table)
	if len(refs) != 1 {
		t.Fatalf("expected 1 reference, got %d", len(refs))
	}
	if refs[0].Sym.Name != "User" {
		t.Errorf("expected 'User', got %q", refs[0].Sym.Name)
	}
}

func TestResolveSite_FullPath(t *testing.T) {
	file := &ast.File{
		URI: "test.bhaus",
		Decls: []ast.Decl{
			&ast.StructDecl{
				Name: &ast.QualifiedName{
					Segments: []*ast.Ident{
						{Name: "Domain"}, {Name: "Entity"}, {Name: "User"},
					},
				},
			},
			&ast.StructDecl{
				Name: &ast.QualifiedName{
					Segments: []*ast.Ident{{Name: "Profile"}},
				},
				Members: []ast.StructMember{
					&ast.PropertyMember{
						Name: &ast.Ident{Name: "user"},
						Vis:  ast.VisibilityPublic,
						Type: &ast.NamedType{
							Name: &ast.QualifiedName{
								Segments: []*ast.Ident{
									{Name: "Domain"}, {Name: "Entity"}, {Name: "User"},
								},
							},
						},
					},
				},
			},
		},
	}

	table := BuildSymbolTable(map[string]*ast.File{"test.bhaus": file})
	refs := CollectReferences(map[string]*ast.File{"test.bhaus": file}, table)
	if len(refs) != 1 {
		t.Fatalf("expected 1 reference, got %d", len(refs))
	}
	if refs[0].Sym.FullName != "Domain/Entity/User" {
		t.Errorf("expected 'Domain/Entity/User', got %q", refs[0].Sym.FullName)
	}
}
