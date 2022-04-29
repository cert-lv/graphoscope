# Hashlookup plugin

Plugin to query [hashlookup services](https://github.com/hashlookup).
For instance hashlookup.circl.lu

Sample command to use plugin:
```sh
curl 'https://localhost:443/api?uuid=auth-key&sql=FROM+hashlookup+WHERE+sha1=%27deac5aeda66017c25a9e21f36f5ee618d2ad9d3d%27'
```

Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o hashlookup.so ./*.go
```
Or use the Makefile command:
`make plugins-local`

# Access details

Source YAML definition's `access` fields:
- **url**: hashlookup API endpoint, for example - `https://hashlookup.circl.lu`
- **apiKey**: optional hashlookup API key
