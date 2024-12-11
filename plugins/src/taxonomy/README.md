# Taxonomy plugin

Apply the predefined taxonomy to all nodes. New virtual nodes will be inserted, which will group existing nodes.


Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o taxonomy.so ./*.go
```

# Configuration

YAML definition's `data` fields:
- **field**: field to use as a filter
- **group**: as all graph nodes belong to some group/type, it can be used to apply taxonomy to the specific group only
- **taxonomy**: mapping to use. If node/edge's **field** is equal to the "key" - insert a new relation with "value" as a new node

Mapping example:
```yaml
taxonomy:
    field_value_1: taxonomy_group
    field_value_2: taxonomy_group
```
