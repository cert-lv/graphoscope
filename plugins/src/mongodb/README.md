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


# Golang driver

The **mongo-go-driver** contains four object types:

- **bson.D**: A BSON document. This type should be used in situations where order matters, such as MongoDB commands
- **bson.M**: An unordered map. It is the same as D, except it does not preserve order
- **bson.A**: A BSON array
- **bson.E**: A single element inside a D


## TODO

- [ ] Implement 'NOT BETWEEN' in SQL
