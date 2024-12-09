# Phishtank plugin

Connector sends a GET request to the `phishtank.org` API and expects an XML response back.
Request can contain `url` field only with a URL to check.

`curl` to test:
```sh
curl 'https://localhost:443/api?uuid=auth-key&sql=FROM+service+WHERE+url=%27http%3A%2F%2Fexample.com%27'
```

Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o phishtank.so ./*.go
```

# Limitations

Does not support complex SQL queries and datetime range selection.


# Access details

Source YAML definition's `access` fields:
- **url**: API access point, for example - `https://checkurl.phishtank.com/checkurl/index.php`
- **agent**: User-Agent to use


# Definition file example

```yaml
name: phishtank
label: Phishtank
icon: database

plugin: phishtank
inGlobal: true
includeDatetime: false
supportsSQL: false

access:
    url: https://checkurl.phishtank.com/checkurl/index.php
    agent: phishtank/graphoscope

queryFields:
    - url
    - domain


relations:
  -
    from:
        id: url
        group: url
        search: url
        attributes: ["in_database", "phish_id", "phish_detail_page", "verified", "verified_at", "valid"]

    to:
        id: domain
        group: domain
        search: domain
```