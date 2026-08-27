package parser

import (
	"strings"
	"testing"

	"github.com/sjsone/bhaus-util/pkg/ast"
)

func TestParseFileBasic(t *testing.T) {
	content := `
CLASS ValidClass:
    PUBLIC validMethod()
    PRIVATE anotherMethod(x: String)
STRUCT ValidStruct:
PROTOCOL ValidProtocol:
`
	file, err := Parse("test.bhaus", content)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	// Expect 3 declarations
	if len(file.Decls) != 3 {
		t.Errorf("expected 3 decls, got %d", len(file.Decls))
		for i, d := range file.Decls {
			t.Logf("  decl[%d]: %T", i, d)
		}
	}

	// First: CLASS ValidClass
	cd, ok := file.Decls[0].(*ast.ClassDecl)
	if !ok {
		t.Fatalf("expected *ClassDecl, got %T", file.Decls[0])
	}
	if cd.Name.Simple() != "ValidClass" {
		t.Errorf("expected class name 'ValidClass', got %q", cd.Name.Simple())
	}
	if len(cd.Members) != 2 {
		t.Errorf("expected 2 members, got %d", len(cd.Members))
	}

	// Method 1
	mm, ok := cd.Members[0].(*ast.MethodMember)
	if !ok {
		t.Fatalf("expected *MethodMember, got %T", cd.Members[0])
	}
	if mm.Name.Name != "validMethod" {
		t.Errorf("expected method 'validMethod', got %q", mm.Name.Name)
	}
	if mm.Visibility() != ast.VisibilityPublic {
		t.Error("expected PUBLIC visibility")
	}

	// Method 2 has a parameter
	mm2, ok := cd.Members[1].(*ast.MethodMember)
	if !ok {
		t.Fatalf("expected *MethodMember, got %T", cd.Members[1])
	}
	if mm2.Name.Name != "anotherMethod" {
		t.Errorf("expected method 'anotherMethod', got %q", mm2.Name.Name)
	}
	if len(mm2.Params) != 1 {
		t.Errorf("expected 1 param, got %d", len(mm2.Params))
	}
}

func TestParseFileC4Model(t *testing.T) {
	content := `VERSION 0.1
SYSTEM MailSystem "Mail System":
    CONTAINER MDA:
    CONTAINER MTA "Mail Transfer Agent":
CONNECTION MailSystem.MTA -> MailSystem.MDA
`
	file, err := Parse("test.bhaus", content)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	// Expect: VERSION, SYSTEM, CONNECTION = 3 decls
	if len(file.Decls) != 3 {
		t.Errorf("expected 3 decls, got %d", len(file.Decls))
	}

	// System has 2 containers
	sd, ok := file.Decls[1].(*ast.SystemDecl)
	if !ok {
		t.Fatalf("expected *SystemDecl, got %T", file.Decls[1])
	}
	if sd.Name.Name != "MailSystem" {
		t.Errorf("expected 'MailSystem', got %q", sd.Name.Name)
	}
	if sd.Description != "Mail System" {
		t.Errorf("expected description 'Mail System', got %q", sd.Description)
	}
	if len(sd.Containers) != 2 {
		t.Errorf("expected 2 containers, got %d", len(sd.Containers))
	}

	// Connection
	conn, ok := file.Decls[2].(*ast.ConnectionDecl)
	if !ok {
		t.Fatalf("expected *ConnectionDecl, got %T", file.Decls[2])
	}
	if conn.Source.String() != "MailSystem.MTA" {
		t.Errorf("expected source 'MailSystem.MTA', got %q", conn.Source.String())
	}
	if conn.Target.String() != "MailSystem.MDA" {
		t.Errorf("expected target 'MailSystem.MDA', got %q", conn.Target.String())
	}
	if conn.Arrow != ast.ArrowUnidirectional {
		t.Error("expected unidirectional arrow")
	}
}

