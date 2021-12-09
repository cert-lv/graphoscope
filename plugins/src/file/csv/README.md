# CSV file plugin

Plugin to query CSV file as a data source.

SQL doesn't allow to query missing columns, like Elasticsearch does.
An error `field X does not exist` will be received. That means you must be very
careful with designing a data source and creating a YAML config file to be able
to combine it with data source types other than SQL.

The easiest solution is to exclude data source from the `global` namespace
and query it independently, to make sure all columns exist.

`curl` to test:
```sh
curl '127.0.0.1:9000/?sql=FROM+csvfile+WHERE+ip=%278.8.8.8%27'
```

Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o file-csv.so ./*.go
```

# Access details

YAML source configuration possible fields:
- **path**: CSV file to use, for example - `/data/test.csv`
