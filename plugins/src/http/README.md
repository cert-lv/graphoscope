# HTTP plugin

HTTP connector sends a GET/POST request with fields
and expects a `[{...},{...},{...}]` formatted JSON response back.

`curl` to test:
```sh
curl 'https://localhost:443/api?uuid=auth-key&sql=FROM+service+WHERE+ip=%278.8.8.8%27'
```

Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o http.so ./*.go
```

# Access details

Source YAML definition's `access` fields:
- **url**: HTTP access point, for example - `http://localhost:8000`
- **method**: `GET` or `POST`, GET by default
