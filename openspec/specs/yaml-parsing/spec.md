# yaml-parsing Specification

## Purpose
Parses each rendered layer into one or more YAML documents normalized to string-keyed maps that the
merge stage can consume.

## Requirements

### Requirement: Multi-Document Parsing
The system SHALL parse a layer as one or more YAML documents separated by `---`, returning one map
per document in file order.

#### Scenario: Two documents in one layer
- GIVEN a layer holding two documents separated by `---`
- WHEN the layer is parsed
- THEN both are returned in order and both feed the merge, the later one winning

### Requirement: Empty Content Tolerance
The system SHALL treat empty layer content and empty documents as empty maps rather than errors.

#### Scenario: Blank layer file
- GIVEN a layer file that is empty or holds only whitespace
- WHEN the layer is parsed
- THEN it yields a single empty map and resolution continues

#### Scenario: Empty document within a multi-document layer
- GIVEN a layer where one document between `---` separators is empty
- WHEN the layer is parsed
- THEN that document yields an empty map and contributes nothing to the merge

### Requirement: Mapping-Rooted Documents
The system SHALL require every document to have a mapping at its top level.

#### Scenario: Sequence at the document root
- GIVEN a layer whose top-level document is a list or a scalar
- WHEN the layer is parsed
- THEN it fails with a `ParseError` stating the top-level document is not a map

#### Scenario: Malformed YAML
- GIVEN a layer holding syntactically invalid YAML
- WHEN the layer is parsed
- THEN it fails with a `ParseError` naming the file and the underlying detail

### Requirement: String-Keyed Normalization
The system SHALL recursively normalize parsed values so every map is string-keyed, and SHALL reject
documents containing a non-string key at any depth.

#### Scenario: Nested map with a non-string key
- GIVEN a nested mapping keyed by an integer
- WHEN the layer is parsed
- THEN it fails with a `ParseError` stating a nested map contains a non-string key

#### Scenario: Nested lists and maps are normalized
- GIVEN a document with maps nested inside lists nested inside maps
- WHEN the layer is parsed
- THEN every level is string-keyed and scalar types are preserved