func TestParseFileContextualNames(t *testing.T) {
	content := `VERSION 0.1
CLASS Controller/AbstractIntertiaController EXTENDS ActionController:
    OVERRIDE PUBLIC renderView()
PROTOCOL Domain/AssetVersion/StrategyProtocol:
    PUBLIC resolveAssetVersion(Request): String
STRUCT Domain/Render/Context:
    PUBLIC component: String
`
	file, err := Parse("test.bhaus", content)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	// Class with path name (index 1, after VERSION)
	cd, ok := file.Decls[1].(*ast.ClassDecl)
	if !ok {
		t.Fatalf("expected *ClassDecl at index 1, got %T", file.Decls[1])
	}
	if cd.Name.String() != "Controller/AbstractIntertiaController" {
		t.Errorf("expected 'Controller/AbstractIntertiaController', got %q", cd.Name.String())
	}
	if cd.Name.Simple() != "AbstractIntertiaController" {
		t.Errorf("expected 'AbstractIntertiaController', got %q", cd.Name.Simple())
	}
	if cd.Extends == nil {
		t.Error("expected EXTENDS clause")
	} else if cd.Extends.String() != "ActionController" {
		t.Errorf("expected extends 'ActionController', got %q", cd.Extends.String())
	}

	// Override method
	if len(cd.Members) != 1 {
		t.Fatalf("expected 1 member, got %d", len(cd.Members))
	}
	mm, ok := cd.Members[0].(*ast.MethodMember)
	if !ok {
		t.Fatalf("expected *MethodMember, got %T", cd.Members[0])
	}
	if !mm.Override {
		t.Error("expected OVERRIDE on method")
	}
	if mm.Name.Name != "renderView" {
		t.Errorf("expected 'renderView', got %q", mm.Name.Name)
	}

	// Protocol with path name
	pd, ok := file.Decls[2].(*ast.ProtocolDecl)
	if !ok {
		t.Fatalf("expected *ProtocolDecl at index 2, got %T", file.Decls[2])
	}
	if pd.Name.String() != "Domain/AssetVersion/StrategyProtocol" {
		t.Errorf("expected 'Domain/AssetVersion/StrategyProtocol', got %q", pd.Name.String())
	}
	if pd.Name.Simple() != "StrategyProtocol" {
		t.Errorf("expected 'StrategyProtocol', got %q", pd.Name.Simple())
	}

	// Struct with path name
	sd, ok := file.Decls[3].(*ast.StructDecl)
	if !ok {
		t.Fatalf("expected *StructDecl at index 3, got %T", file.Decls[3])
	}
	if sd.Name.String() != "Domain/Render/Context" {
		t.Errorf("expected 'Domain/Render/Context', got %q", sd.Name.String())
	}
}

func TestParseFileFunctions(t *testing.T) {
	content := `VERSION 0.1
FUNCTION helloWorld()
FUNCTION calculateSum(a: Int, b: Int): Integer
FUNC greet(name: String): String
`
	file, err := Parse("test.bhaus", content)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	funcCount := 0
	for _, d := range file.Decls {
		if _, ok := d.(*ast.FunctionDecl); ok {
			funcCount++
		}
	}
	if funcCount != 3 {
		t.Errorf("expected 3 functions, got %d", funcCount)
	}
}

// TestParseC4NestingWithComments guards against a nesting bug. Comments
// between sibling C4 containers could break the nesting. That popped later
// containers out of the system. Then their qualified names, such as
// MailSystem.MTA, never resolved.
func TestParseC4NestingWithComments(t *testing.T) {
	content := `VERSION 0.1
SYSTEM MailSystem "Backend":
    # first container
    CONTAINER MDA:
    # second container
    CONTAINER MTA "Transfer":
        COMPONENT SmtpServer:
    # third container
    CONTAINER Database:
`
	file, err := Parse("test.bhaus", content)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(file.Errors) != 0 {
		t.Fatalf("unexpected parse errors: %+v", file.Errors)
	}

	var sys *ast.SystemDecl
	for _, d := range file.Decls {
		if s, ok := d.(*ast.SystemDecl); ok {
			sys = s
		}
	}
	if sys == nil {
		t.Fatal("no SystemDecl parsed")
	}
	// All three containers must be nested inside the system, not popped out.
	if len(sys.Containers) != 3 {
		t.Fatalf("expected 3 containers nested in MailSystem, got %d", len(sys.Containers))
	}
	names := []string{sys.Containers[0].Name.Name, sys.Containers[1].Name.Name, sys.Containers[2].Name.Name}
	for i, want := range []string{"MDA", "MTA", "Database"} {
		if names[i] != want {
			t.Errorf("container[%d] = %q, want %q", i, names[i], want)
		}
	}
	// The nested component must survive too.
	if len(sys.Containers[1].Components) != 1 || sys.Containers[1].Components[0].Name.Name != "SmtpServer" {
		t.Errorf("expected MTA to contain component SmtpServer, got %+v", sys.Containers[1].Components)
	}
	// Comments are still collected (now as extras).
	if len(file.Comments) != 3 {
		t.Errorf("expected 3 comments collected, got %d", len(file.Comments))
	}
}

