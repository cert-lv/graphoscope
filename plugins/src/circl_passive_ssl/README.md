# CIRCL Passive SSL plugin

Data source: https://www.circl.lu/services/passive-ssl/

Connector sends a GET request to the `https://www.circl.lu/v2pssl/` and expects a JSON back.
To the preconfigured URL `field/value` will be attached as query.

`curl` to test:
```sh
curl 'https://localhost:443/api?uuid=auth-key&sql=FROM+service+WHERE+ip=%278.8.8.8%27'
```

Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o circl_passive_ssl.so ./*.go
```

# Limitations

Does not support complex SQL queries and datetime range selection.


# Access details

Source YAML definition's `access` fields:
- **url**: REST API access point, `https://www.circl.lu/v2pssl/`
- **username**: Username
- **password**: User's password


# YAML definition example

```yaml
name: circl_passive_ssl
label: CIRCL Passive SSL
icon: retweet

plugin: circl_passive_ssl
inGlobal: false
includeDatetime: false
supportsSQL: false

access:
    url: https://www.circl.lu/v2pssl
    username: user
    password: password_

queryFields:
    - ip
    - network
    - sha1

statsFields:
    - ip
    - sha1
    - subject


relations:
  -
    from:
        id: sha1
        group: sha1
        search: sha1
        attributes: ["subject"]

    to:
        id: ip
        group: ip
        search: ip
```
