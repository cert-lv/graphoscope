# Pastelyzer plugin

Plugin to query Pastelyzer (https://github.com/cert-lv/pastelyzer) as a data source.

Sample command to use plugin:
```sh
# Get paste IDs where IP 8.8.8.8 was mentioned
curl 'https://localhost:443/api?uuid=auth-key&sql=FROM+pastelyzer+WHERE+ip=%278.8.8.8%27'
# Get all artefacts of the given paste ID
curl 'https://localhost:443/api?uuid=auth-key&sql=FROM+pastelyzer+WHERE+source=35853628'
```

As there is no way to get automatically all the possible fields to query (for the Web GUI autocomplete) - such artefacts are:
- cc-number
- credential
- domain
- email
- ip
- onion
- sha1
- any

Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o pastelyzer.so ./*.go
```

# Limitations

Does not support complex SQL queries and datetime range selection.


# Access details

Source YAML definition's `access` fields:
- **url**: HTTP access point, for example - `http://localhost:7000`


# Definition file example

```yaml
name: pastelyzer
label: Pastelyzer
icon: copy outline

plugin: pastelyzer
inGlobal: true
includeDatetime: false
supportsSQL: false

access:
    url: http://127.0.0.1:7000

queryFields:
    - source
    - cc-number
    - credential
    - domain
    - email
    - ip
    - onion
    - sha1
    - any

statsFields:
  - ip
  - domain
  - type


relations:
  -
    from:
        id: domain
        group: domain
        search: domain

    to:
        id: source
        group: paste
        search: source

    edge:
        label: was published

  -
    from:
        id: ip
        group: ip
        search: ip

    to:
        id: source
        group: paste
        search: source

    edge:
        label: was published

  -
    from:
        id: address
        group: ip
        search: ip

    to:
        id: source
        group: paste
        search: source

    edge:
        label: was published

  -
    from:
        id: cc-number
        group: cc-number
        search: cc-number

    to:
        id: source
        group: paste
        search: source

    edge:
        label: was published

  -
    from:
        id: credential
        group: credentials
        search: credential

    to:
        id: source
        group: paste
        search: source

    edge:
        label: was published

  -
    from:
        id: email
        group: email
        search: email

    to:
        id: source
        group: paste
        search: source

    edge:
        label: was published

  -
    from:
        id: onion
        group: onion
        search: onion

    to:
        id: source
        group: paste
        search: source

    edge:
        label: was published

  -
    from:
        id: sha1
        group: sha1
        search: sha1

    to:
        id: source
        group: paste
        search: source

    edge:
        label: was published
```