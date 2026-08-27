---
name: bhaus
description: >-
  Generate source code from BHaus design files (.bhaus). Update that code when
  the design changes. Use this skill when the user has a .bhaus file and wants
  working code from it. Example requests: "implement this bhaus file",
  "generate the Go code for this model", "scaffold code from my design", "the
  .bhaus changed, update the code". Use it even when the user does not name
  the tool. The goal is code, not .bhaus files. Run `bhaus-util scaffold`
  first. It makes the code shape. Then write the behaviour yourself.
---

# Generate and update code from BHaus models

A `.bhaus` file is a design document. It is not source code. It declares
three things:

- types,
- signatures,
- the intended behaviour of each method, written in prose.

Turn that design into working code in a target language.

Do not write the boilerplate by hand. The `bhaus-util scaffold` command maps
the model to the target language. It maps:

- the types,
- the visibility,
- the optionals,
- the unions,
- the method signatures.

If you write these by hand, your code will not match the tool's result. Let
the tool make the skeleton. Then do the one task that the tool cannot do:
write the behaviour for each method. The model describes this behaviour in a
functional-intent line.

Work in two steps:

1. Scaffold the shape with the tool.
2. Hydrate the behaviour yourself. ("Hydrate" means: replace a stub with real
   code.)

## Mental model

| Step | Actor | Result |
| --- | --- | --- |
| Scaffold | `bhaus-util scaffold` | Types, fields and method signatures. Each body is a stub. A stub is `panic("not implemented")` or `throw`. Each stub has a `// TODO: <intent>` comment. The tool copies this comment from the model's `>` line. |
| Hydrate | You | Real method bodies. Each body does what the intent says. Keep the scaffolded signature as the contract. |

The `> ...` lines in the model are the specification for hydration. See
`references/language-spec.md`, section _Functional intent_. A stub's `// TODO:`
comment holds this intent. Implement exactly this intent. Do not add more.

## Read the model itself for context

The scaffold output does not hold everything. The tool drops some parts of
the model. So read the raw `.bhaus` file too. It gives context that guides
your hydration.

Look for these things in the file:

- C4 declarations: `SYSTEM`, `CONTAINER`, `COMPONENT`, `CONNECTION`. The tool
  emits no code for them. They show how the parts fit together. A
  `CONNECTION` shows which component calls which. Use this to place your code
  and to wire the parts.
- Comments. A comment line starts with `#`. The tool drops comments. They
  often hold intent, constraints or notes that the signatures do not show.
- Types and paths. The `/`-separated path of a declaration shows how the
  author groups related types. This helps you name packages and place files.

Read the model first. Then let the scaffold give you the exact shape.

## Before you start: find the CLI and the languages

The `bhaus-util` binary is in one of three places:

- on `PATH`,
- built at `bhaus-util/bin/bhaus-util`,
- run with `go run ./cmd/bhaus-util` from the `bhaus-util/` directory.

Find it. Then list the target languages. The set is not fixed. Third-party
YAML files can add more languages.

```bash
bhaus-util scaffold --help    # shows the languages and the flags
```

Use the language that the user names. If the user names no language, ask. Do
not guess. Lint the model first. A broken model makes broken stubs.

```bash
bhaus-util lint <file.bhaus>
```

The `scaffold` command reads only the files that you pass. It combines their
declarations. It does not follow `INCLUDE`. Pass every file that you need.
You can use a glob such as `domain/*.bhaus`.

## Workflow A: Make new code (no code exists yet)

Use this workflow when no target-language code exists.

1. Lint the model.
2. Scaffold to disk with `--out-dir`. The tool writes one stub file for each
   declaration. The file tree matches the model namespaces.
   ```bash
   bhaus-util scaffold <language> --out-dir <dir> <file.bhaus>...
   ```
3. Hydrate each stub. Replace each `panic("not implemented")` or `throw` body
   with real code. The code must do what the `// TODO:` intent says. Keep the
   signatures.
4. Add what the language needs to build. This can be package headers,
   imports, or a small entrypoint. Then build the code to check it.

`--out-dir` is safe here. Nothing exists yet, so the tool overwrites nothing.
After you write real bodies, use Workflow B.

