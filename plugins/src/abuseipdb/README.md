# AbuseIPDB plugin

AbuseIPDB connector sends IP address in a GET request to the `abuseipdb.com` API and expects a list of reports back.
More info: https://docs.abuseipdb.com/#check-endpoint


Simple request to the API:
```
curl -G https://api.abuseipdb.com/api/v2/check \
  --data-urlencode "ipAddress=8.8.8.8" \
  -d maxAgeInDays=90 \
  -d verbose \
  -H "Key: YOUR_OWN_API_KEY" \
  -H "Accept: application/json"
```
where `YOUR_OWN_API_KEY` is your personal/unique API key.


`curl` to test plugin:
```sh
curl 'https://localhost:443/api?uuid=auth-key&sql=FROM+abuseipdb+WHERE+ip=%278.8.8.8%27'
```

Compile with:
```sh
go build -buildmode=plugin -ldflags="-w" -o abuseipdb.so ./*.go
```

# Access details

Source YAML definition's `access` fields:
- **url**: HTTPS access point, `https://api.abuseipdb.com/api/v2/check` at the moment
- **maxAgeInDays**: how far back in time we go to fetch reports, max 365
- **key**: unique API key
