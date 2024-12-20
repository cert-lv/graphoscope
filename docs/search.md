1. [Syntax](#syntax)
2. [Common queries](#common-queries)
3. [Extended usage of basic APIs](#extended-usage-of-basic-apis)
4. [Field names autocomplete](#field-names-autocomplete)
5. [Common fields](#common-fields)
6. [Hide visible nodes](#hide-visible-nodes)
7. [Filters](#filters)
8. [Too much results](#too-much-results)
9. [Faster search](#faster-search)
10. [Large list of indicators](#large-list-of-indicators)
11. [Direct API usage](#direct-api-usage)
12. [Limit the amount of returned data](#limit-the-amount-of-returned-data)
13. [Order of returned data](#order-of-returned-data)
14. [Show partial search results](#show-partial-search-results)
15. [Output format](#output-format)


![datasources](assets/img/datasources.png)

A search bar is the main way to interact with the data sources that are running in the background. Just like with the other search engines, type what you are looking for, select the needed source from a dropdown and press the search button.


## Syntax

It's important to follow the correct query syntax. In general it is SQL-based, so a common query could look like:
```sql
FROM database WHERE field_a='example.com' OR (field_b LIKE 'example%' AND field_c=105)
```
... where:
- `database` is a data source to search in. `global` is a special keyword to request all allowed data sources. In the `Administration` documentation section it's explained that some data sources can be extremely slow, therefore to prevent every single request from being slow, some sources are excluded from the `global` space.
- `field_*=value` is the field that the user is interested in, as well as its required value. Wildcards are also possible with a `LIKE` operator

If the data source dropdown is being used, you can skip the `FROM database WHERE` part.


## Common Web GUI queries

Search for a specific field and value:
```sql
domain = 'example.com'
```

Any value is less then something:
```sql
count < 10
```

Any value is more or equal then something:
```sql
count >= 30
```

Wildcard search:
```sql
ip LIKE '8.8.8.%'
```
... here `ip` field's value must start with a `8.8.8.` and may continue with any string.


Search by multiple fields with combining `OR` or `AND` operators:
```sql
field_a = 'example.com' OR field_b = 5
```

Group fields with parenthesis to manipulate the order of processing and build more complex queries:
```sql
field_a = 'example.com' OR ((field_b = 5 AND field_c = 21) OR field_d = 'english')
```

Provide an array to search a value in:
```sql
domain IN ('example.lv', 'example.com')
```

Select all values in between of two values:
```sql
datetime BETWEEN '2020-08-30T12:27:50.447767Z' AND '2020-09-02T12:27:50.447767Z'
```

Exclude from results by field value:
```sql
domain <> 'example.com'
```


## Extended usage of basic APIs

Some data sources support single `field=value` queries only, but if connected properly it's possible to use queries like `... OR ... OR ...` or `field IN ('...', '...')`, which will be splitted into multiple independent single queries in a background. Check for `supportsSQL: true/false` setting.


## Field names autocomplete

Any query have to contain at least one field to search the given value in. Autocomplete makes it easier to find and type the correct data source's field name. In a search bar type at least one character and press a `Tab` key, all fields from all data sources that start with the given characters will make a dropdown (or type nothing to get all the possible fields):

![autocomplete](assets/img/autocomplete.png)

Use arrow keys and `Enter` or mouse to choose the needed item.

Autocomplete works even if the field is not at the end of the query. Simply place a cursor at some place, where field name is incomplete and press a `Tab` key. When needed item is selected a word under a cursor will be replaced.

When there is a need to get all the possible fields - make sure there is no text on both sides of the cursor, so there is no attempt to autocomplete it.


## Common fields

Specific fields can recognizable by all data sources, for example:
- **ip**
- **domain**
- **timestamp**

That means you can query `ip=10.10.1.1` even if the data source does NOT contain such field, but something like `host_address`. Other fields can be taken from a specific source.

`NOTE:` This feature has to be configured by the administrator in the data sources YAML definition files.


## Hide visible nodes

Sometimes you want to hide some already existing graph elements - not needed, sensitive etc.

If the amount is small - deleting one by one is acceptable. To automate the process a special query can be constructed, which doesn't search in any data source, but just hides existing visible nodes. The syntax is:
```sh
NOT field='value'
```
... where `NOT` keyword instructs the system to create a hiding filter:

![red-filter](assets/img/red-filter.png)

That means `green` filters request new data and `red` hide existing elements.


## Filters

After a successful search (if were no errors) a new colored filter below a search input appears. It contains 3 action buttons:

- Copy filter's request to the search input. Useful when new request is similar to the existing one
- Hide/show all graph elements received by this filter. Does not hide elements attached to the other filters too
- Delete filter and related graph changes. Also skips elements attached to the other filters too


## Too much results

Users can control the amount of entries each data source should return. Check the `Profile` section to set the limit. If server-side limit is smaller than user's limit - query will be edited according to the server-side limit.

Such limit prevents returning billions of entries, what makes the graph much cleaner. In cases when any data source **can** return more data, but limit is exceeded - it returns statistics about data found:

![stats](assets/img/stats.png)

From the charts it's possible to get an idea about interesting (or not) things. Right click opens new options to include/exclude them - it produces a new query which should return less data. Continue shrinking requested data until limit is not exceeded.


## Faster search

Sometimes you have many indicators to query and it's inefficient enough to manually write correct syntax with tens of `... OR ...`. In such cases `Format the request` button (left from a data sources dropdown) can help.

Steps to follow:

1. In a search bar type comma or space separated indicators. Example:
```
8.8.8.8,example.com,example@example.com
```
2. Press the `Format the request` button
3. The search input now contains a valid query:
```sql
ip='8.8.8.8' OR domain='example.com' OR email='example@example.com'
```

The system accepts and recognizes the following data types at the moment:

- IP address
- Domain
- Email


## Large list of indicators

To test thousands of indicators - upload a specially formatted text file and wait for a notification when processing is complete. Check `Profile` &rarr; `Actions` for more info and a required data format.


## Direct API usage

API returns a JSON formatted data - it can be queried without a Web GUI. For example with `curl`:
```sh
curl -XGET 'https://server/api?uuid=09e545f2-3986-493c-983a-e39d310f695a&sql=FROM+people+WHERE+age>30'
```

... where `uuid` is user's unique auth UUID, can be found in a `Profile` &rarr; `Account` page.


## Limit the amount of returned data

Direct queries can include SQL feature `LIMIT 0,10` to limit and select the needed data. In Web GUI it can be set at `Profile` &rarr; `Limit` option, when using API it's a part of the SQL query.

However, `LIMIT` is not a total amount of graph nodes or edges - it goes to the data source as a part of the query, for example PostreSQL. When the needed amount of entries is returned Graphoscope displays unique edges according to the data sources definitions. That means `LIMIT 3` can produce:

- only **1** edge when all 3 data source's entries are identical
- **9** edges when all 3 data source's entries are unique and each one is splitted into 3 graph edges, like `Age` &rarr; `Name` &rarr; `City` &rarr; `Country`


## Order of returned data

Direct queries also can include `ORDER BY field` or `ORDER BY field DESC`. The default sorting method is alphabetical. However, `DESC` can be used for displaying rows in descending order.


## Show partial search results

When search results limit exceeded - partial results can be displayed. By default only statistics charts are displayed to update the query when possible. But in some cases you may want to see partial results too.

Example in API query:
```sh
curl -XGET 'https://server/api?uuid=09e545f2-3986-493c-983a-e39d310f695a&show_limited=true&sql=FROM+people+WHERE+age>30'
```
... where `show_limited=true` parameter enables or disables partial results.


## Output format

By default JSON is used to represent graph relations data. However, sometimes you may need to display data as a table. In such cases the output formatting feature can be used.`json` or `table` are currently supported.

Example in API query:
```sh
curl -XGET 'https://server/api?uuid=09e545f2-3986-493c-983a-e39d310f695a&format=table&sql=FROM+people+WHERE+age>30'
```
note the added `format=table`, or select from a dropdown when uploading the indicators file.

API json output example:
```json
{
    "relations": [
        {
            "edge": {
                "label": "lives in"
            },
            "from": {
                "attributes": {
                    "age": "35"
                },
                "group": "name",
                "id": "Monica",
                "search": "name"
            },
            "source": "sample",
            "to": {
                "group": "country",
                "id": "Canada",
                "search": "country"
            }
        },

        ...
```

API table output example:
```
+--------+------------+------------+---------+----------+---------+---------------------+
| SOURCE | EDGE LABEL | FROM GROUP | FROM ID | TO GROUP |  TO ID  | FROM ATTRIBUTES AGE |
+--------+------------+------------+---------+----------+---------+---------------------+
| sample | lives in   | name       | Monica  | country  | Canada  |                  35 |
| sample |            | name       | Monica  | name     | Kate    |                     |
| sample | lives in   | name       | Ben     | country  | Japan   |                  41 |
| sample |            | name       | Ben     | name     | Kate    |                     |
| sample |            | name       | Ben     | name     | John    |                     |
| sample |            | name       | Ben     | name     | Chin    |                     |
| sample | lives in   | name       | Sofy    | country  | Sweden  |                  51 |
| sample |            | name       | Sofy    | name     | Tom     |                     |
| sample | lives in   | name       | Tom     | country  | Sweden  |                  55 |
| sample |            | name       | Tom     | name     | Sofy    |                     |
| sample |            | name       | Tom     | name     | Chin    |                     |
| sample | lives in   | name       | Chin    | country  | China   |                  61 |
| sample |            | name       | Chin    | name     | Tom     |                     |
| sample |            | name       | Chin    | name     | Ben     |                     |
+--------+------------+------------+---------+----------+---------+---------------------+
```
... where:

- **EDGE** describes a relation and its attributes
- **FROM** describes a relation's `From` node
- **TO** describes a relation's `To` node
- **SOURCE** is a source data comes from
