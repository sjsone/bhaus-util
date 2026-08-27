package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sjsone/bhaus-util/pkg/ast"
	"github.com/sjsone/bhaus-util/pkg/parser"
	"github.com/sjsone/bhaus-util/pkg/scaffold"
	_ "github.com/sjsone/bhaus-util/pkg/scaffold/language" // registers the go, typescript and php scaffolders
)

// handleScaffold parses one or more .bhaus files and prints rough target-language
// code to stdout (or writes files with --out-dir). It combines declarations
// from all input files. It returns the process exit code.
//
// Flags MUST precede the file list: scaffold <language> [flags] <file>...
// A shell glob expands to the trailing arguments. Go's flag package stops
// parsing at the first positional argument. Putting flags first lets both
// work together.
func handleScaffold(args []string) int {
	if len(args) < 1 || isHelpFlag(args[0]) {
		out := os.Stderr
		code := 1
		if len(args) >= 1 && isHelpFlag(args[0]) {
			out, code = os.Stdout, 0
		}
		scaffoldUsage(out)
		return code
	}
	language := args[0]

	fs := flag.NewFlagSet("scaffold", flag.ExitOnError)
	fs.Usage = func() { scaffoldUsage(os.Stderr) }
	name := fs.String("name", "", "only scaffold the declaration with this qualified name")
	templateDir := fs.String("template-dir", defaultTemplateDir(), "directory of *.yaml language definitions")
	outDir := fs.String("out-dir", "", "write generated files here (mirrors namespaces); default: stream to stdout")
	fs.Parse(args[1:])

	filePaths := fs.Args()
	if len(filePaths) == 0 {
		fmt.Fprintln(os.Stderr, "bhaus-util: scaffold requires at least one .bhaus file (flags must come before files)")
		return 1
	}
	// A flag-looking positional means the flags came after the file list. The
	// flag parser cannot see flags in that position. Give a clear message
	// instead of a confusing "no such file" error.
	for _, p := range filePaths {
		if strings.HasPrefix(p, "-") {
			fmt.Fprintf(os.Stderr, "bhaus-util: %q looks like a flag but appears after a file; put all flags before the file list\n", p)
			return 1
		}
	}

	// Register any third-party YAML language definitions before resolving.
	externals, err := scaffold.LoadDir(*templateDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bhaus-util: loading template dir: %v\n", err)
		return 1
	}
	for _, s := range externals {
		scaffold.Register(s)
	}

	scaffolder, err := scaffold.Get(language)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bhaus-util: %v\n", err)
		return 1
	}

	// Parse each input file. Combine their declarations.
	var allDecls []ast.Decl
	for _, filePath := range filePaths {
		content, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "bhaus-util: reading %s: %v\n", filePath, err)
			return 1
		}
		file, err := parser.Parse("file://"+filePath, string(content))
		if err != nil {
			fmt.Fprintf(os.Stderr, "bhaus-util: parsing %s: %v\n", filePath, err)
			return 1
		}
		allDecls = append(allDecls, file.Decls...)
	}

	decls := scaffold.FilterByName(allDecls, *name)
	files, err := scaffolder.Scaffold(decls)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bhaus-util: scaffolding: %v\n", err)
		return 1
	}

	if *outDir != "" {
		if err := writeFiles(*outDir, files); err != nil {
			fmt.Fprintf(os.Stderr, "bhaus-util: writing files: %v\n", err)
			return 1
		}
		return 0
	}
	streamFiles(os.Stdout, files)
	return 0
}

// scaffoldUsage writes the scaffold command's detailed help to w. It queries
// the available language list at call time. This way, third-party YAML
// definitions that were loaded into the registry appear here too.
func scaffoldUsage(w io.Writer) {
	fmt.Fprintf(w, `bhaus-util scaffold — generate rough target-language source from a model

Usage:
  bhaus-util scaffold <language> [flags] <file>...

Parses every input .bhaus file, combines their declarations and emits source in
the chosen language. By default the result is streamed to stdout; with --out-dir
it is written to files whose paths mirror the declaration namespaces.

Arguments:
  <language>   Target language. One of: %v
               (plus any YAML defs found in --template-dir).
  <file>...    One or more .bhaus files. A shell glob such as domain/*.bhaus works.

Flags (MUST come before the file list — see note below):
  --name <QualifiedName>   Only scaffold the declaration with this name, e.g. Domain/User.
  --template-dir <dir>     Directory of *.yaml third-party language definitions.
                           Default: <user config dir>/bhaus/scaffold.
  --out-dir <dir>          Write generated files here (mirrors namespaces).
                           Default: stream to stdout.

Flag ordering: flags must precede the files. Go's flag parser stops at the first
non-flag argument and a shell glob expands into that trailing file list — so
"scaffold go --name X a.bhaus" works, but "scaffold go a.bhaus --name X" does not.

Examples:
  # Print Go for one file to stdout
  bhaus-util scaffold go domain/user.bhaus

  # Only the Domain/User declaration, as TypeScript
  bhaus-util scaffold typescript --name Domain/User domain/user.bhaus

  # Every model into ./gen, one file per namespace
  bhaus-util scaffold php --out-dir ./gen domain/*.bhaus
`, scaffold.Available())
}

// writeFiles writes each generated file under outDir. It creates parent
// directories as needed. It reports each path written to stderr.
func writeFiles(outDir string, files []scaffold.GeneratedFile) error {
	for _, f := range files {
		dest := filepath.Join(outDir, f.Path)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, []byte(f.Content), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "wrote %s\n", dest)
	}
	return nil
}

// streamFiles prints files to w. It prints a single file raw. For multiple
// files, it adds a "// ==== <path> ====" marker before each one, so the
// stream is splittable.
func streamFiles(w io.Writer, files []scaffold.GeneratedFile) {
	if len(files) == 1 {
		io.WriteString(w, files[0].Content)
		return
	}
	for _, f := range files {
		fmt.Fprintf(w, "// ==== %s ====\n%s\n", f.Path, f.Content)
	}
}

// defaultTemplateDir returns the directory that holds third-party YAML
// language definitions.
func defaultTemplateDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "bhaus", "scaffold")
}
