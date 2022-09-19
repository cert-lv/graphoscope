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

# Access details

Source YAML definition's `access` fields:
- **url**: HTTP access point, for example - `http://localhost:7000`
