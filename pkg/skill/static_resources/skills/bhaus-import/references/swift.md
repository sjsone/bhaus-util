# Importing Swift

## Mapping notes

- **Module** → path prefix when the design spans several modules
  (`Core/User`), omitted for a single-module design. A Swift type has no
  namespace; the module name is the only grouping there is.
- **`struct`** → `STRUCT`, **`class`** → `CLASS`, **`protocol`** →
  `PROTOCOL`, **`enum`** → `EXTERN` + a comment listing the cases. Enums with
  associated values carry data — note the payload in the comment, since the
  enum itself cannot be declared.
- **Conformances**: keep protocols the *domain* defines (`UserRepository`).
  Drop framework and compiler protocols (`Codable`, `Identifiable`,
  `Sendable`, `Hashable`, `Equatable`, `CaseIterable`, `RawRepresentable`).
  If a dropped conformance matters to the design (e.g. "the type is
  serialized"), say it in a comment.
- **Visibility**: `public`/`open` → `PUBLIC`; `internal` → `PUBLIC` (the
  module is the design boundary); `fileprivate`/`private` → `PRIVATE`.
- **Optionals** → `?T`: `String?` → `?String`, `User?` → `?User`.
- **Arrays** → `Array[T]`. **Dictionaries** → `EXTERN` or `Array[Struct]`.
  **`Any`** → the concrete types from the call sites, else `EXTERN Any`.
- **`async` / `throws`** drop. A `throws` call becomes a `?T` return or a
  `Boolean`-returning operation, with the failure in the intent.
- **`mutating`** drops. **`inout` parameters** become ordinary parameters.
  **Default argument values** drop — the signature keeps the types.
- **Accessors** (`var foo: Int { get set }`, `private(set)`) → properties.
  Computed properties with logic → keep as a property with a comment, or as a
  method if the logic is behaviour.
- **`init`** → drop when it only assigns stored properties. An `init` with
  logic keeps its purpose as a comment or factory intent.
- **Foundation/SwiftUI/Combine types** — `UUID`, `Date`, `URL`, `Data`,
  `ObservableObject`, `@Published` etc. → `EXTERN` with a `#` comment.
  Property wrappers (`@State`, `@Published`) drop; the wrapped property stays.

## Worked example

Source, `Sources/UserService/User.swift` and `UserRepository.swift`:

```swift
import Foundation

public struct User: Identifiable, Codable {
    public let id: UUID
    public var name: String
    public var email: String?
    public let roles: [String]
    private var accessToken: String?

    public mutating func addRole(_ role: String) {
        if !roles.contains(role) {
            roles.append(role)
        }
    }

    public func hasRole(_ role: String) -> Bool {
        roles.contains(role)
    }
}

public protocol UserRepository: Sendable {
    func find(by id: UUID) async throws -> User?
    func save(_ user: User) async -> Bool
}
```

Result, `design/user.bhaus` and `design/userRepository.bhaus`:

```bhaus
# user.bhaus
VERSION 0.1

# platform identifier type
EXTERN UUID

STRUCT User:
    PUBLIC id: UUID
    PUBLIC name: String
    PUBLIC email: ?String
    PUBLIC roles: Array[String]
    PRIVATE accessToken: ?String
    PUBLIC addRole(String)
        > adds the role when it is not already present
    PUBLIC hasRole(String): Boolean
        > returns whether the user has the given role
```

```bhaus
# userRepository.bhaus
VERSION 0.1
INCLUDE user.bhaus

PROTOCOL UserRepository:
    PUBLIC find(by: UUID): ?User
        > returns the user with the given identifier
        > throws or returns null when no user matches
    PUBLIC save(User): Boolean
        > persists the user
        > returns whether the save succeeded
```

Notes on the result:

- `Identifiable` and `Codable` drop — framework protocols. The stored property
  `id: UUID` stays, which is the real design content.
- `async throws` on `find` became `?User` with the failure spelled out in the
  intent. `Sendable` drops.
- `accessToken` is `private var` → `PRIVATE` property, kept because the user
  carries it; `mutating` disappears.
- Two files, `INCLUDE user.bhaus` links them: `UserRepository` references
  `User` from the sibling file, and the linter resolves it through the
  include. The include names the file explicitly — a glob is only for an
  open-ended set of files.
- `EXTERN UUID` appears only in `user.bhaus`. Declaring it in both files would
  make the linter report `duplicate-decl` and ambiguous references, because
  `INCLUDE` joins the two externs into one fileset. Rule: each `EXTERN` is
  declared once, in the file that owns it; every other file sees it through
  `INCLUDE`.
- Extern comments sit **above** their `EXTERN` line, like a doc comment.
- The label `by` in `find(by: UUID)` is kept as a named parameter — it adds
  meaning to the design.
