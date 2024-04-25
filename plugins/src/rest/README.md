# REST API plugin

Connector sends a GET request and expects a list of flat JSON objects back, one line - one JSON.
To the preconfigured REST API URL `field/value` will be attached as query.

`curl` to test:
```sh
curl 'https://localhost:443/api?uuid=auth-key&sql=FROM+service+WHERE+ip=%278.8.8.8%27'
```

Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o rest.so ./*.go
```

# Access details

Source YAML definition's `access` fields:
- **url**: REST API access point, for example - `http://localhost:8000/RESTv1`
- **username**: Username if exists
- **password**: User's password if exists
