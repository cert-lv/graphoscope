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


# Limitations

Does not support complex SQL queries and datetime range selection.


# Access details

Source YAML definition's `access` fields:
- **url**: hashlookup API endpoint, for example - `https://hashlookup.circl.lu`
- **apiKey**: optional hashlookup API key


# Definition file example

Replace API key with your own:
```yaml
name: hashlookup
label: Hashlookup
icon: database

plugin: hashlookup
inGlobal: true
includeDatetime: false
supportsSQL: false

access:
    url: https://hashlookup.circl.lu
    apiKey: .

queryFields:
    - md5
    - sha1
    - sha256


relations:
  -
    from:
        id: SHA-1
        group: sha1
        search: sha1
        attributes: ["FileName", "FileSize", "source-url", "MD5", "SHA-512", "SHA-256", "SSDEEP", "TLSH", "insert-timestamp", "mimetype", "source", "hashlookup-parent-total", "snap-authority", "hashlookup:trust"]

    to:
        id: parent
        group: sha1
        search: sha1
        attributes: ["FileName", "FileSize", "source-url", "MD5", "SHA-512", "SHA-256", "SSDEEP", "TLSH", "insert-timestamp", "mimetype", "source", "hashlookup-parent-total", "snap-authority", "hashlookup:trust"]

    edge:
        label: ChildOf

  -
    from:
        id: SHA-1
        group: sha1
        search: sha1
        attributes: ["FileName", "FileSize", "source-url", "MD5", "SHA-512", "SHA-256", "SSDEEP", "TLSH", "insert-timestamp", "mimetype", "source", "hashlookup-parent-total", "snap-authority", "hashlookup:trust"]

    to:
        id: children
        group: sha1
        search: sha1
        attributes: ["FileName", "FileSize", "source-url", "MD5", "SHA-512", "SHA-256", "SSDEEP", "TLSH", "insert-timestamp", "mimetype", "source", "hashlookup-parent-total", "snap-authority", "hashlookup:trust"]

    edge:
        label: ParentOf

  -
    from:
        id: SHA-1
        group: sha1
        search: sha1
        attributes: ["FileName", "FileSize", "source-url", "SHA-512", "SHA-256", "SSDEEP", "TLSH", "insert-timestamp", "mimetype", "source", "hashlookup-parent-total", "snap-authority", "hashlookup:trust"]

    to:
        id: MD5
        group: md5
        search: md5
        attributes: ["FileName", "FileSize", "source-url", "SHA-512", "SHA-256", "SSDEEP", "TLSH", "insert-timestamp", "mimetype", "source", "hashlookup-parent-total", "snap-authority", "hashlookup:trust"]

  -
    from:
        id: SHA-1
        group: sha1
        search: sha1
        attributes: ["FileName", "FileSize", "source-url", "MD5", "SHA-512", "SSDEEP", "TLSH", "insert-timestamp", "mimetype", "source", "hashlookup-parent-total", "snap-authority", "hashlookup:trust"]

    to:
        id: SHA-256
        group: sha256
        search: sha256
        attributes: ["FileName", "FileSize", "source-url", "MD5", "SHA-512", "SSDEEP", "TLSH", "insert-timestamp", "mimetype", "source", "hashlookup-parent-total", "snap-authority", "hashlookup:trust"]
```
