# Recorded Future plugin

Connector sends GET requests to the `https://api.recordedfuture.com/v2/` API endpoints.
Request can contain `ip`, `domain`, `hash`, `malware`, `url`, `vulnerability`, `company`
fields only - configurable in a YAML file as `queryFields`.
List depends on your subscription level.

`curl` to test:
```sh
curl 'https://localhost:443/api?uuid=auth-key&sql=FROM+recorded_future+WHERE+ip=%278.8.8.8%27'
```

Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o recorded_future.so ./*.go
```

# Limitations

Does not support complex SQL queries and datetime range selection.


# Access details

Source YAML definition's `access` fields:
- **server**: API server, for example - `https://api.recordedfuture.com/v2/`
- **token**: User's access token


# Definition file example

Replace API token with your own:
```yaml
name: recorded_future
label: Recorded Future
icon: database

plugin: recorded_future
inGlobal: false
includeDatetime: false
supportsSQL: false

access:
    server: https://api.recordedfuture.com/v2/
    token: ................................

queryFields:
    - ip
    - domain
    - hash
    - malware
    - url
    # - vulnerability
    # - company


relations:
  -
    from:
        id: rule
        group: rule
        search: rule
        attributes: ["evidenceCriticality", "evidenceCriticalityLabel"]

    to:
        id: ip
        group: ip
        search: ip
        attributes: ["criticalityLabel", "riskString", "rules", "criticality", "riskSummary", "score", "mitigationString", "evidenceString", "aiInsightsComment", "aiInsights", "firstSeen", "lastSeen"]

    edge:
        attributes: ["timestamp"]

  -
    from:
        id: rule
        group: rule
        search: rule
        attributes: ["evidenceCriticality", "evidenceCriticalityLabel"]

    to:
        id: domain
        group: domain
        search: domain
        attributes: ["criticalityLabel", "riskString", "rules", "criticality", "riskSummary", "score", "mitigationString", "evidenceString", "aiInsightsComment", "aiInsights", "firstSeen", "lastSeen"]

    edge:
        attributes: ["timestamp"]

  -
    from:
        id: rule
        group: rule
        search: rule
        attributes: ["evidenceCriticality", "evidenceCriticalityLabel"]

    to:
        id: hash
        group: hash
        search: hash
        attributes: ["criticalityLabel", "riskString", "rules", "criticality", "riskSummary", "score", "mitigationString", "evidenceString", "aiInsightsComment", "aiInsights", "firstSeen", "lastSeen"]

    edge:
        attributes: ["timestamp"]

  -
    from:
        id: rule
        group: rule
        search: rule
        attributes: ["evidenceCriticality", "evidenceCriticalityLabel"]

    to:
        id: url
        group: url
        search: url
        attributes: ["criticalityLabel", "riskString", "rules", "criticality", "riskSummary", "score", "mitigationString", "evidenceString", "aiInsightsComment", "aiInsights", "firstSeen", "lastSeen"]

    edge:
        attributes: ["timestamp"]

  -
    from:
        id: malwareID
        group: infection
        search: malware
        attributes: ["MalwareCategory", "aiInsightsComment", "aiInsights", "firstSeen", "lastSeen"]

    to:
        id: malware
        group: infection
        search: malware
        attributes: ["MalwareCategory", "aiInsightsComment", "aiInsights", "firstSeen", "lastSeen"]

    edge:
        label: id
```