func TestParseFileConnectionShorthand(t *testing.T) {
	content := `VERSION 0.1
SYSTEM Webmail:
    CONNECTION => MailSystem.MTA
    CONTAINER Backend:
        CONNECTION => MailSystem.Database
CONNECTION Webmail.Backend -> MailSystem.MTA
`
	file, err := Parse("test.bhaus", content)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	sd, ok := file.Decls[1].(*ast.SystemDecl)
	if !ok {
		t.Fatalf("expected *SystemDecl at index 1")
	}

	// System-level shorthand
	if len(sd.Connections) != 1 {
		t.Errorf("expected 1 system-level connection shorthand, got %d", len(sd.Connections))
	} else if sd.Connections[0].Target.String() != "MailSystem.MTA" {
		t.Errorf("expected target 'MailSystem.MTA', got %q", sd.Connections[0].Target.String())
	}

	// Container-level shorthand
	if len(sd.Containers) != 1 {
		t.Fatalf("expected 1 container, got %d", len(sd.Containers))
	}
	ct := sd.Containers[0]
	if ct.Name.Name != "Backend" {
		t.Errorf("expected container 'Backend', got %q", ct.Name.Name)
	}
	if len(ct.Connections) != 1 {
		t.Errorf("expected 1 container-level connection, got %d", len(ct.Connections))
	}
}

func TestParseFileErrorDetection(t *testing.T) {
	content := `
CLASS MissingColon
STRUCT MissingColon
`
	file, err := Parse("test.bhaus", content)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(file.Errors) == 0 {
		t.Error("expected parse errors for malformed input")
	}
}

func TestParseFilePropertyMember(t *testing.T) {
	content := `VERSION 0.1
STRUCT Domain/Entity/User:
    PUBLIC id: Integer
    PRIVATE name: String
`
	file, err := Parse("test.bhaus", content)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	sd, ok := file.Decls[1].(*ast.StructDecl)
	if !ok {
		t.Fatalf("expected *StructDecl, got %T", file.Decls[1])
	}
	if len(sd.Members) != 2 {
		t.Fatalf("expected 2 struct members, got %d", len(sd.Members))
	}

	pm, ok := sd.Members[0].(*ast.PropertyMember)
	if !ok {
		t.Fatalf("expected *PropertyMember, got %T", sd.Members[0])
	}
	if pm.Name.Name != "id" {
		t.Errorf("expected property 'id', got %q", pm.Name.Name)
	}
	if pm.Visibility() != ast.VisibilityPublic {
		t.Error("expected PUBLIC visibility")
	}
	st, ok := pm.Type.(*ast.SimpleType)
	if !ok {
		t.Errorf("expected SimpleType, got %T", pm.Type)
	} else if st.Name != "Integer" {
		t.Errorf("expected type 'Integer', got %q", st.Name)
	}
}

func TestParseFileIncludeDirectives(t *testing.T) {
	content := `VERSION 0.1
INCLUDE *.bhaus
INCLUDE base.bhaus
`
	file, err := Parse("test.bhaus", content)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	incCount := 0
	for _, d := range file.Decls {
		if inc, ok := d.(*ast.IncludeDecl); ok {
			if inc.Pattern == "*.bhaus" || inc.Pattern == "base.bhaus" {
				incCount++
			}
		}
	}
	if incCount != 2 {
		t.Errorf("expected 2 include directives, got %d", incCount)
	}
}

