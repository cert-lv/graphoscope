# MySQL plugin

Plugin to query MySQL (https://www.mysql.com/) as a data source.


Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o mysql.so ./*.go
```

**Warning**

SQL doesn't allow to query missing columns, like Elasticsearch does.
An error `column "X" does not exist` will be received. That means you must be
very careful with designing a data source and creating a YAML config file to be
able to combine it with data source types other than SQL.

The easiest solution is to exclude MySQL DB from the `global` namespace
and query it independently, to make sure all columns exist.


# Access details

Source YAML definition's `access` fields:
- **addr**: HOST:PORT database's access point, for example - `localhost:3306`
- **user**: username to connect to the database
- **password**: user's password
- **db**: database name to use
- **table**: table name to query


# Usage

Simple example of a new MySQL data source:
```sql
mysql -u root -p

CREATE DATABASE mydb;
CREATE USER 'graphoscope'@'%' IDENTIFIED BY 'password';
GRANT SELECT ON mydb.* TO 'graphoscope'@'%';
FLUSH PRIVILEGES;
USE mydb;
CREATE TABLE mycoll (id SERIAL PRIMARY KEY, email VARCHAR(255) NOT NULL, username VARCHAR(255) NOT NULL, fqdn VARCHAR(255) NOT NULL, count integer NOT NULL, seen TIMESTAMP);
INSERT INTO mycoll (email, username, fqdn, count, seen) VALUES ('a@example.com', 'a', 'example.com', 13, now());
INSERT INTO mycoll (email, username, fqdn, count, seen) VALUES ('b@example.com', 'b', 'example.com', 13, now());
INSERT INTO mycoll (email, username, fqdn, count, seen) VALUES ('c@example.com', 'c', 'example.com', 13, now());
INSERT INTO mycoll (email, username, fqdn, count, seen) VALUES ('d@example.com', 'd', 'example.com', 13, now());
INSERT INTO mycoll (email, username, fqdn, count, seen) VALUES ('e@example.com', 'e', 'example.com', 13, now());
```

Access data will be used by the source's YAML definition. Example:
```yaml
name: mytest

plugin: mysql
inGlobal: false
includeDatetime: false

access:
    addr: 127.0.0.1:3306
    user: graphoscope
    password: password
    db: mydb
    table: mycoll

statsFields:
  - domain


relations:
  -
    from:
        id: email
        group: email
        search: email
        attributes: [ "seen", "fqdn" ]

    to:
        id: username
        group: domain
        search: username

    edge:
        attributes: [ "count" ]
```

Test with a query:
```sh
curl -XGET 'https://localhost:443/api?uuid=auth-key&sql=FROM+mytest+WHERE+email+like+%27a%25%27'
```
