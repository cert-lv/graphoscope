# Pastelyzer plugin

Plugin to query Pastelyzer (https://github.com/cert-lv/pastelyzer) as a data source.

Sample command to use plugin:
```sh
curl 'https://localhost:443/api?uuid=auth-key&sql=FROM+pastelyzer+WHERE+ip=%278.8.8.8%27'
```


Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o pastelyzer.so ./*.go
```


# Access details

Source YAML definition's `access` fields:
- **url**: HTTP access point, for example - `http://localhost:7000`
