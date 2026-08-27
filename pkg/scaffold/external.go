package scaffold

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/sjsone/bhaus-util/pkg/ast"
	"gopkg.in/yaml.v3"
)

// yamlDef is the on-disk definition of a third-party language. The Go engine does
// the recursive type mapping, driven by Types/Formats. It then hands the templates
// a view model whose types are already rendered strings.
type yamlDef struct {
	Lang      string            `yaml:"language"`
	Extension string            `yaml:"file_extension"` // output file extension (default: language name)
	Types     map[string]string `yaml:"types"`
	Formats   yamlFormats       `yaml:"formats"`
	Templates map[string]string `yaml:"templates"` // keys: class, protocol, struct, function
}

// yamlFormats holds the compound-type and parameter format strings. Type formats
// feed TypeMap. Param and ParamSep control parameter-list rendering.
type yamlFormats struct {
	Array    string `yaml:"array"`
	Optional string `yaml:"optional"`
	Union    string `yaml:"union"`
	Named    string `yaml:"named"`
	Param    string `yaml:"param"`     // %[1]s = name, %[2]s = type
	ParamSep string `yaml:"param_sep"` // default ", "
}

// yamlScaffolder wraps a parsed yamlDef so a data-driven language satisfies the
// same Scaffolder interface as the compiled-in languages.
type yamlScaffolder struct {
	lang     string
	ext      string
	types    TypeMap
	param    string
	paramSep string
	tmpl     *template.Template
}

func (s *yamlScaffolder) Language() string { return s.lang }

// loadYAMLDef parses one YAML definition into a Scaffolder.
func loadYAMLDef(data []byte) (Scaffolder, error) {
	var def yamlDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}
	if def.Lang == "" {
		return nil, fmt.Errorf("yaml definition missing 'language'")
	}

	named := def.Formats.Named
	if named == "" {
		named = "%s"
	}
	param := def.Formats.Param
	if param == "" {
		param = "%[1]s %[2]s"
	}
	sep := def.Formats.ParamSep
	if sep == "" {
		sep = ", "
	}
	ext := def.Extension
	if ext == "" {
		ext = def.Lang
	}

	root := template.New(def.Lang)
	for name, body := range def.Templates {
		if _, err := root.New(name).Parse(body); err != nil {
			return nil, fmt.Errorf("template %q: %w", name, err)
		}
	}

	return &yamlScaffolder{
		lang: def.Lang,
		ext:  ext,
		types: TypeMap{
			Simple:   def.Types,
			Array:    def.Formats.Array,
			Optional: def.Formats.Optional,
			Union:    def.Formats.Union,
			Named:    named,
		},
		param:    param,
		paramSep: sep,
		tmpl:     root,
	}, nil
}

// LoadDir loads every *.yaml / *.yml definition in dir. A missing directory yields
// no scaffolders and no error, so an absent template dir is not fatal.
func LoadDir(dir string) ([]Scaffolder, error) {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Scaffolder
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if ext := filepath.Ext(e.Name()); ext != ".yaml" && ext != ".yml" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		s, err := loadYAMLDef(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		out = append(out, s)
	}
	return out, nil
}

func (s *yamlScaffolder) Scaffold(decls []ast.Decl) ([]GeneratedFile, error) {
	w := &strings.Builder{}
	for _, d := range decls {
		var (
			tmpl string
			data any
		)
		switch decl := d.(type) {
		case *ast.ClassDecl:
			tmpl, data = "class", s.classView(decl.Name.Simple(), decl.Extends, decl.Implements, Members(decl.Members))
		case *ast.StructDecl:
			tmpl = "struct"
			if s.tmpl.Lookup("struct") == nil {
				tmpl = "class"
			}
			data = s.classView(decl.Name.Simple(), nil, nil, Members(decl.Members))
		case *ast.ProtocolDecl:
			tmpl, data = "protocol", s.interfaceView(decl.Name.Simple(), Members(decl.Members))
		case *ast.FunctionDecl:
			tmpl, data = "function", s.functionView(decl)
		default:
			continue // C4, INCLUDE, VERSION, EXTERN
		}
		if s.tmpl.Lookup(tmpl) == nil {
			continue // language does not define this construct
		}
		if err := s.tmpl.ExecuteTemplate(w, tmpl, data); err != nil {
			return nil, fmt.Errorf("%s template %q: %w", s.lang, tmpl, err)
		}
		w.WriteString("\n\n")
	}
	return []GeneratedFile{{Path: "scaffold." + s.ext, Content: w.String()}}, nil
}

func (s *yamlScaffolder) classView(name string, extends *ast.QualifiedName, impls []*ast.QualifiedName, ms []ast.ClassMember) ClassView {
	v := ClassView{Name: name}
	if extends != nil {
		v.Extends = extends.Simple()
	}
	for _, impl := range impls {
		v.Implements = append(v.Implements, impl.Simple())
	}
	for _, m := range ms {
		switch member := m.(type) {
		case *ast.PropertyMember:
			v.Properties = append(v.Properties, s.propertyView(member))
		case *ast.MethodMember:
			v.Methods = append(v.Methods, s.methodView(member))
		}
	}
	return v
}

func (s *yamlScaffolder) interfaceView(name string, ms []ast.ClassMember) InterfaceView {
	v := InterfaceView{Name: name}
	for _, m := range ms {
		switch member := m.(type) {
		case *ast.PropertyMember:
			v.Properties = append(v.Properties, s.propertyView(member))
		case *ast.MethodMember:
			v.Methods = append(v.Methods, s.methodView(member))
		}
	}
	return v
}

func (s *yamlScaffolder) functionView(decl *ast.FunctionDecl) FunctionView {
	list, joined := s.params(decl.Params)
	return FunctionView{
		Name:       decl.Name.Simple(),
		Params:     joined,
		ParamList:  list,
		ReturnType: s.types.Render(decl.Result),
		Intents:    Intents(decl.Intents),
	}
}

func (s *yamlScaffolder) methodView(m *ast.MethodMember) MethodView {
	list, joined := s.params(m.Params)
	return MethodView{
		Name:       m.Name.Name,
		Visibility: VisLower(m.Vis),
		Params:     joined,
		ParamList:  list,
		ReturnType: s.types.Render(m.ReturnType),
		Intents:    Intents(m.Intents),
	}
}

func (s *yamlScaffolder) propertyView(p *ast.PropertyMember) PropertyView {
	return PropertyView{
		Name:       p.Name.Name,
		Visibility: VisLower(p.Vis),
		Type:       s.types.Render(p.Type),
	}
}

// params builds the structured parameter list and its pre-joined rendering using
// the definition's param format (%[1]s = name, %[2]s = type) and separator.
func (s *yamlScaffolder) params(params []*ast.Parameter) ([]ParamView, string) {
	list := make([]ParamView, len(params))
	parts := make([]string, len(params))
	for i, p := range params {
		pv := ParamView{Name: ParamName(p, i), Type: s.types.Render(p.Type)}
		list[i] = pv
		parts[i] = apply(s.param, pv.Name, pv.Type)
	}
	return list, strings.Join(parts, s.paramSep)
}
