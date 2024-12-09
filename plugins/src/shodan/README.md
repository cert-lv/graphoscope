# Shodan plugin

Connector sends requests to the `shodan.io` API.

Supported API endpoints:
    - `https://api.shodan.io/shodan/host/{ip}`    -> accepts `ip`
    - `https://api.shodan.io/shodan/host/search`  -> accepts text `query`
    - `https://api.shodan.io/dns/domain/{domain}` -> accepts `domain`
    - `https://api.shodan.io/dns/resolve`         -> accepts `domain`
    - `https://api.shodan.io/dns/reverse`         -> accepts `ip`
    - `https://exploits.shodan.io/api/search`     -> accepts vulnerability `query`

API docs at: https://developer.shodan.io/api

`curl` to test:
```sh
curl 'https://localhost:443/api?uuid=auth-key&sql=FROM+shodan+WHERE+ip=%278.8.8.8%27'
```

Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o shodan.so ./*.go
```

# Access details

Source YAML definition's `access` fields:
- **key**: User's API token
- **pages**: Max amount of Shodan pages to request. For every 100 results past the 1'st page 1 query credit is deducted.
- **credits**: Query credits limit per month


# Definition file example

Replace API key with your own:
```yaml
name: shodan
label: Shodan
icon: globe

plugin: shodan
inGlobal: false
includeDatetime: false
supportsSQL: false

access:
    key: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
    pages: 10
    credits: 200000

statsFields:
    - organization
    - hostLocationCountryCode
    - product
    - port
    - version
    - ip

queryFields:
    - ip
    - domain
    - query
    - vulnerability


relations:
  -
    from:
        id: hostname
        group: domain
        search: domain
        attributes: [ "shodanCreditsLeft" ]

    to:
        id: ip
        group: ip
        search: ip
        attributes: [ "shodanCreditsLeft" ]

    edge:
        attributes: [ "shodanCreditsLeft" ]

  -
    from:
        id: ip
        group: ip
        search: ip
        attributes: [ "os", "isp", "asn", "lastUpdate", "hostLocationCity", "hostLocationRegionCode", "hostLocationAreaCode", "hostLocationLatitude", "hostLocationLongitude", "hostLocationCountry", "hostLocationCountryCode", "hostLocationCountryCode3", "hostLocationPostal", "hostLocationDMA", "shodanCreditsLeft" ]

    to:
        id: organization
        group: institution
        search: institution
        attributes: [ "isp", "asn", "lastUpdate", "hostLocationCity", "hostLocationRegionCode", "hostLocationAreaCode", "hostLocationLatitude", "hostLocationLongitude", "hostLocationCountry", "hostLocationCountryCode", "hostLocationCountryCode3", "hostLocationPostal", "hostLocationDMA", "shodanCreditsLeft" ]

    edge:
        attributes: [ "shodanCreditsLeft" ]

  -
    from:
        id: ip
        group: ip
        search: ip
        attributes: [ "domain", "shodanCreditsLeft" ]

    to:
        id: port
        group: port
        search: port
        attributes: [ "shodanCreditsLeft" ]

    edge:
        label: listening on
        attributes: [ "product", "version", "title", "ssl", "cpe", "banner", "transport", "domain", "timestamp" ,"deviceType", "data", "opts", "shodanCreditsLeft" ]

  -
    from:
        id: ip
        group: ip
        search: ip
        attributes: [ "shodanCreditsLeft" ]

    to:
        id: vulnerability
        group: vulnerability
        search: vulnerability
        attributes: [ "shodanCreditsLeft" ]

    edge:
        attributes: [ "shodanCreditsLeft" ]

  -
    from:
        id: domain
        group: domain
        search: domain
        attributes: [ "tags", "txt", "shodanCreditsLeft" ]

    to:
        id: subdomain
        group: domain
        search: domain
        attributes: [ "type", "lastSeen", "txt", "shodanCreditsLeft" ]

    edge:
        label: subdomain
        attributes: [ "shodanCreditsLeft" ]

  -
    from:
        id: subdomain
        group: domain
        search: domain
        attributes: [ "type", "lastSeen", "shodanCreditsLeft" ]

    to:
        id: ip
        group: ip
        search: ip
        attributes: [ "shodanCreditsLeft" ]

    edge:
        attributes: [ "shodanCreditsLeft" ]

  -
    from:
        id: bid
        group: vulnerability
        search: vulnerability
        attributes: [ "related_source", "shodanCreditsLeft" ]

    to:
        id: id
        group: vulnerability
        search: vulnerability
        attributes: [ "description", "vulnerability_source", "author", "code", "date", "platform", "port", "type", "privileged", "rank", "version", "shodanCreditsLeft" ]

    edge:
        label: related
        attributes: [ "shodanCreditsLeft" ]

  -
    from:
        id: cve
        group: vulnerability
        search: vulnerability
        attributes: [ "related_source", "shodanCreditsLeft" ]

    to:
        id: id
        group: vulnerability
        search: vulnerability
        attributes: [ "description", "vulnerability_source", "author", "code", "date", "platform", "port", "type", "privileged", "rank", "version", "shodanCreditsLeft" ]

    edge:
        label: related
        attributes: [ "shodanCreditsLeft" ]

  -
    from:
        id: msb
        group: vulnerability
        search: vulnerability
        attributes: [ "related_source", "shodanCreditsLeft" ]

    to:
        id: id
        group: vulnerability
        search: vulnerability
        attributes: [ "description", "vulnerability_source", "author", "code", "date", "platform", "port", "type", "privileged", "rank", "version", "shodanCreditsLeft" ]

    edge:
        label: related
        attributes: [ "shodanCreditsLeft" ]

  -
    from:
        id: osvdb
        group: vulnerability
        search: vulnerability
        attributes: [ "related_source", "shodanCreditsLeft" ]

    to:
        id: id
        group: vulnerability
        search: vulnerability
        attributes: [ "description", "vulnerability_source", "author", "code", "date", "platform", "port", "type", "privileged", "rank", "version", "shodanCreditsLeft" ]

    edge:
        label: related
        attributes: [ "shodanCreditsLeft" ]
```