# terraform-data-source Specification

## Purpose
Exposes the resolution pipeline to Terraform as the read-only `confstack_layered_config` data source,
defining its attribute contract and how resolved values become Terraform values.

## Requirements

### Requirement: Read-Only Provider Surface
The provider SHALL expose exactly one data source, `confstack_layered_config`, and SHALL expose no
managed resources.

#### Scenario: Provider registration
- GIVEN a Terraform configuration that loads the provider
- WHEN the provider is initialized
- THEN `confstack_layered_config` is available and no resource types are registered

### Requirement: Input Attributes
The data source SHALL accept a required `layers` list of strings and the optional attributes
`on_missing_layer`, `variables`, `secrets`, and `flat_separator`, applying the pipeline defaults when
an optional attribute is null or unknown.

#### Scenario: Only layers supplied
- GIVEN a data source block that sets `layers` and nothing else
- WHEN the data source is read
- THEN resolution uses `on_missing_layer = "error"` and `flat_separator = "."`

#### Scenario: Secrets are marked sensitive
- GIVEN a data source block that sets the `secrets` map
- WHEN Terraform renders the plan
- THEN the map is treated as sensitive and its values are not displayed

#### Scenario: Invalid input is surfaced as a diagnostic
- GIVEN an `on_missing_layer` value that is not supported
- WHEN the data source is read
- THEN the read fails with an error diagnostic rather than a panic

### Requirement: Output Attributes
The data source SHALL compute `config`, `sensitive_config`, `flat_config`, `loaded_layers`, and
`secret_paths`, marking `sensitive_config` sensitive.

#### Scenario: Redacted versus plaintext outputs
- GIVEN a layer that renders `{{ secret "DB_PASS" }}`
- WHEN the data source is read
- THEN `config` shows `(sensitive)` at that path and `sensitive_config` carries the real value

#### Scenario: Loaded layers reflect what was read
- GIVEN a `layers` list where globs expanded and one entry was skipped
- WHEN the data source is read
- THEN `loaded_layers` lists the files actually read, in merge order

### Requirement: Dynamic Value Mapping
The data source SHALL convert the resolved map into a Terraform dynamic value, representing maps as
objects, lists as tuples so mixed element types are allowed, and numbers as float64.

#### Scenario: Mixed-type list
- GIVEN a resolved list holding a string and a number
- WHEN it is converted to a Terraform value
- THEN it becomes a tuple and the read succeeds

#### Scenario: Empty list
- GIVEN a resolved value that is an empty list
- WHEN it is converted to a Terraform value
- THEN it becomes an empty tuple

#### Scenario: Null value
- GIVEN a resolved value that is null
- WHEN it is converted to a Terraform value
- THEN it becomes a Terraform null dynamic value

### Requirement: Flattened String View
The data source SHALL expose `flat_config` as a map of strings keyed by `flat_separator`-delimited
paths, recursing only into maps and stringifying every leaf value.

#### Scenario: Nested map is flattened
- GIVEN a resolved config holding `db: {host: localhost, port: 5432}`
- WHEN the data source is read
- THEN `flat_config` holds `db.host = "localhost"` and `db.port = "5432"`

#### Scenario: Custom separator
- GIVEN a `flat_separator` of `/`
- WHEN the data source is read
- THEN the flattened key is `db/host`

#### Scenario: List leaves are not expanded
- GIVEN a resolved value that is a list
- WHEN the config is flattened
- THEN it appears as one flattened key whose value is the list's string representation

### Requirement: Deterministic Secret Path Ordering
The data source SHALL emit `secret_paths` sorted so repeated plans do not show spurious diffs.

#### Scenario: Multiple secrets
- GIVEN a configuration where three paths hold secrets
- WHEN the data source is read repeatedly
- THEN `secret_paths` lists them in sorted order every time