func TestParseFileNestedTypes(t *testing.T) {
	content := `VERSION 0.1
STRUCT Test:
    PUBLIC items: Array[String]
    PUBLIC maybe: ?Integer
    PUBLIC result: String|Int
`
	file, err := Parse("test.bhaus", content)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	sd, ok := file.Decls[1].(*ast.StructDecl)
	if !ok {
		t.Fatalf("expected *StructDecl")
	}
	if len(sd.Members) != 3 {
		t.Fatalf("expected 3 members, got %d", len(sd.Members))
	}

	// Array type
	pm0, _ := sd.Members[0].(*ast.PropertyMember)
	at, _ := pm0.Type.(*ast.ArrayType)
	if at == nil || at.Elem.(*ast.SimpleType).Name != "String" {
		t.Error("expected Array[String]")
	}

	// Optional type
	pm1, _ := sd.Members[1].(*ast.PropertyMember)
	ot, _ := pm1.Type.(*ast.OptionalType)
	if ot == nil || ot.Inner.(*ast.SimpleType).Name != "Integer" {
		t.Error("expected ?Integer")
	}

	// Union type
	pm2, _ := sd.Members[2].(*ast.PropertyMember)
	ut, _ := pm2.Type.(*ast.UnionType)
	if ut == nil {
		t.Error("expected union type")
	}
}

// TestReparseAfterEditKeepsPositions guards against stale tree-sitter node
// positions after an edit. The LSP reuses a single Cache across two parses
// of the same URI on didChange. An edit that inserts a line at the top must
// shift every declaration's reported position down. Otherwise position-based
// features like go-to-definition break.
func TestReparseAfterEditKeepsPositions(t *testing.T) {
	cache := NewCache()
	defer cache.Close()
	uri := "file://edit.bhaus"

	v1 := "STRUCT User:\n    PUBLIC id: Integer\n"
	if _, err := cache.Parse(uri, v1); err != nil {
		t.Fatalf("v1 parse: %v", err)
	}

	// Simulate an edit that shifts everything down one line.
	v2 := "# inserted at the top\nSTRUCT User:\n    PUBLIC id: Integer\n"
	f2, err := cache.Parse(uri, v2)
	if err != nil {
		t.Fatalf("v2 parse: %v", err)
	}

	var sd *ast.StructDecl
	for _, d := range f2.Decls {
		if s, ok := d.(*ast.StructDecl); ok {
			sd = s
		}
	}
	if sd == nil {
		t.Fatalf("no struct decl in v2 (decls=%d, errors=%d)", len(f2.Decls), len(f2.Errors))
	}
	if got := sd.Name.Pos().Line; got != 1 {
		t.Errorf("struct name reported at line %d, want 1 (stale positions after edit)", got)
	}

	// The reparse must pick up the inserted comment.
	if len(f2.Comments) != 1 {
		t.Errorf("expected 1 comment after edit, got %d", len(f2.Comments))
	}

	// The cursor over "User" at its new position resolves to the name node.
	path := ast.PathTo(f2, 1, 8)
	foundName := false
	for _, n := range path {
		if id, ok := n.(*ast.Ident); ok && id.Name == "User" {
			foundName = true
		}
	}
	if !foundName {
		t.Errorf("PathTo at the edited position did not reach the 'User' name node; path=%v", path)
	}
}

func TestParseFileComments(t *testing.T) {
	content := `VERSION 0.1
# top-level comment
STRUCT User:
    # nested member comment
    PUBLIC id: Integer #    trailing, extra spaces trimmed
SYSTEM S:
    CONTAINER C:
        # deeply nested C4 comment
`
	file, err := Parse("test.bhaus", content)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(file.Errors) != 0 {
		t.Fatalf("expected no parse errors, got %d: %+v", len(file.Errors), file.Errors)
	}

	// The parser collects every comment in source order, regardless of
	// nesting depth. It strips the leading "#" and surrounding whitespace
	// from each comment.
	want := []string{
		"top-level comment",
		"nested member comment",
		"trailing, extra spaces trimmed",
		"deeply nested C4 comment",
	}
	if len(file.Comments) != len(want) {
		t.Fatalf("expected %d comments, got %d", len(want), len(file.Comments))
	}
	for i, w := range want {
		if file.Comments[i].Text != w {
			t.Errorf("comment[%d]: expected %q, got %q", i, w, file.Comments[i].Text)
		}
	}

	// Nested comments must not leak into declaration members. The struct has
	// exactly one real member (id), not a phantom member for its comment.
	sd, ok := file.Decls[1].(*ast.StructDecl)
	if !ok {
		t.Fatalf("expected *StructDecl at index 1, got %T", file.Decls[1])
	}
	if len(sd.Members) != 1 {
		t.Errorf("expected 1 struct member (comment excluded), got %d", len(sd.Members))
	}
}

