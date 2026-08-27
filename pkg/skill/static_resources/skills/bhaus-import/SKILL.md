---
name: bhaus-import
description: >-
  Convert existing source code into .bhaus design documents. This is the
  reverse of the bhaus skill, which generates code from designs. Use this skill
  when the user has code and wants a .bhaus design for it, even if the user
  does not say "bhaus". Example requests: "convert this Go code to a design",
  "write a .bhaus design for my PHP project", "import my Swift types into
  bhaus", "reverse engineer this codebase", "document my architecture", "make
  a bhaus model of our services", "I need a design doc for this code". The
  goal is .bhaus files made from code. It is never code made from .bhaus
  files.
---

# Convert source code to BHaus designs

A `.bhaus` file is a design document. It is not a copy of the code.
An import extracts three things from the code:

- the types,
- the signatures,
- the intended behaviour.

It drops the rest. This skill turns source code into `.bhaus` files that
describe the same system. It works for any language.

## Why import instead of writing the design by hand

- The code shows the true system. Read the code to get exact type names,
  signatures, and relations. A design written from memory risks inventing a
  system that the code does not match.
- The `bhaus` skill regenerates code from a design. `bhaus-util scaffold`
  builds the skeleton. A code generator fills the bodies from the `>` intent
  lines. So a good import produces a design from which the original code can
  be regenerated.
- Intent lines are the heart of the import. Everything else is shape.

## What you produce

Write one `.bhaus` file for each source module or package. Write the files
into a `design/` directory that mirrors the source tree. A module is the unit
that the source language names: a Go package, a PHP namespace, or a Swift
module.

- Name the file after the module, in lower case: `model.bhaus`, `user.bhaus`.
- Build each type's contextual name from the source namespace path plus the
  type name.
  - Go package `model`, type `User` → `Model/User`.
  - PHP namespace `App\Domain\User`, class `User` → `Domain/User/User`. Drop
    the vendor prefix `App\`.
  - Swift type `User` with no module → `User`.
- Resolve cross-file references with `INCLUDE`. Name the file when you know
  it: `INCLUDE user.bhaus`. Use a glob (`INCLUDE *`) only when the set is
  open, for example when more files may join the directory later. The glob is
  relative to the current file.

## The import workflow

Follow these steps in order.

1. Set the scope before you read code. Default scope: everything the user
   names. Drop tests, generated code, vendored code, third-party code, build
   files, and config files. A design document is a selective view. If the
   user names nothing specific, ask. Or state your scope and let the user
   correct it.
2. Read whole source files. Do not read snippets. For each type, read its
   callers and importers too. Relations that span files belong in the design:
   who implements an interface, who constructs what.
3. Map the types. Use the mapping table below. Then use the per-language
   guide in `references/` when one exists for the source language.
4. Write intent lines from each method body. See _Intent lines_. This step
   gives the import its value.
5. Check source comments for roles. A comment such as `// User is an account
   holder` states a role. Apply the _Roles and protocols_ rule. Do not drop a
   role that the source states.
6. Lay out the files. Split by module. Add `VERSION`, `INCLUDE`, and `EXTERN`
   as needed. Write the declarations.
7. Verify with the linter. Run `bhaus-util lint` on every file you wrote. Fix
   what it reports. See _Verify_.

## Mapping source concepts to BHaus

| Source concept | BHaus form |
| --- | --- |
| namespace / package / module | `/`-separated path prefix. Drop vendor prefixes. |
| struct, record, data class | `STRUCT` |
| interface / protocol | `PROTOCOL` |
| class | `CLASS` |
| free function | `FUNCTION` at the top level |
| method | method. No keyword. Write visibility + name + parentheses. |
| single inheritance | `EXTENDS` |
| implements / conformance | `IMPLEMENTS`. Keep domain protocols only. Drop framework protocols. |
| override | `OVERRIDE` before the visibility keyword |
| nullable / optional / pointer / nil | leading `?`: `?String` |
| array / list / vector / slice | `Array[T]` |
| map / dictionary | no BHaus type. Model it as `Array[Struct]` or `EXTERN` it. |
| enum | no BHaus type. Write `EXTERN` for the type. Put a comment above it that lists the values. |
| generics | no BHaus generics. Use the concrete type when the code binds one. Use `EXTERN` when it does not. |
| static method | top-level `FUNCTION` with the class path as prefix |
| `(T, error)` returns | `?T`. Put the failure cases in the intent line. |
| void / unit / no return | no return type on the declaration |
| boolean / integer / float / string | `Boolean` / `Integer` / `Float` / `String`. Integer widths collapse. |

Map visibility directly: `public` → `PUBLIC`, `private` → `PRIVATE`,
`protected` → `PROTECTED`. For languages without these keywords: Go exported →
`PUBLIC`; Go unexported → `PRIVATE`; Swift `internal` → `PUBLIC`, because
module-visible means in-design. Drop language modifiers that say nothing about
design shape: `final`, `async`, `throws`, `mutating`, `readonly`. Keep type
and method names as the source spells them. Note the naming convention: type
names start uppercase. Function and method names start lowercase. If the
source breaks the convention, rename at the boundary of the design and note
the rename in a comment.

## Intent lines

An intent line is an indented `> ...` line under a method or function. It
describes the intended behaviour in prose. Imported intents are the contract
for regeneration: `scaffold` copies them into `// TODO:` stubs and the `bhaus`
skill implements them. A reader must be able to reimplement the body from the
intent alone.

