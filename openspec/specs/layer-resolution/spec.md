# layer-resolution Specification

## Purpose
Turns the ordered `layers` input into a concrete, ordered list of YAML files to load, expanding glob
patterns and deciding what happens when an entry resolves to nothing.

## Requirements

### Requirement: Ordered Layer Precedence
The system SHALL treat the `layers` list as ordered by ascending priority: index 0 is the lowest
priority and the last entry is the highest. Later layers override earlier ones.

#### Scenario: Later layer overrides an earlier one
- GIVEN two layer files that both define the key `region`
- WHEN they are listed as `["base.yaml", "prod.yaml"]`
- THEN the resolved config holds the value from `prod.yaml`

#### Scenario: At least one layer is required
- GIVEN a resolve request whose `layers` list is empty
- WHEN the request is constructed
- THEN it fails with the error `layers must not be empty`

### Requirement: Glob Expansion
The system SHALL expand any layer entry containing the glob metacharacters `*`, `?`, or `[` into the
matching files, sorted alphabetically, inserted at the pattern's position in the ordering. `**` SHALL
match recursively.

#### Scenario: Pattern expands in place
- GIVEN a directory `envs/` containing `b.yaml` and `a.yaml`
- WHEN `layers` is `["base.yaml", "envs/*.yaml", "override.yaml"]`
- THEN the effective order is `base.yaml`, `envs/a.yaml`, `envs/b.yaml`, `override.yaml`

#### Scenario: Recursive pattern
- GIVEN YAML files nested at several depths under `conf/`
- WHEN a layer entry is `conf/**/*.yaml`
- THEN every one of them is matched regardless of depth

#### Scenario: Directories are not layers
- GIVEN a glob pattern whose matches include a directory
- WHEN the pattern is expanded
- THEN the directory is excluded and only files are returned

### Requirement: Literal Path Escape Hatch
The system SHALL treat a layer entry prefixed with `literal:` as an exact path, never as a glob, and
SHALL strip the prefix before reading the file.

#### Scenario: Filename containing glob metacharacters
- GIVEN a file literally named `conf/[staging].yaml`
- WHEN the layer entry is `literal:conf/[staging].yaml`
- THEN that exact file is read with no glob expansion

### Requirement: Missing Layer Handling
The system SHALL accept an `on_missing_layer` mode of `error` (default), `warn`, or `skip`, and SHALL
reject any other value. A missing file SHALL abort resolution under `error` and be skipped under
`warn` and `skip`.

#### Scenario: Missing file under error mode
- GIVEN a layer entry pointing to a file that does not exist
- WHEN resolution runs with `on_missing_layer` set to `error`
- THEN it fails with a `LayerNotFoundError` naming the layer path

#### Scenario: Missing file under skip or warn mode
- GIVEN a layer entry pointing to a file that does not exist
- WHEN resolution runs with `on_missing_layer` set to `warn` or `skip`
- THEN the layer is omitted, a debug message is logged, and the remaining layers still resolve

#### Scenario: Glob matching nothing under error mode
- GIVEN a glob pattern that matches no files on disk
- WHEN resolution runs with `on_missing_layer` set to `error`
- THEN it fails with a `NoGlobMatchError` naming the pattern

#### Scenario: Glob matching nothing under skip or warn mode
- GIVEN a glob pattern that matches no files on disk
- WHEN resolution runs with `on_missing_layer` set to `warn` or `skip`
- THEN the pattern contributes no layers and resolution continues

#### Scenario: Invalid mode is rejected up front
- GIVEN an `on_missing_layer` value other than `error`, `warn`, or `skip`
- WHEN the resolve request is constructed
- THEN it fails before any file is read

### Requirement: Loaded Layer Reporting
The system SHALL report the ordered list of layer paths that were actually loaded, after glob
expansion and after missing layers were skipped.

#### Scenario: Skipped layer is absent from the report
- GIVEN three layer entries of which the middle one does not exist
- WHEN resolution runs with `on_missing_layer` set to `skip`
- THEN the loaded-layers report contains exactly the two layers that were read, in order
