# MISP plugin

Plugin allows to query a MISP instance.
Can search for attributes by [type/]value and for events by ID/UUID, with or without datetime range.

To search for event:
```sql
FROM misp WHERE event='000000'
FROM misp WHERE event='00000000-0000-0000-80f3-8e92723639a8'
```
To search for attribute of any type:
```sql
FROM misp WHERE attribute='8.8.8.8'
```
To search for attribute of specific type and datetime range:
```sql
FROM misp WHERE hostname='example.com' and datetime BETWEEN '2024-05-04T11:30:14.000Z' AND '2024-06-04T11:30:14.000Z'
```

More info at:
https://www.misp-project.org/openapi/#tag/Attributes/operation/restSearchAttributes
https://www.misp-project.org/openapi/#tag/Events/operation/restSearchEvents

`curl` to test:
```sh
curl 'https://localhost:443/api?uuid=auth-key&sql=FROM+misp+WHERE+attribute=%278.8.8.8%27'
```

Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o misp.so ./*.go
```

# Access details

Source YAML definition's `access` fields:
- **protocol**: "https" or "http"
- **host**: instance's hostname
- **apiKey**: user's unique API access key
- **caCertPath**: CA file path
- **certPath**: certificate file path
- **keyPath**: key file path


# YAML example

As MISP has a very large amount of different attribute types, graph relations are generated on the fly, no need to put them all in a YAML config. So it is enough to start with:
```
name: misp
label: MISP
icon: share square

plugin: misp
inGlobal: true
includeDatetime: true
supportsSQL: false

access:
    protocol: https
    host: misp.example.com
    apiKey: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
    caCertPath: certs/ca.crt
    certPath: certs/misp.crt
    keyPath: certs/misp.key

queryFields: ["event", "attribute", "md5", "sha1", "sha256", "filename", "pdb", "filename|md5", "filename|sha1", "filename|sha256", "ip-src",
              "ip-dst", "hostname", "domain", "domain|ip", "email", "email-src", "eppn", "email-dst", "email-subject", "email-attachment",
              "email-body", "float", "git-commit-id", "url", "http-method", "user-agent", "ja3-fingerprint-md5", "jarm-fingerprint", "favicon-mmh3",
              "hassh-md5", "hasshserver-md5", "regkey", "regkey|value", "AS", "snort", "bro", "zeek", "community-id", "pattern-in-file",
              "pattern-in-traffic", "pattern-in-memory", "pattern-filename", "pgp-public-key", "pgp-private-key", "yara", "stix2-pattern", "sigma",
              "gene", "kusto-query", "mime-type", "identity-card-number", "cookie", "vulnerability", "cpe", "weakness", "attachment",
              "malware-sample", "link", "comment", "text", "hex", "other", "named pipe", "mutex", "process-state", "target-user", "target-email",
              "target-machine", "target-org", "target-location", "target-external", "btc", "dash", "xmr", "iban", "bic", "bank-account-nr",
              "aba-rtn", "bin", "cc-number", "prtn", "phone-number", "threat-actor", "campaign-name", "campaign-id", "malware-type", "uri",
              "authentihash", "vhash", "ssdeep", "imphash", "telfhash", "pehash", "impfuzzy", "sha224", "sha384", "sha512", "sha512/224",
              "sha512/256", "sha3-224", "sha3-256", "sha3-384", "sha3-512", "tlsh", "cdhash", "filename|authentihash", "filename|vhash",
              "filename|ssdeep", "filename|imphash", "filename|impfuzzy", "filename|pehash", "filename|sha224", "filename|sha384",
              "filename|sha512", "filename|sha512/224", "filename|sha512/256", "filename|sha3-224", "filename|sha3-256", "filename|sha3-384",
              "filename|sha3-512", "filename|tlsh", "windows-scheduled-task", "windows-service-name", "windows-service-displayname",
              "whois-registrant-email", "whois-registrant-phone", "whois-registrant-name", "whois-registrant-org", "whois-registrar",
              "whois-creation-date", "x509-fingerprint-sha1", "x509-fingerprint-md5", "x509-fingerprint-sha256", "dns-soa-email", "size-in-bytes",
              "counter", "datetime", "port", "ip-dst|port", "ip-src|port", "hostname|port", "mac-address", "mac-eui-64", "email-dst-display-name",
              "email-src-display-name", "email-header", "email-reply-to", "email-x-mailer", "email-mime-boundary", "email-thread-index",
              "email-message-id", "github-username", "github-repository", "github-organisation", "jabber-id", "twitter-id", "dkim",
              "dkim-signature", "first-name", "middle-name", "last-name", "full-name", "date-of-birth", "place-of-birth", "gender",
              "passport-number", "passport-country", "passport-expiration", "redress-number", "nationality", "visa-number",
              "issue-date-of-the-visa", "primary-residence", "country-of-residence", "special-service-request", "frequent-flyer-number",
              "travel-details", "payment-details", "place-port-of-original-embarkation", "place-port-of-clearance",
              "place-port-of-onward-foreign-destination", "passenger-name-record-locator-number", "mobile-application-id", "chrome-extension-id",
              "cortex", "boolean", "anonymised"]

statsFields:
    - Category
    - Org
    - Orgc
    - Published
    - Distribution
    - ToIDS

replaceFields:
    domain: attribute
    ip:     attribute
    email:  attribute
```