- Read the whole body. Write one to three lines that say **what** the method
  does, not how: `> returns the user with the given id`, `> adds the role when
  it is not already present`, `> returns null when no user matches`.
- Fold failure cases into the intents. An error return in the source becomes
  `> returns null when ...` or `> returns whether ... succeeded`. Do not write
  `Unknown` or a fake type for it.
- A trivial body, such as a field accessor, gets a short intent or none.
- Never paste code into an intent line. If the body is too complex for a
  short intent, the behaviour needs a comment, not a longer intent.

## Resolution discipline: EXTERN vs INCLUDE

The linter reports an error when a reference resolves to nothing or to more
than one declaration. Every type you mention must resolve to exactly one
declaration: your declaration, an included file's declaration, or an
`EXTERN`.

- Types you define → declare them in a `.bhaus` file.
- Types defined in another file of the same design → `INCLUDE` that file.
- Types you deliberately do not design (framework, platform, or third-party
  types, such as `UUID`, HTTP requests and responses, database handles,
  dependency containers) → write one `EXTERN <Name>` line for each. Put a `#`
  comment **above** the `EXTERN` line that says what the type is. The line
  below the extern belongs to the next declaration. This is the normal case
  for framework-heavy code.
- Declare each `EXTERN` once, in the file that owns it. `INCLUDE` joins the
  files into one fileset. So a second `EXTERN UUID` in an included file causes
  `duplicate-decl` and makes every `UUID` reference ambiguous. Other files see
  the extern through `INCLUDE`.
- Enums and generics → follow the mapping table. Always add a comment that
  says what was left out.

## Roles and protocols

A source comment or name sometimes states a role: `// User is an account
holder`, `acts as a connector`. A role is a design idea that the code may not
name. Decide, and show the decision in the design.

- Promote the role to a `PROTOCOL` when the role has members of its own.
  Members can be: capabilities used through a shared interface, fields shared
  by several types, or operations that belong to the role, not to one type.
  The concrete type then declares `IMPLEMENTS`. A type that implements must be
  a `CLASS`. So a promoted role can turn a `STRUCT` into a `CLASS`. Code that
  already shows the role as an interface or a union is a strong signal to
  promote.
- Keep the role as a comment when it is only a label with no capabilities
  beyond the type's own members. Say so. Do not copy the source comment
  unchanged: `# The source calls User an account holder; the role carries no
  members of its own, so it stays a comment.`

Do not invent an empty `PROTOCOL` to make a label look structural. Do not
drop a role that the source states.

## What to leave out

The design shows the shape and the intent. It does not show everything.

- Method bodies. They become intent lines, never code.
- Tests, generated code, vendored code, build files, and config files.
- Constructors that only assign fields. Keep the fields. Drop the
  constructor. A constructor that does real work (validation, invariants,
  defaults) keeps that purpose as a comment on the type or as an intent on a
  factory function.
- Private helpers. Keep a helper that encodes a domain rule. Drop a helper
  that only moves data.
- Framework protocol conformances: `Codable`, `Sendable`, `Serializable`, and
  similar. Keep protocols that the domain defines.
- Implementation markers: `final`, `async`, `throws`, `mutating`, static
  keywords, singletons.

If you cannot tell the purpose of a type or a member, ask the user. Do not
guess.

## C4 model (optional)

The language supports the C4 model (`SYSTEM`, `CONTAINER`, `COMPONENT`,
`CONNECTION`) for architecture-level views. Produce it only when two things
are true: the code shows clear boundaries (an app with controllers, services,
and repositories, or modules with real call graphs) and the user wants
architecture in the design. Derive connections from imports and real call
sites. Never guess a connection that the code does not show.

## Verify

`bhaus-util lint <file.bhaus>` checks each file and follows its `INCLUDE`s.
Run it on every file you wrote. Find the binary in one of three places:

- on `PATH`,
- built at `bhaus-util/bin/bhaus-util`,
- run with `go run ./cmd/bhaus-util` from the `bhaus-util/` directory.

The command is the same in all three cases.

- Errors (`syntax`, `unresolved-ref`, `duplicate-decl`) must be zero. One
  error means the design is wrong.
- Warnings are conventions, not failures. Still, fix them. `naming` (type
  names uppercase, function and method names lowercase) is part of the design
  contract. `structure` (`VERSION` first) is easy to satisfy. `unknown-type`
  means you wrote `Unknown` where a real type exists.

## Language-specific guides

The mapping table covers any language. For Go, PHP, and Swift, the
`references/` directory holds a worked example (source before, design after)
plus the language details:

- `references/go.md`
- `references/php.md`
- `references/swift.md`

For other languages, follow the table and the workflow. The three guides show
the expected depth of the result.

## Full language specification

`references/language-spec.md` is the annotated grammar reference. Read it when
the linter reports something you do not know. Read it for edge cases: unions,
`Bits<N>`, nested arrays, connection shorthand.
