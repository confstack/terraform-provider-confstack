# layer-templating Specification

## Purpose
Renders each layer file as a Go text/template before it is parsed as YAML, injecting non-sensitive
values via `var` and routing sensitive values through opaque sentinels via `secret`.

## Requirements

### Requirement: Go Template Rendering With Sprig
The system SHALL render every layer file as a Go `text/template` with the Sprig function library
available, before the content is parsed as YAML.

#### Scenario: Sprig function is available
- GIVEN a layer containing `name: {{ "prod" | upper }}`
- WHEN the layer is rendered and parsed
- THEN the value of `name` is `PROD`

#### Scenario: Template syntax error
- GIVEN a layer containing an unparseable template expression
- WHEN the layer is rendered
- THEN it fails with a `TemplateRenderError` naming the file and the `parse` stage

#### Scenario: Template execution error
- GIVEN a layer whose template parses but fails while executing
- WHEN the layer is rendered
- THEN it fails with a `TemplateRenderError` naming the file and the `execution` stage

### Requirement: Variable Injection
The system SHALL resolve `{{ var "KEY" }}` from the `variables` map first, then fall back to the OS
environment, and SHALL substitute the raw string value.

#### Scenario: Value from the variables map
- GIVEN a `variables` map containing `REGION=eu-west-1`
- WHEN a layer renders `{{ var "REGION" }}`
- THEN the rendered text contains `eu-west-1`

#### Scenario: Fallback to the environment
- GIVEN `REGION` absent from `variables` but set as a non-empty environment variable
- WHEN a layer renders `{{ var "REGION" }}`
- THEN the environment value is used

#### Scenario: Key resolvable nowhere
- GIVEN a key present in neither `variables` nor the environment
- WHEN a layer renders `{{ var }}` for that key
- THEN it fails with a `MissingVariableError` naming the key and the `var` function

### Requirement: Secret Injection Via Sentinels
The system SHALL resolve `{{ secret "KEY" }}` from the `secrets` map first, then from the OS
environment, and SHALL substitute an opaque sentinel of the form `__CONFSTACK_SECRET_<sha256-hex>__`
rather than the real value, recording the sentinel-to-value mapping for later resolution.

#### Scenario: Secret never reaches the YAML parser
- GIVEN a `secrets` map containing `DB_PASS`
- WHEN a layer renders `password: {{ secret "DB_PASS" }}`
- THEN the text handed to the YAML parser holds a sentinel, not the real password

#### Scenario: Secret key resolvable nowhere
- GIVEN a key present in neither `secrets` nor the environment
- WHEN a layer renders `{{ secret }}` for that key
- THEN it fails with a `MissingVariableError` naming the key and the `secret` function

### Requirement: Per-Resolution Sentinel Nonce
The system SHALL generate one nonce per resolution call and derive every sentinel from that nonce
plus the secret key, so sentinels are stable within a run and differ across runs.

#### Scenario: Same key twice in one run
- GIVEN two layers that both render `{{ secret "TOKEN" }}`
- WHEN both are resolved in the same call
- THEN both produce the identical sentinel and resolve to the same value

#### Scenario: Same key across two runs
- GIVEN the same configuration resolved twice
- WHEN the sentinels of each run are compared
- THEN they differ, so a sentinel from one run cannot collide with another

### Requirement: Sentinel Concatenation
The system SHALL support a sentinel appearing inline within a larger string value, not only as the
entire value.

#### Scenario: Secret embedded in a connection string
- GIVEN a `secrets` map containing `DB_PASS`
- WHEN a layer renders `url: "postgres://user:{{ secret "DB_PASS" }}@host/db"`
- THEN the value is a single string carrying the sentinel in place of the password
