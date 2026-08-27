# Importing Go

## Mapping notes

- **Package** → path prefix. Use the package name, not the full import path
  (`github.com/acme/shop/internal/model` → `Model/`). If packages with the
  same name collide in the design, disambiguate with one directory segment of
  the import path.
- **`struct`** → `STRUCT`. A struct holding only unexported fields with no
  exported members and no methods is likely an implementation detail — drop it
  or make it `PRIVATE`-membered if the design needs the type.
- **`interface`** → `PROTOCOL`. Go interfaces are often small; keep them.
  The concrete implementing types: keep a `CLASS` only when the design needs
  to name the implementation (e.g. multiple implementations, or a repository
  bound to a real store). A single anonymous impl can be dropped.
- **Visibility**: exported → `PUBLIC`, unexported → `PRIVATE`. Go has no
  `protected`.
- **Pointers** → `?T`: `*User` → `?Model/User`. A value return that can be
  nil and a `(T, error)` pair both become `?T` with the failure written into
  the intent line.
- **Slices** → `Array[T]`. **Maps** → `Array[KvStruct]` when the value
  matters, else `EXTERN`. **`error`** never becomes a type: fold it into the
  intent or return `Boolean` for operations that succeed or fail.
- **Typed constants as enums** (`type Status int` + `const` block) → `EXTERN`
  the type + a comment listing the values.
- **`func (r *repo) Method`** → method; **package-level funcs** → `FUNCTION`.
  Constructors (`NewXxx`) are usually just struct wiring → drop them; if a
  constructor does real work, keep its purpose as a comment.
- **`context.Context`, `time.Time`, `io.Reader` and friends** → `EXTERN` with
  a `#` comment.
- **`interface{}` / `any`** → ask what the code actually passes. If one
  concrete type appears at every call site, use it. If genuinely mixed, use
  `EXTERN any` or the two-member union that the call sites show. Avoid
  `Unknown` when the code reveals a real type.
- **Role comments** (`// User is an account holder`) → decide deliberately.
  Promote to a `PROTOCOL` only when the role carries its own members; a label
  alone stays a comment, with the choice stated:

  ```bhaus
  # The source calls User an account holder. The role carries no members of
  # its own, so it stays a comment rather than a PROTOCOL.
  STRUCT Model/User:
      ...
  ```

## Worked example

Source, `model/model.go`:

```go
package model

type Status int

const (
	StatusActive Status = iota
	StatusSuspended
)

type User struct {
	ID     int64
	Name   string
	Email  *string
	Roles  []string
	Status Status
	Config map[string]string
}

type Profile struct {
	Bio       string
	AvatarURL string
}

type Repository interface {
	FindByID(id int64) (*User, error)
	Save(u *User) (bool, error)
}

type repo struct{}

func (r *repo) FindByID(id int64) (*User, error) {
	if id < 1 {
		return nil, errors.New("invalid id")
	}
	return store.get(id), nil
}

func (r *repo) Save(u *User) (bool, error) {
	return store.put(u), nil
}
```

Result, `design/model.bhaus`:

```bhaus
VERSION 0.1

# values: active, suspended
EXTERN Model/Status

# the user's settings map; kept external because BHaus has no map type
EXTERN Model/UserConfig

STRUCT Model/User:
    PUBLIC id: Integer
    PUBLIC name: String
    PUBLIC email: ?String
    PUBLIC roles: Array[String]
    PUBLIC status: Model/Status
    PUBLIC config: Array[Model/UserConfig]

STRUCT Model/Profile:
    PUBLIC bio: String
    PUBLIC avatarUrl: String

PROTOCOL Model/Repository:
    PUBLIC findById(Integer): ?Model/User
        > returns the user with the given id
        > returns null when no user matches or the id is invalid
    PUBLIC save(Model/User): Boolean
        > persists the user
        > returns whether the save succeeded
```

Notes on the result:

- `Status` is an enum → `EXTERN` + values comment. `Config` is a map →
  external, with a comment explaining why.
- `(T, error)` and nil returns collapse to `?T`; the error case is spelled out
  in the intent (`returns null when no user matches or the id is invalid`).
- The unexported `repo` struct is dropped: `Model/Repository` names the
  capability, and the design has one implementation.
- The anonymous `store` is outside the design; the intents describe what
  happens, not how.
