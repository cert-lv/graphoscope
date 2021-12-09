# SQLite plugin

Plugin to query SQLite (https://www.sqlite.org) as a data source.


Compile with:
```sh
CGO_EBABLED=1 CGO_CFLAGS="-g -O2 -Wno-return-local-addr" go build -buildmode=plugin -ldflags="-w" -o sqlite.so ./*.go
```
Test with:
```sh
CGO_CFLAGS="-g -O2 -Wno-return-local-addr" go test
```
**CFLAGS** as a temp. solution for the https://github.com/mattn/go-sqlite3/issues/803


# WARNINGS

When a `SIGSEGV` occurs while running a C code called via cgo (what SQLite
plugin does), that `SIGSEGV` is not turned into a Go panic. The mechanism that
Go uses to turn a memory error into a panic can only work for a Go code, not
for a C code. That means `segmentation violation` errors in a C code will crash
the API service.

---

SQL doesn't allow to query missing columns, like Elasticsearch does.
An error `no such column: X` will be received. That means you must be very
careful with designing a data source and creating a YAML config file to be able
to combine it with data source types other than SQL.

The easiest solution is to exclude SQLite DB from the `global` namespace and
query it independently, to make sure all columns exist.


# Access details

YAML source configuration possible fields:
- **db**: database file to use, for example - `/data/sqtest.db`
- **table**: table name to query


# Demo

Simple example of creation a new SQLite data source from a CLI:
```sql
sqlite3 sqtest.db

CREATE TABLE sqcoll (email VARCHAR(255) NOT NULL, username VARCHAR(255) NOT NULL, fqdn VARCHAR(255) NOT NULL, count integer NOT NULL, seen TIMESTAMP);
INSERT INTO sqcoll (email, username, fqdn, count, seen) VALUES ('a@example.com', 'a', 'example.com', 13, DateTime('now', 'localtime'));
INSERT INTO sqcoll (email, username, fqdn, count, seen) VALUES ('b@example.com', 'b', 'example.com', 13, DateTime('now', 'localtime'));
INSERT INTO sqcoll (email, username, fqdn, count, seen) VALUES ('c@example.com', 'c', 'example.com', 13, DateTime('now', 'localtime'));
INSERT INTO sqcoll (email, username, fqdn, count, seen) VALUES ('d@example.com', 'd', 'example.com', 13, DateTime('now', 'localtime'));
INSERT INTO sqcoll (email, username, fqdn, count, seen) VALUES ('e@example.com', 'e', 'example.com', 13, DateTime('now', 'localtime'));
.quit
```

Access data will be used by the YAML configs. Example:
```yaml
name: sqtest

plugin: sqlite
inGlobal: true
includeDatetime: false

access:
    db: /data/sqtest.db
    table: sqcoll

statsFields:
  - domain

replaceFields:
  - [ "datetime", "seen" ]
  - [ "domain",   "fqdn" ]


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
curl -XGET '127.0.0.1:9000/?sql=FROM+sqtest+WHERE+email+like+%27a%25%27'
```
