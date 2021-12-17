### STEP 11.
### Write plugin description and documentation


# Template plugin

Template to build new plugins.
Check GUI built-in documentation section `Administration` for a complete
step-by-step workflow.


Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o template.so ./*.go
```

# Access details

Source YAML definition's `access` fields:
- **url**: PROTO://HOST:PORT database's access point, for example - `127.0.0.1:3000`
- **user**: username to connect with
- **password**: user's password
- **db**: database name to use
- **collection**: collection name to query
