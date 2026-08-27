package ast

import (
	"testing"
)

func TestContains(t *testing.T) {
	sp := Span{Start: Pos{Line: 0, Column: 6}, End: Pos{Line: 0, Column: 15}}

	if !Contains(sp, Pos{Line: 0, Column: 6}) {
		t.Error("start of span should be contained")
	}
	if !Contains(sp, Pos{Line: 0, Column: 10}) {
		t.Error("middle of span should be contained")
	}
	if Contains(sp, Pos{Line: 0, Column: 15}) {
		t.Error("end of span (exclusive) should not be contained")
	}
	if Contains(sp, Pos{Line: 1, Column: 0}) {
		t.Error("position on next line should not be contained")
	}
}

func TestQualifiedName(t *testing.T) {
	qn := &QualifiedName{
		Segments: []*Ident{
			{Name: "Domain"},
			{Name: "Entity"},
			{Name: "User"},
		},
	}
	if qn.String() != "Domain/Entity/User" {
		t.Errorf("String() = %q, want %q", qn.String(), "Domain/Entity/User")
	}
	if qn.Simple() != "User" {
		t.Errorf("Simple() = %q, want %q", qn.Simple(), "User")
	}
	if !qn.IsPath() {
		t.Error("expected IsPath() == true")
	}

	single := &QualifiedName{
		Segments: []*Ident{
			{Name: "User"},
		},
	}
	if single.String() != "User" {
		t.Errorf("String() = %q, want %q", single.String(), "User")
	}
	if single.IsPath() {
		t.Error("expected IsPath() == false for single segment")
	}
}

func TestC4Path(t *testing.T) {
	cp := &C4Path{
		Segments: []*Ident{
			{Name: "MailSystem"},
			{Name: "MTA"},
		},
	}
	if cp.String() != "MailSystem.MTA" {
		t.Errorf("String() = %q, want %q", cp.String(), "MailSystem.MTA")
	}
}

func TestTypeRefInterface(t *testing.T) {
	// Compile-time check: these must satisfy TypeRef
	var _ TypeRef = (*SimpleType)(nil)
	var _ TypeRef = (*NamedType)(nil)
	var _ TypeRef = (*ArrayType)(nil)
	var _ TypeRef = (*OptionalType)(nil)
	var _ TypeRef = (*UnionType)(nil)
}

func TestMemberInterfaces(t *testing.T) {
	// Both concrete types satisfy all three member interfaces
	var _ ClassMember = (*PropertyMember)(nil)
	var _ ClassMember = (*MethodMember)(nil)
	var _ StructMember = (*PropertyMember)(nil)
	var _ StructMember = (*MethodMember)(nil)
	var _ ProtocolMember = (*PropertyMember)(nil)
	var _ ProtocolMember = (*MethodMember)(nil)

	// Type switch discrimination
	pm := &PropertyMember{Vis: VisibilityPublic}
	mm := &MethodMember{Vis: VisibilityPrivate, Override: true}

	for _, m := range []ClassMember{pm, mm} {
		switch m := m.(type) {
		case *PropertyMember:
			if m.Visibility() != VisibilityPublic {
				t.Error("expected PUBLIC visibility")
			}
		case *MethodMember:
			if m.Visibility() != VisibilityPrivate || !m.Override {
				t.Error("expected PRIVATE visibility with OVERRIDE")
			}
		}
	}
}

func TestVisibilityString(t *testing.T) {
	if VisibilityPublic.String() != "PUBLIC" {
		t.Errorf("expected PUBLIC, got %q", VisibilityPublic.String())
	}
	if VisibilityPrivate.String() != "PRIVATE" {
		t.Errorf("expected PRIVATE, got %q", VisibilityPrivate.String())
	}
	if VisibilityProtected.String() != "PROTECTED" {
		t.Errorf("expected PROTECTED, got %q", VisibilityProtected.String())
	}
	if Visibility(99).String() != "UNKNOWN" {
		t.Errorf("expected UNKNOWN for invalid value, got %q", Visibility(99).String())
	}
}

func TestDeclInterface(t *testing.T) {
	var _ Decl = (*ClassDecl)(nil)
	var _ Decl = (*StructDecl)(nil)
	var _ Decl = (*ProtocolDecl)(nil)
	var _ Decl = (*FunctionDecl)(nil)
	var _ Decl = (*VersionDecl)(nil)
	var _ Decl = (*IncludeDecl)(nil)
	var _ Decl = (*ExternDecl)(nil)
	var _ Decl = (*SystemDecl)(nil)
	var _ Decl = (*ConnectionDecl)(nil)
}
