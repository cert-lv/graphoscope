# Elasticsearch plugin

Plugin to query Elasticsearch (https://www.elastic.co/elasticsearch) as a data source.

SQl convertor's base: https://github.com/cch123/elasticsql


Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o elasticsearch.v7.so ./*.go
```

# Access details

Source YAML definition's `access` fields:
- **url**: HTTP access point, for example - `http://localhost:9200`
- **username**: username for the basic auth
- **password**: password for the basic auth
- **key**: authorization key
- **indices**: comma separated indices patterns to query, for example - `apps-*`

Only `username/password` or `key` can be used at once.


## Limitations

- Go package supports specific Elasticsearch major version only,
  so version number is included in a plugin's name


## TODO

- [ ] Try the official Elasticsearch Go package
- [ ] Implement 'NOT BETWEEN' in SQL
