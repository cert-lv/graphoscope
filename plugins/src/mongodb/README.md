# MongoDB plugin

Plugin to query MongoDB (https://www.mongodb.com) as a data source.


Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o mongodb.so ./*.go
```

# Access details

Source YAML definition's `access` fields:
- **addr**: HOST:PORT database's access point, for example - `localhost:27017`
- **db**: database name to use
- **collection**: collection name to query


# Get all the possible fields

As MongoDB doesn't provide a built-in way to get all the possible collection's
fields without requesting all documents plugin will try:
1. To return a manually filled `queryFields` - useful when some fields rarely appear
2. To request 1000 documents and get all their unique fields, including nested


# Golang driver

The **mongo-go-driver** contains four object types:

- **bson.D**: A BSON document. This type should be used in situations where order matters, such as MongoDB commands
- **bson.M**: An unordered map. It is the same as D, except it does not preserve order
- **bson.A**: A BSON array
- **bson.E**: A single element inside a D


## TODO

- [ ] Implement 'NOT BETWEEN' in SQL
