package ast

import (
	"strings"
	"testing"
)

func zeroPt() Pos { return Pos{} }

func TestWalkOrder(t *testing.T) {
	file := &File{
		Base: At(zeroPt(), zeroPt()),
		URI:  "test.bhaus",
		Decls: []Decl{
			&StructDecl{
				Base: At(zeroPt(), zeroPt()),
				Name: &QualifiedName{
					Segments: []*Ident{{Name: "User"}},
				},
				Members: []StructMember{
					&PropertyMember{
						Base: At(zeroPt(), zeroPt()),
						Name: &Ident{Name: "id"},
						Vis:  VisibilityPublic,
						Type: &SimpleType{Name: "Integer"},
					},
				},
			},
		},
	}

	var kinds []string
	Inspect(file, func(n Node) bool {
		switch n.(type) {
		case *File:
			kinds = append(kinds, "file")
		case *StructDecl:
			kinds = append(kinds, "struct")
		case *QualifiedName:
			kinds = append(kinds, "qname")
		case *Ident:
			kinds = append(kinds, "ident")
		case *PropertyMember:
			kinds = append(kinds, "prop")
		case *SimpleType:
			kinds = append(kinds, "simple")
		}
		return true
	})

	expected := "file,struct,qname,ident,prop,ident,simple"
	got := strings.Join(kinds, ",")
	if got != expected {
		t.Errorf("walk order:\n  got:      %s\n  expected: %s", got, expected)
	}
}

func TestNodeAt_Deepest(t *testing.T) {
	userStruct := &StructDecl{
		Base: At(Pos{Line: 0, Column: 0, Offset: 0}, Pos{Line: 1, Column: 26, Offset: 60}),
		Name: &QualifiedName{
			Base: At(Pos{Line: 0, Column: 7, Offset: 7}, Pos{Line: 0, Column: 11, Offset: 11}),
			Segments: []*Ident{
				{Base: At(Pos{Line: 0, Column: 7, Offset: 7}, Pos{Line: 0, Column: 11, Offset: 11}), Name: "User"},
			},
		},
		Members: []StructMember{
			&PropertyMember{
				Base: At(Pos{Line: 1, Column: 4, Offset: 20}, Pos{Line: 1, Column: 25, Offset: 55}),
				Name: &Ident{Base: At(Pos{Line: 1, Column: 11, Offset: 27}, Pos{Line: 1, Column: 15, Offset: 31}), Name: "name"},
				Vis:  VisibilityPublic,
				Type: &SimpleType{Base: At(Pos{Line: 1, Column: 17, Offset: 33}, Pos{Line: 1, Column: 23, Offset: 39}), Name: "String"},
			},
		},
	}

	file := &File{
		Base:  At(Pos{Line: 0, Column: 0, Offset: 0}, Pos{Line: 1, Column: 26, Offset: 60}),
		URI:   "test.bhaus",
		Decls: []Decl{userStruct},
	}

	// Cursor on "name" (line 1, col 12)
	n := NodeAt(file, 1, 12)
	if n == nil {
		t.Fatal("NodeAt returned nil")
	}
	ident, ok := n.(*Ident)
	if !ok {
		t.Fatalf("expected *Ident, got %T", n)
	}
	if ident.Name != "name" {
		t.Errorf("expected ident 'name', got %q", ident.Name)
	}

	// Cursor on "User" (line 0, col 8)
	n = NodeAt(file, 0, 8)
	if n == nil {
		t.Fatal("NodeAt returned nil for User")
	}
	qn, ok := n.(*QualifiedName)
	if !ok {
		t.Fatalf("expected *QualifiedName, got %T", n)
	}
	if qn.Simple() != "User" {
		t.Errorf("expected 'User', got %q", qn.Simple())
	}

	// Cursor outside the file
	n = NodeAt(file, 999, 0)
	if n != nil {
		t.Errorf("expected nil for out-of-bounds, got %T", n)
	}
}

func TestPathTo(t *testing.T) {
	userStruct := &StructDecl{
		Base: At(Pos{Line: 0, Column: 0, Offset: 0}, Pos{Line: 1, Column: 26, Offset: 60}),
		Name: &QualifiedName{
			Base: At(Pos{Line: 0, Column: 7, Offset: 7}, Pos{Line: 0, Column: 11, Offset: 11}),
			Segments: []*Ident{
				{Base: At(Pos{Line: 0, Column: 7, Offset: 7}, Pos{Line: 0, Column: 11, Offset: 11}), Name: "User"},
			},
		},
		Members: []StructMember{
			&PropertyMember{
				Base: At(Pos{Line: 1, Column: 4, Offset: 20}, Pos{Line: 1, Column: 25, Offset: 55}),
				Name: &Ident{Base: At(Pos{Line: 1, Column: 11, Offset: 27}, Pos{Line: 1, Column: 15, Offset: 31}), Name: "name"},
				Vis:  VisibilityPublic,
				Type: &SimpleType{Base: At(Pos{Line: 1, Column: 17, Offset: 33}, Pos{Line: 1, Column: 23, Offset: 39}), Name: "String"},
			},
		},
	}

	file := &File{
		Base:  At(Pos{Line: 0, Column: 0, Offset: 0}, Pos{Line: 1, Column: 26, Offset: 60}),
		URI:   "test.bhaus",
		Decls: []Decl{userStruct},
	}

	path := PathTo(file, 1, 12)
	if len(path) != 4 {
		t.Fatalf("expected path of length 4, got %d", len(path))
	}

	if _, ok := path[0].(*File); !ok {
		t.Errorf("path[0] expected *File, got %T", path[0])
	}
	if _, ok := path[1].(*StructDecl); !ok {
		t.Errorf("path[1] expected *StructDecl, got %T", path[1])
	}
	if _, ok := path[2].(*PropertyMember); !ok {
		t.Errorf("path[2] expected *PropertyMember, got %T", path[2])
	}
	if ident, ok := path[3].(*Ident); !ok {
		t.Errorf("path[3] expected *Ident, got %T", path[3])
	} else if ident.Name != "name" {
		t.Errorf("expected ident 'name', got %q", ident.Name)
	}
}
