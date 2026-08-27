# secret-redaction Specification

## Purpose
Turns the sentinels planted during templating back into either the literal `(sensitive)` placeholder
or the real secret value, and reports where secrets landed in the tree.

## Requirements

### Requirement: Dual Output
The system SHALL produce two resolved trees from the same merged input: a redacted one in which every
sentinel becomes `(sensitive)`, and a full one in which every sentinel becomes its real value.

#### Scenario: Redacted and full trees differ only at secrets
- GIVEN a configuration holding one secret and several plain values
- WHEN resolution completes
- THEN the two trees are identical except at the secret's path, where one holds `(sensitive)` and the other the real value

#### Scenario: Unresolvable sentinel
- GIVEN a sentinel with no entry in the sentinel map
- WHEN secrets are resolved
- THEN the sentinel string is left in place rather than crashing resolution

### Requirement: Whole-Value And Inline Substitution
The system SHALL replace a value that is exactly one sentinel as a whole, and SHALL replace each
sentinel occurrence in place when a sentinel is embedded in a longer string.

#### Scenario: Value is exactly a secret
- GIVEN a `password` value consisting of a single sentinel and nothing else
- WHEN secrets are redacted
- THEN the redacted value is exactly `(sensitive)`

#### Scenario: Secret embedded in a URL
- GIVEN a `url` value of `postgres://user:<sentinel>@host/db`
- WHEN secrets are resolved
- THEN the redacted value is `postgres://user:(sensitive)@host/db` and the full value carries the real password

#### Scenario: Multiple secrets in one string
- GIVEN a single string value holding two different sentinels
- WHEN secrets are resolved
- THEN each is substituted independently

### Requirement: Recursive Traversal
The system SHALL find sentinels at any depth, inside nested maps and inside list elements.

#### Scenario: Secret inside a list
- GIVEN a secret sitting as the third element of a list at `args`
- WHEN secrets are resolved
- THEN it is redacted in the redacted tree and resolved in the full tree

### Requirement: Secret Path Reporting
The system SHALL report the set of paths whose values contained a secret, using dot notation for map
keys and bracket notation for list indices.

#### Scenario: Nested map path
- GIVEN a secret at `db.credentials.password`
- WHEN resolution completes
- THEN that dotted path appears in the reported set

#### Scenario: List element path
- GIVEN a secret at index 2 of the list `args`
- WHEN resolution completes
- THEN the reported path is `args[2]`

#### Scenario: No secrets used
- GIVEN a configuration where no layer calls the `secret` function
- WHEN resolution completes
- THEN the reported set of secret paths is empty
