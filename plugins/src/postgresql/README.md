# PostgreSQL plugin

Plugin to query PostgreSQL (https://www.postgresql.org/) as a data source.


Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o postgresql.so ./*.go
```

**Warning**

SQL doesn't allow to query missing columns, like Elasticsearch does.
An error `column "X" does not exist` will be received.
That means you must be very careful with designing a data source
and creating a YAML config file to be able to
combine it with data source types other than SQL.

The easiest solution is to exclude PostgreSQL DB from the `global` namespace
and query it independently, to make sure all columns exist.


# Access details

Source YAML definition's `access` fields:
- **addr**: HOST:PORT database's access point, for example - `localhost:5432`
- **user**: username to connect to the database
- **password**: user's password
- **db**: database name to use
- **table**: table name to query


# Demo

Simple example of a new PostgreSQL data source:
```sql
sudo -u postgres psql

CREATE DATABASE pgdb;
CREATE USER graphoscope WITH ENCRYPTED PASSWORD 'password';
GRANT ALL PRIVILEGES ON DATABASE pgdb TO graphoscope;
\connect pgdb
CREATE TABLE pgcoll (id SERIAL PRIMARY KEY, email VARCHAR(255) NOT NULL, username VARCHAR(255) NOT NULL, fqdn VARCHAR(255) NOT NULL, count integer NOT NULL, seen TIMESTAMP);
GRANT ALL PRIVILEGES ON TABLE pgcoll TO graphoscope;

INSERT INTO pgcoll (email, username, fqdn, count, seen) VALUES ('a@example.com', 'a', 'example.com', 13, now());
INSERT INTO pgcoll (email, username, fqdn, count, seen) VALUES ('b@example.com', 'b', 'example.com', 13, now());
INSERT INTO pgcoll (email, username, fqdn, count, seen) VALUES ('c@example.com', 'c', 'example.com', 13, now());
INSERT INTO pgcoll (email, username, fqdn, count, seen) VALUES ('d@example.com', 'd', 'example.com', 13, now());
INSERT INTO pgcoll (email, username, fqdn, count, seen) VALUES ('e@example.com', 'e', 'example.com', 13, now());
```

Access data will be used by the YAML configs. Example:
```yaml
name: pgtest
label: PGTest
icon: database

plugin: postgresql
inGlobal: true
includeDatetime: false
supportsSQL: true

access:
    addr: 127.0.0.1:5432
    db: pgdb
    table: pgcoll
    user: graphoscope
    password: password

statsFields:
  - domain

replaceFields:
    datetime: seen
    domain:   fqdn


relations:
  -
    from:
        id: email
        group: email
        search: email
        attributes: [ "username", "fqdn" ]

    to:
        id: fqdn
        group: domain
        search: domain

    edge:
        attributes: [ "count" ]
```

Test with a query:
```sh
curl -XGET 'https://localhost:443/api?uuid=auth-key&sql=FROM+pgtest+WHERE+email+like+%27a%25%27'
```

# TODO

 - [ ] Check `TODO` in `convert.go`
