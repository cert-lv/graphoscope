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

# Example definition file

```yaml
name: hashlookup
label: Hashlookup
icon: database

plugin: hashlookup
inGlobal: true
includeDatetime: false
supportsSQL: true

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
        group: identifier
        search: sha1
        attributes: ["FileName", "FileSize", "source-url","MD5" ,"SHA-512" , "SHA-256", "SHA-1", "SSDEEP", "TLSH", "insert-timestamp", "mimetype", "source", "hashlookup-parent-total", "snap-authority"]

    to:
        id: parent
        group: identifier
        search: sha1
        attributes: ["FileName", "FileSize", "source-url","MD5" ,"SHA-512" , "SHA-256", "SHA-1", "SSDEEP", "TLSH", "insert-timestamp", "mimetype", "source", "hashlookup-parent-total", "snap-authority"]

    edge:
        label: ChildOf
        attributes: []
  -
    from:
      id: SHA-1
      group: identifier
      search: sha1
      attributes: ["FileName", "FileSize", "source-url","MD5" ,"SHA-512" , "SHA-256", "SHA-1", "SSDEEP", "TLSH", "insert-timestamp", "mimetype", "source", "hashlookup-parent-total", "snap-authority"]

    to:
      id: children
      group: identifier
      search: sha1
      attributes: ["FileName", "FileSize", "source-url","MD5" ,"SHA-512" , "SHA-256", "SHA-1", "SSDEEP", "TLSH", "insert-timestamp", "mimetype", "source", "hashlookup-parent-total", "snap-authority"]

    edge:
      label: ParentOf
      attributes: []

  -
    from:
        id: SHA-1
        group: identifier
        search: sha1
        attributes: ["FileName", "FileSize", "source-url","MD5" ,"SHA-
512" , "SHA-256", "SHA-1", "SSDEEP", "TLSH", "insert-timestamp",
"mimetype", "source", "hashlookup-parent-total", "snap-authority"]

    to:
        id: MD5
        group: identifier
        search: md5
        attributes: ["FileName", "FileSize", "source-url","MD5" ,"SHA-
512" , "SHA-256", "SHA-1", "SSDEEP", "TLSH", "insert-timestamp",
"mimetype", "source", "hashlookup-parent-total", "snap-authority"]

  -
    from:
        id: SHA-1
        group: identifier
        search: sha1
        attributes: ["FileName", "FileSize", "source-url","MD5" ,"SHA-
512" , "SHA-256", "SHA-1", "SSDEEP", "TLSH", "insert-timestamp",
"mimetype", "source", "hashlookup-parent-total", "snap-authority"]

    to:
        id: SHA-256
        group: identifier
        search: sha256
        attributes: ["FileName", "FileSize", "source-url","MD5" ,"SHA-
512" , "SHA-256", "SHA-1", "SSDEEP", "TLSH", "insert-timestamp",
"mimetype", "source", "hashlookup-parent-total", "snap-authority"]
```