# BHaus util

`bhaus-util` is the toolset for the BHaus modelling language.

```bhaus
VERSION 0.1

STRUCT Domain/Entity/User:
    PUBLIC id: Integer
    PUBLIC email: String
    PUBLIC roles: Array[String]
    PUBLIC manager: ?Integer
```

It parses `.bhaus` files into a typed AST.
Every tool here is built on that AST: the editor language server, the linter and the code scaffolder.

The toolset is written in Go.
The pipeline has three stages:

1. tree-sitter builds a concrete syntax tree
2. the parser converts it to a typed AST
3. the analysis layer resolves references

## Quick Start

You need Go 1.27 or later.

1. Build the command-line tool:
   ```bash
   go build -o bin/bhaus-util ./cmd/bhaus-util
   ```
2. Check a file for problems:
   ```bash
   ./bin/bhaus-util lint bhaus/examples/user.bhaus
   ```
3. Generate code from a model:
   ```bash
   ./bin/bhaus-util scaffold go bhaus/examples/user.bhaus
   ```

Run `./bin/bhaus-util help` for the full command list. Run
`./bin/bhaus-util help <command>` for one command's flags and examples.

## Language

BHaus is a textual language for software design.
A `.bhaus` file declares types, signatures and architecture.
It is a design document, not source code.

The tools read these declarations:

- **Simple types** — `String`, `Integer`, `Float`, `Boolean`, `Bits<N>` and
  more.
- **Structural types** — `STRUCT`, `PROTOCOL` and `CLASS`.
- **Functions** — top-level `FUNCTION` and methods on a structural type.
- **C4 model** — `SYSTEM`, `CONTAINER`, `COMPONENT` and `CONNECTION`.

For the full grammar, read the language specification in the `bhaus/` directory.

## Modes

The toolset runs as five commands. Each command uses the same parser and AST.

### Language Server (`ls`)

The `ls` command starts the language server. It talks over stdio. An editor starts it, not a human.
It uses the [glsp](https://github.com/tliron/glsp) library for the LSP protocol.

The server gives these editor features:

- Go to definition.
- Find references.
- Document symbols and workspace symbols.
- Hover information.
- Completion.
- On-type formatting.
- Diagnostics from the linter.

Flags:

- `--log-file <path>` — write logs to this file.
- `--log-verbosity <int>` — set the log level. `0` is Notice, `1` is Info, `2`
  is Debug, and `-4` turns logging off.

```bash
./bin/bhaus-util ls --log-file /tmp/bhaus-util.log --log-verbosity 1
```

### Lint

The `lint` command checks a file for problems.
It also checks the files that the file includes.

```bash
./bin/bhaus-util lint [--format] <file>
```

The `--format` flag prints the file in a normal format.

### Scaffold

The `scaffold` command generates target-language source from a model.
It maps each declaration to types and method signatures.
Each method body is a stub. By default the tool streams the result to stdout.
With `--out-dir` it writes files instead.

```bash
# Print Go for one file to stdout
./bin/bhaus-util scaffold go bhaus/examples/user.bhaus

# Write TypeScript for every model into a directory
./bin/bhaus-util scaffold typescript --out-dir ./gen bhaus/examples/*.bhaus
```

The built-in languages are `go`, `typescript` and `php`.
You can add more with YAML definitions in the template directory.

### Skill

The `skill` command manages the agentic skill for BHaus.
The skill teaches an AI agent to generate and update code from `.bhaus` files.

```bash
# Install into a known agent's skill directory
./bin/bhaus-util skill install bhaus --agent claude

# Or into an explicit directory
./bin/bhaus-util skill install bhaus --dir ./.claude/skills
```

The skills are embedded in the binary, so installing one only copies files. It
needs no network access.

### Version

The `version` command prints the build identity: the release version, the commit
it was built from, the platform and the Go version. An unstamped local build
reports `dev`.

```bash
./bin/bhaus-util version
# bhaus-util 0.1.0 (a1b2c3d) darwin/arm64 go1.24.3
```

## Development

### Release

Package version exposes the build identity of the bhaus toolchain.

Version and Commit are overwritten at link time by the release build, so this
package must stay a stdlib-only leaf: both pkg/cli (for the `version`
subcommand) and pkg/lsp (for the LSP ServerInfo handshake) read it, which is
why the values do not live in package main — an -X flag aimed at main cannot
reach either of them.

```bash
go build -ldflags "-X github.com/sjsone/bhaus-util/pkg/version.Version=1.2.3 \
                   -X github.com/sjsone/bhaus-util/pkg/version.Commit=$(git rev-parse --short HEAD)"
```
