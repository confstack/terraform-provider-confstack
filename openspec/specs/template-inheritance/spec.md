# template-inheritance Specification

## Purpose
Lets configuration entries reuse named blocks declared under `_templates` by referencing them with
`_inherit`, resolved after merging and stripped from the final output.

## Requirements

### Requirement: Reserved Keys Are Configurable
The system SHALL use `_templates` and `_inherit` as the default reserved keys and SHALL allow both to
be overridden per resolution.

#### Scenario: Defaults apply
- GIVEN a resolve request that supplies no key overrides
- WHEN inheritance is resolved
- THEN `_templates` and `_inherit` are the recognized reserved keys

### Requirement: Template Definition And Reference
The system SHALL merge the referenced template's contents as the base for the entry, with the entry's
own keys overriding the inherited ones.

#### Scenario: Entry overrides an inherited key
- GIVEN a template `base` defining `cpu: 1` and `mem: 2`
- WHEN an entry declares `_inherit: base` and its own `cpu: 4`
- THEN the entry resolves to `cpu: 4` and `mem: 2`

#### Scenario: Multiple templates applied in order
- GIVEN templates `a` and `b` that define the same key
- WHEN an entry declares `_inherit: [a, b]`
- THEN `b` wins over `a`, and the entry's own value wins over both

### Requirement: Selective Inheritance With Except
The system SHALL accept `_inherit` list items in object form `{template, except}` and SHALL omit the
keys named in `except` from what is inherited.

#### Scenario: Excluding a key from a template
- GIVEN a template `base` defining `cpu` and `mem`
- WHEN an entry declares `_inherit: [{template: base, except: [mem]}]`
- THEN the entry inherits `cpu` only

#### Scenario: Malformed directive
- GIVEN an `_inherit` value that is neither a string nor a list, or a list object lacking a string `template` key
- WHEN inheritance is resolved
- THEN it fails with an error naming the offending entry path

### Requirement: Bubble-Up Template Lookup
The system SHALL resolve a template name against the `_templates` blocks visible from the entry's own
scope and every ancestor scope.

#### Scenario: Template declared at the root
- GIVEN a `_templates` block at the document root
- WHEN a deeply nested entry references one of its templates
- THEN the template resolves

#### Scenario: Unknown template name
- GIVEN an entry referencing a template name visible in no ancestor scope
- WHEN inheritance is resolved
- THEN it fails with a `TemplateNotFoundError` naming the entry path and the template name

### Requirement: Template Uniqueness
The system SHALL require template names to be globally unique across all `_templates` blocks in the
merged tree.

#### Scenario: Same name in two blocks
- GIVEN two `_templates` blocks that both define a template named `base`
- WHEN inheritance is resolved
- THEN it fails with a `DuplicateTemplateError` naming `base`

### Requirement: Templates May Not Inherit
The system SHALL reject a template definition that itself contains an `_inherit` directive.

#### Scenario: Nested inheritance in a definition
- GIVEN a template body that contains `_inherit`
- WHEN inheritance is resolved
- THEN it fails with a `TemplateWithInheritError` naming the template and the inherit key

### Requirement: Reserved Keys Stripped From Output
The system SHALL remove every `_templates` and `_inherit` key at every depth after inheritance is
resolved, so neither appears in any output.

#### Scenario: No reserved keys survive
- GIVEN a configuration that uses templates and inheritance
- WHEN resolution completes
- THEN the resolved config, the sensitive config, and the flattened config hold no `_templates` or `_inherit` key

### Requirement: Deterministic Resolution Order
The system SHALL process keys in sorted order while resolving inheritance so results are
reproducible across runs.

#### Scenario: Repeated resolution
- GIVEN the same set of layers
- WHEN they are resolved twice
- THEN the resolved tree is identical
