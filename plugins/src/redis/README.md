# Redis plugin

Plugin to query Redis (https://redis.io) as a data source.


Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o redis.so ./*.go
```

**Warning**

Redis does NOT accept complex queries, like SQL databases do.

The easiest workaround is to exclude Redis DB from the `global` namespace
and query it independently, to execute needed queries one by one.


# Access details

Source YAML definition's `access` fields:
- **addr**: HOST:PORT database's access point, for example - `localhost:6379`
- **user**: username to connect to the database
- **password**: user's password
- **db**: database number to use
- **field**: Redis key will be used as this field name


# Usage

Simple example of a new Redis data source. Insert test data:
```sh
redis-cli -u redis://localhost:6379/8
ACL SETUSER graphoscope on >password allkeys +hset +hget +hgetall +select +ping
ACL SAVE  # Or 'CONFIG REWRITE'
AUTH graphoscope password

HSET 'a@example.com' username 'a' fqdn 'example.com' count 13 seen '18-02-2023T15:34:00.000000Z'
HSET 'b@example.com' username 'b' fqdn 'example.com' count 13 seen '19-02-2023T15:34:00.000000Z'
HSET 'c@example.com' username 'c' fqdn 'example.com' count 13 seen '20-02-2023T15:34:00.000000Z'
HSET 'd@example.com' username 'd' fqdn 'example.com' count 13 seen '21-02-2023T15:34:00.000000Z'
HSET 'e@example.com' username 'e' fqdn 'example.com' count 13 seen '22-02-2023T15:34:00.000000Z'
```

Access data will be used by the source's YAML definition. Example:
```yaml
name: retest
label: RETest
icon: database

plugin: redis
inGlobal: false
includeDatetime: false
supportsSQL: false

access:
    addr: 127.0.0.1:6379
    user: graphoscope
    password: password
    db: 8
    field: email

queryFields:
  - email

replaceFields:
    datetime: seen
    domain:   fqdn


relations:
  -
    from:
        id: email
        group: email
        search: email
        attributes: [ "seen", "fqdn" ]

    to:
        id: username
        group: username
        search: username

    edge:
        attributes: [ "count" ]
```

Test with a query:
```sh
curl -XGET 'https://localhost:443/api?uuid=auth-key&sql=FROM+retest+WHERE+email=%27a@example.com%27'
```
