# Modify plugin

Allows to modify parameters of existing graph elements.


Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o modify.so ./*.go
```

# Configuration

YAML definition's `data` fields:
- **group**: as all graph nodes belong to some group/type, it can be used to apply modifications to the specific group only
- **modify**: list of replacements. If **regex** matches node/edge's **field** value - put **replacement** instead

More info about used function: https://pkg.go.dev/regexp#Regexp.ReplaceAllString,
where:
    - `MustCompile` receives regex
    - `ReplaceAllString` receives graph field value and replacement

List of replacements example:
```yaml
modify:
    - field: field_name
      regex: a(x*)b
      replacement: y

    # Anonymize prices
    - field: price
      regex: (?i)\d{1,3}(?:[.,]\d{3})*(?:[.,]\d{2}) *(eur|€)
      replacement: *** Eur
```
