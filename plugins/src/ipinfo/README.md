# ipinfo.io plugin

Connector sends a GET request to the `ipinfo.io` API and expects an JSON response with IP details back.
Request can contain `ip` field only with an IP address to check.

`curl` to test:
```sh
curl 'https://localhost:443/api?uuid=auth-key&sql=FROM+ipinfo+WHERE+ip=%278.8.8.8%27'
```

Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o ipinfo.so ./*.go
```

# Limitations

Does not support complex SQL queries and datetime range selection.


# Access details

Source YAML definition's `access` fields:
- **server**: API server, for example - `https://ipinfo.io`
- **token**: User's access token, for extended queries limit


# Definition file example

Replace API token with your own:
```yaml
name: ipinfo
label: ipinfo.io
icon: database

plugin: ipinfo
inGlobal: true
includeDatetime: false
supportsSQL: false

access:
    server: https://ipinfo.io
    token: xxxxxxxxxxxxxx

queryFields:
    - ip


relations:
  -
    from:
        id: ip
        group: ip
        search: ip
        attributes: ["anycast", "city", "loc", "postal", "region"]

    to:
        id: hostname
        group: domain
        search: domain

  -
    from:
        id: ip
        group: ip
        search: ip
        attributes: ["anycast", "city", "loc", "postal", "region"]

    to:
        id: org
        group: institution
        search: institution
        attributes: ["asn", "city", "loc", "postal", "region"]
```