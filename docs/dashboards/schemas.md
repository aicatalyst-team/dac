# Schemas

DAC YAML files use versioned Bruin schema identifiers.

## Schema IDs

| File type | Schema |
|-----------|--------|
| Dashboard YAML | `https://getbruin.com/schemas/dac/dashboard/v1` |
| Semantic model YAML | `https://getbruin.com/schemas/dac/semantic-model/v1` |
| Theme YAML | `https://getbruin.com/schemas/dac/theme/v1` |

Dashboard example:

```yaml
schema: https://getbruin.com/schemas/dac/dashboard/v1
name: Sales
rows:
  - widgets:
      - name: Revenue
        type: metric
        sql: SELECT SUM(amount) AS value FROM sales
        column: value
```

Semantic model example:

```yaml
schema: https://getbruin.com/schemas/dac/semantic-model/v1
name: sales
source:
  table: sales
metrics:
  - name: revenue
    expression: sum(amount)
```

Theme example:

```yaml
schema: https://getbruin.com/schemas/dac/theme/v1
name: corporate
tokens:
  background: "#FFFFFF"
```

## Validation

`dac validate`, `dac check`, `dac serve`, and `dac build` validate YAML files against these schemas before running deeper DAC validation.

JSON Schema validates structure: required fields, field types, enum values, and unknown fields. DAC's Go validators still validate meaning: query references, semantic model references, metric references, segments, filters, sort fields, and layout rules.

## Extension Fields

Schema-defined objects allow extension fields that start with `x-`:

```yaml
schema: https://getbruin.com/schemas/dac/dashboard/v1
name: Sales
x-owner: analytics
rows:
  - widgets:
      - name: Notes
        type: text
        content: Owned by analytics
```

Use `x-` fields for local metadata. DAC ignores them.
