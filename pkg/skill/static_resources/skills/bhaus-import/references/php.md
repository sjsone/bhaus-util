# Importing PHP

## Mapping notes

- **Namespace** → path prefix, vendor root dropped: `App\Domain\User\User` →
  `Domain/User/User`. The full namespace keeps its segments; the type name is
  the final segment.
- **`final class` / `abstract class`** → `CLASS`; finality and abstraction are
  implementation markers. `abstract class` additionally means "do not
  instantiate" — if the design needs that, say it in a comment.
- **`interface`** → `PROTOCOL`.
- **Visibility**: `public`/`protected`/`private` map 1:1. Property types
  without a visibility keyword (rare, from `var`) → `PRIVATE`.
- **Nullables** → `?T`. **`array`** → `Array[T]`; the element type comes from
  the docblock (`/** @var string[] */`), the property default, or the call
  sites — never from the bare `array` keyword alone. If no element type can be
  found, `EXTERN array` with a comment rather than guessing.
- **Union types** (`string|int`) → binary unions only: `String | Integer`.
  A union of three or more alternatives does not fit — pick the two that carry
  the design meaning and `EXTERN` the rest, or keep the whole thing external.
  `mixed` → name the actual types from the call sites; avoid `Unknown` when
  the code shows what is passed.
- **`enum` (PHP 8.1)** → `EXTERN` + a comment listing the cases.
- **`__construct`** → drop when it only assigns fields. A constructor with
  validation or derived defaults keeps its purpose as a comment on the class
  or as a `FUNCTION` factory with an intent.
- **Static methods** → top-level `FUNCTION` with the class path prefix:
  `Domain/User/User/create(...)`.
- **Frameworks** (Laravel, Symfony, ...) — `Request`, `Response`, `Model`
  bases, facades, container bindings → `EXTERN` with a `#` comment saying what
  each is. Eloquent/Doctrine `Model` base classes drop; the design keeps the
  domain shape, not the ORM plumbing.

## Worked example

Source, `src/Domain/User/User.php`:

```php
<?php

namespace App\Domain\User;

use App\Domain\User\ValueObject\UserId;

final class User
{
    private UserId $id;
    private string $name;
    /** @var string[] */
    private array $roles = [];
    private ?string $email = null;

    public function __construct(UserId $id, string $name, array $roles = [], ?string $email = null)
    {
        $this->id = $id;
        $this->name = $name;
        $this->roles = $roles;
        $this->email = $email;
    }

    public function addRole(string $role): void
    {
        if (in_array($role, $this->roles, true)) {
            return;
        }
        $this->roles[] = $role;
    }

    public function hasRole(string $role): bool
    {
        return in_array($role, $this->roles, true);
    }
}
```

Result, `design/user.bhaus`:

```bhaus
VERSION 0.1

# value object wrapping the user identifier
EXTERN Domain/User/UserId

CLASS Domain/User/User:
    PRIVATE id: Domain/User/UserId
    PRIVATE name: String
    PRIVATE roles: Array[String]
    PRIVATE email: ?String
    PUBLIC addRole(String)
        > adds the role when it is not already present
    PUBLIC hasRole(String): Boolean
        > returns whether the user has the given role
```

Notes on the result:

- The `App\` vendor prefix is dropped; the class is `Domain/User/User`, the
  faithful namespace-plus-name.
- `UserId` is a value object from another namespace. When it is not part of
  this design's scope it becomes `EXTERN` with a comment. If it is in scope,
  define it in its own file and `INCLUDE` it instead.
- `__construct` only assigns fields → gone. The fields remain.
- The `final` keyword, defaults and the docblock are implementation details.
  `@var string[]` did its job: it supplied the element type `String` for the
  array property.
- Bodies became intents; note how `addRole`'s guard (`in_array` early return)
  is captured as "when it is not already present" — what, not how.
