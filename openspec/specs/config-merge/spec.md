# config-merge Specification

## Purpose
Combines the parsed documents of every layer into a single configuration tree using recursive
deep-merge semantics where the highest-priority layer wins.

## Requirements

### Requirement: Recursive Map Merge
The system SHALL merge maps recursively, so keys present only in a lower-priority layer survive when
a higher-priority layer supplies a sibling key.

#### Scenario: Sibling keys are preserved
- GIVEN a base layer defining `db.host` and `db.port`
- WHEN an overlay defines only `db.port`
- THEN the result holds the base `db.host` alongside the overlay `db.port`

#### Scenario: Documents within one layer merge in order
- GIVEN a single layer holding two documents that define the same key
- WHEN the layer is merged
- THEN the later document's value wins

### Requirement: Replace Semantics For Lists And Scalars
The system SHALL replace, not merge, when both sides of a key are scalars or both are lists.

#### Scenario: List is replaced wholesale
- GIVEN a base layer defining `tags: [a, b]`
- WHEN an overlay defines `tags: [c]`
- THEN the result is `tags: [c]`, not `[a, b, c]`

### Requirement: Null Tombstone Deletion
The system SHALL treat an explicit `null` in a higher-priority layer as a deletion of that key.

#### Scenario: Overlay deletes a base key
- GIVEN a base layer defining `feature.beta`
- WHEN an overlay sets `feature.beta: null`
- THEN the key is absent from the resolved configuration

#### Scenario: Null base value is overwritten, not conflicting
- GIVEN a base layer defining a key as `null`
- WHEN an overlay defines that same key as a map
- THEN the overlay value is used and no conflict is raised

### Requirement: Type Conflict Detection
The system SHALL fail when the same key is a map in one layer and a non-map in another, reporting the
dotted path, both type names, and the contributing files.

#### Scenario: Map versus scalar at the same key
- GIVEN a base layer defining `db` as a map
- WHEN an overlay defines `db` as a string
- THEN it fails with a `MergeConflictError` naming the path `db`, the two types, and the overlay file

#### Scenario: Conflict deep in the tree
- GIVEN two layers whose type mismatch sits at `a.b.c`
- WHEN they are merged
- THEN the reported path is the full dotted path `a.b.c`