func TestParseFileExtern(t *testing.T) {
	content := `VERSION 0.1
EXTERN Domain/ExternalType
`
	file, err := Parse("test.bhaus", content)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	ed, ok := file.Decls[1].(*ast.ExternDecl)
	if !ok {
		t.Fatalf("expected *ExternDecl")
	}
	if ed.Name.String() != "Domain/ExternalType" {
		t.Errorf("expected 'Domain/ExternalType', got %q", ed.Name.String())
	}
}

func TestParseFunctionalIntents(t *testing.T) {
	content := `VERSION 0.1

PROTOCOL Base/Bla:
    PUBLIC rec: Base/Entity
    PUBLIC toLowerCase(String): String
        > walk over every char in String and swap that for the lowercase

FUNCTION calculateTotal(Array[Integer]): Integer
    > sum every element
    > return the total
`
	file, err := Parse("test.bhaus", content)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(file.Errors) != 0 {
		t.Fatalf("expected no syntax errors, got %d: %+v", len(file.Errors), file.Errors)
	}

	// The protocol method carries one intent. The property before it carries none.
	pd, ok := file.Decls[1].(*ast.ProtocolDecl)
	if !ok {
		t.Fatalf("expected *ProtocolDecl, got %T", file.Decls[1])
	}
	if len(pd.Members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(pd.Members))
	}
	mm, ok := pd.Members[1].(*ast.MethodMember)
	if !ok {
		t.Fatalf("expected method member, got %T", pd.Members[1])
	}
	if len(mm.Intents) != 1 {
		t.Fatalf("expected 1 intent on method, got %d", len(mm.Intents))
	}
	if got := mm.Intents[0].Text; got != "walk over every char in String and swap that for the lowercase" {
		t.Errorf("unexpected intent text: %q", got)
	}

	// Top-level function carries two intents in source order.
	fd, ok := file.Decls[2].(*ast.FunctionDecl)
	if !ok {
		t.Fatalf("expected *FunctionDecl, got %T", file.Decls[2])
	}
	if len(fd.Intents) != 2 {
		t.Fatalf("expected 2 intents on function, got %d", len(fd.Intents))
	}
	if fd.Intents[0].Text != "sum every element" || fd.Intents[1].Text != "return the total" {
		t.Errorf("unexpected function intents: %q, %q", fd.Intents[0].Text, fd.Intents[1].Text)
	}
}

func TestSyntaxErrorMessages(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		wantSub string // substring the first error message must contain
	}{
		{
			name:    "misplaced functional intent",
			src:     "> asdf\n",
			wantSub: "an intent line (\"> ...\") must be inside a function or method body",
		},
		{
			name:    "missing colon after class header",
			src:     "CLASS C\n",
			wantSub: `expected ":" in a CLASS declaration`,
		},
		{
			name:    "unexpected token in member",
			src:     "PROTOCOL P:\n    PUBLIC foo(: String\n",
			wantSub: "unexpected",
		},
		{
			name:    "top-level connection shorthand",
			src:     "CONNECTION => MailSystem.MTA\n",
			wantSub: "needs an enclosing SYSTEM",
		},
		{
			name:    "connection arrow with missing target",
			src:     "CONNECTION A =>\n",
			wantSub: "expected an identifier",
		},
		{
			name:    "connection with unidirectional arrow",
			src:     "CONNECTION -> B\n",
			wantSub: "expected an identifier",
		},
		{
			name:    "malformed nested connection shorthand",
			src:     "SYSTEM S:\n  CONNECTION =>\n",
			wantSub: "expected an identifier",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file, err := Parse("test.bhaus", tc.src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if len(file.Errors) == 0 {
				t.Fatalf("expected at least one syntax error, got none")
			}
			got := file.Errors[0].Message
			if !strings.Contains(got, tc.wantSub) {
				t.Errorf("message %q does not contain %q", got, tc.wantSub)
			}
			t.Logf("message: %q", got)
		})
	}
}
