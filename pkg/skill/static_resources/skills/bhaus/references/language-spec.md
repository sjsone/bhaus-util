# BHaus language specification (reference)

The complete syntax. Read this when the cheat-sheet in `SKILL.md` doesn't cover
what you need. Contents:

- [Comments](#comments)
- [Version](#version)
- [Include](#include)
- [Simple types](#simple-types)
- [Type modifiers: optional and union](#type-modifiers)
- [Group types: Array](#group-types)
- [Names, paths, and EXTERN](#names-paths-and-extern)
- [Functions and methods](#functions-and-methods)
- [Structural declarations: STRUCT, PROTOCOL, CLASS](#structural-declarations)
- [Functional intent](#functional-intent)
- [C4 model](#c4-model)

## Comments

Single-line only. A comment MUST start with `#` followed by a space. Light
markdown is allowed inside: `_text_`, `*text*`, `**text**`, `***text***`,
`` `code` ``. Consecutive comment lines with no blank line between them MAY be
treated as one multi-line comment.

## Version

The first meaningful statement MUST be the version: the keyword `VERSION`,
whitespace, then a SemVer version designator.

```bhaus
VERSION 0.1
```

## Include

`INCLUDE <glob>` pulls in other `.bhaus` files so their declarations can be
referenced. The glob is relative to the current file. `INCLUDE *` includes
sibling files.

## Simple types

A simple type is conceptually a single value. Some have short aliases.

| Type | Alias | Meaning |
| --- | --- | --- |
| `Character` | `Char` | one character of an alphabet |
| `String` | | a list of characters |
| `Integer` | `Int` | signed integer, platform-largest by default |
| `UnsignedInteger` | `UInt` | unsigned integer |
| `Float` | | signed floating point, platform-largest by default |
| `UnsignedFloat` | `UFloat` | unsigned floating point |
| `Boolean` | `Bool` | `True` or `False` |
| `Bits<N>` | | fixed-width bit sequence, e.g. `Bits<8>` |

## Type modifiers

- **Optional** — prefix any type with `?` to signal it may be empty / not yet
  initialised: `?Integer`, `?Domain/Entity/User`, `?Array[String]`.
- **Union** — combine alternatives with `|`: `String | Integer | Boolean`.

## Group types

- `Array[Type]` — the `Array` keyword, `[`, the element `Type`, then `]`.
  Nests infinitely: `Array[Array[Array[String]]]`.

Future group types (`Set`, `Tuple`, ...) may follow the same bracket form.

## Names, paths, and EXTERN

A declaration has a **contextual name**: either a single word (`User`) or a
`/`-separated path (`Domain/Entity/User`). Paths namespace related
declarations; they are logical, not tied to the filesystem.

`EXTERN <ContextualName>` declares that a name exists without defining it. Use
it to reference a type that lives outside the model (a platform type, a type
defined elsewhere you don't want to import) so `unresolved-ref` stays quiet.

## Functions and methods

**Free function** (top level): `FUNCTION` (or `FUNC`), whitespace, contextual
name, `(`, optional comma-separated parameter types, `)`. With a return type,
follow `)` with `:`, whitespace, and the return type/name.

```bhaus
FUNCTION hash(String): Bits<256>
```

**Method** (member of a STRUCT/PROTOCOL/CLASS): identical, but the `FUNCTION`
keyword is **omitted** — the parameter list is what marks it as a method.

Function and method names SHOULD start with a lowercase letter (the `naming`
lint check).

## Structural declarations

Each is: keyword, whitespace, contextual name, `:`. Members are indented (one or
more whitespace). Each member begins with a visibility keyword — `PUBLIC`,
`PRIVATE`, or `PROTECTED` — then its name, then its type. Structural names
SHOULD start with an uppercase letter.

### PROTOCOL

An interface — capabilities, no implementation. Members are properties and
methods.

```bhaus
PROTOCOL Base/Entity:
    PUBLIC getIdentifier(): UUID
    PUBLIC value: String
```

### STRUCT

One-dimensional pairing of named keys with types; plain data. Nests infinitely.

```bhaus
STRUCT UUID:
    PRIVATE raw: String
```

### CLASS

A concrete type. Optional `EXTENDS <ClassName>` (single inheritance) and/or
`IMPLEMENTS <ProtocolName>`. When overriding a method inherited from a parent
class, prefix the member with `OVERRIDE`, placed **before** the visibility
keyword (`OVERRIDE PUBLIC ...`) — the parent may not be part of this design
document, so the override must be explicit.

```bhaus
CLASS Model EXTENDS Base IMPLEMENTS Base/Entity:
    PUBLIC test: Base/Bla
    OVERRIDE PUBLIC getIdentifier(): UUID
```

## Functional intent

A method (in any structural declaration) may be followed by an indented line
starting with `>` that describes its intended behaviour in prose. It documents
intent; it is not executable.

```bhaus
    PUBLIC getIdentifier(): UUID
        > returns current or creates new
```

## C4 model

Four nesting levels for architecture. Each of `SYSTEM`, `CONTAINER`,
`COMPONENT` is: keyword, whitespace, contextual name, an optional
`"description"` in double quotes, then `:`.

- `SYSTEM` — a high-level software system; contains `CONTAINER`s.
- `CONTAINER` — a deployable unit within a system; contains `COMPONENT`s.
- `COMPONENT` — a modular part within a container.

```bhaus
SYSTEM MailSystem "E-Mail Backend":
    CONTAINER MTA "Mail Transfer Agent":
        COMPONENT SmtpServer:
        COMPONENT QueueManager:
    CONTAINER Database:
```

### CONNECTION

A relationship between two elements: keyword, source path, `->`
(unidirectional) or `<->` (bidirectional), target path. Paths use **dot**
notation: `SystemName.ContainerName.ComponentName`.

```bhaus
CONNECTION Webmail.Backend -> MailSystem.MTA
CONNECTION Webmail.Frontend <-> Webmail.Backend
```

**Shorthand**: inside a `SYSTEM`/`CONTAINER`/`COMPONENT` block, the source path
MAY be omitted — the source is implicitly the enclosing element:

```bhaus
CONTAINER Backend:
    CONNECTION => MailSystem.MTA   # equivalent to: CONNECTION Webmail.Backend -> MailSystem.MTA
```

The shorthand MUST NOT appear at the top level — there is no enclosing element
to supply the source.
