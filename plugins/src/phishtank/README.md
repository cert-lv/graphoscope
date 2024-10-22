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

# Access details

Source YAML definition's `access` fields:
- **url**: API access point, for example - `https://checkurl.phishtank.com/checkurl/index.php`
- **agent**: User-Agent to use