## Workflow B: Update code that exists (the model changed)

Use this workflow when target-language code exists and someone changed the
`.bhaus` file.

The danger: if you run the tool again with `--out-dir`, it makes fresh stubs.
These stubs overwrite the bodies that you wrote. Never do this.

Instead, scaffold to stdout. Leave out `--out-dir`. Then apply the changes by
hand:

1. Lint the changed model.
2. Scaffold to stdout. This shows the fresh skeleton and writes no file.
   ```bash
   bhaus-util scaffold <language> <file.bhaus>...
   ```
   Add `--name <QualifiedName>` to show only one declaration. Example:
   `--name Domain/Entity/User`. Use this when you know which declaration
   changed.
3. Read the stdout. It shows the current shape: the types, the fields and
   the signatures. See _Read the scaffold output_ below for the format.
4. Merge member by member into the files that exist. Compare the scaffold to
   the current code. Apply only the differences:
   - New field or new method → add the stub. Then hydrate it.
   - Changed signature (a new name, a new parameter, a removed parameter or
     a new return type) → change the signature in place. Adapt the current
     body to the new signature.
   - Removed member → remove it. If the body held real logic, ask the user
     first.
   - Unchanged member → do not touch it. Keep every body that you wrote.
5. Hydrate the new stubs (the `panic`/`TODO` ones). Do this the same way as
   Workflow A. Leave the unchanged implementations as they are.

Before you write, check one thing: will the change overwrite real code with
a stub? If yes, stop. That is a full regenerate, not a merge.

## Read the scaffold output (stdout format)

The tool writes stdout in one of two shapes.

**Many files.** The tool marks each declaration's code with a path line. You
can split the stream on this line:

```
// ==== User.go ====
type User struct { ... }

// ==== Test.go ====
type Test struct { ... }
```

The marker is always `// ==== <path> ====`. This is true for every target
language. The marker is the stream separator. It is not a comment in the
code. Split the stream on this marker. Each `<path>` is the file for that
declaration.

**One file.** The tool prints the code raw. It writes no marker. In update
mode you already know which file you merge into. So the missing path is
fine. Use `--name` to show one declaration when this is simpler.

## Hydrate the stubs

- The signature is the contract. Keep the scaffolded parameter list, return
  type, visibility and name. If a signature is wrong, fix the `.bhaus` model,
  not the code. Then scaffold again.
- Implement the intent. Do no more. The `// TODO:` comment is the method's `>`
  line. Build exactly that behaviour. Do not add scope.
- Remove the stub after you implement it. Delete the
  `panic("not implemented")` or `throw`. Delete the `// TODO:` line.
- Some types live outside the model. An `EXTERN` marks such a type. The tool
  emits a `// EXTERN ... (defined elsewhere)` stub for it. Reference this
  type. Do not redefine it.
- Use the C4 model and the comments for context. See _Read the model itself
  for context_ above.

## Scaffold flags (quick reference)

```
bhaus-util scaffold <language> [flags] <file>...
  --name <QualifiedName>   Scaffold only this declaration, e.g. Domain/Entity/User.
  --out-dir <dir>          Write files (the tree mirrors namespaces). Leave this out for updates.
  --template-dir <dir>     Load extra YAML language definitions.
```

Put the flags before the file list. Go's flag parser stops at the first
positional argument. A shell glob expands into that trailing list. So
`scaffold go --name X a.bhaus` works. `scaffold go a.bhaus --name X` does not
work.

## Language-specific behaviour

The workflow above works for every language. Some languages have their own
skill extension. Such an extension gives idiom rules for the hydrate step:
package layout, error handling, dependency injection and more. If an
extension exists for the target language, follow it for the hydrate step. If
no extension exists, use the normal idioms of the target language.

## Full language specification

Read `references/language-spec.md` for the model syntax. It covers `EXTERN`,
`Bits<N>`, optionals, unions, `EXTENDS`, `IMPLEMENTS`, `OVERRIDE`, functional
intent, the C4 model and the whitespace rules. It is the annotated grammar
reference. Read it when the linter reports something that you do not know or
when you meet an edge case.
