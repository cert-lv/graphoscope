1. [Set administrators](#set-administrators)
2. [Global graph settings](#global-graph-settings)
3. [User management](#user-management)
4. [Actions](#actions)
5. [Demo data](#demo-data)
6. [New data source](#new-data-source)
7. [Fields autocomplete](#fields-autocomplete)
8. [Query auto-formatting rules](#query-auto-formatting-rules)
9. [Debug info](#debug-info)
10. [Custom graph elements style](#custom-graph-elements-style)
11. [Limit returned data](#limit-returned-data)
12. [Plugins development](#plugins-development)


After a fresh installation the service's environment is `development` - all users have the same highest level rights. Therefore the first step is to set administrators.


## Set administrators

1. In the top-right corner click on `Options` and select `Administration`
2. Select `Users` tab:

![users](assets/img/users.png)

3. Initially you have just one user - enable `Is admin` to make that user an administrator. The same option can be applied to the other users to make them administrators.


Now service is ready to run in a production mode. Open `graphoscope.yaml` file, set `environment: prod` and restart the service.


## Global graph settings

`Settings` tab contains many settings related to the styling of nodes/edges and interaction with graph elements. Some individual settings can be found in a `Profile` section.


## User management

In the same window you can:

- **Reset user's password** - the user will now be able to sign up with the same username again, what is now allowed with a non-empty password.
- **Delete user**
- Give or remove **admin rights**


## Actions

On `Actions` tab some actions can be made without restarting the service. For example: reload collectors.


## Demo data

After a production environment installation is completed it includes a CSV file as a first demo data source:

- CSV file with all the data included, headers in the first line - `/opt/graphoscope/files/demo.csv` (`files/demo.csv` in sources)
- Graphoscope data source definition - `/opt/graphoscope/sources/demo.yaml`

... simple list of 10 people and their friends. The first query could be requesting all people with an age over **30**:
```sql
FROM demo WHERE age > 30
```

The results will be similar to:

![results](assets/img/results.png)

Now it's possible to extend the graph by searching for more of John's neighbors - right click on `John` and choose `Search Demo`. We find that **Jennifer** and **Kate** also are his friends:

![results-extended](assets/img/results-extended.png)


At this point users can continue reading **UI** and **Search** documentation sections.


## New data source

To add a new data source the only thing that is required is to add its definition in `sources/` directory. Definition step by step:

Data source name to query by users, its Web GUI display label and icon:

```yaml
name: demo
label: Demo
icon: database
```
Plugin to use:
```yaml
plugin: file-csv
```
Whether this source should be queried when `global` namespace is requested:
```yaml
inGlobal: false
```
... some sources can be very slow. To prevent every request being slow - you can exclude such sources from the `global` namespace.
Whether data source should process `datetime` requested range, which is always added by the Web GUI:
```yaml
includeDatetime: false
```
... some data sources don't have a timestamp field, so no data will ever be returned.
```yaml
supportsSQL: true
```
... whether the data source supports SQL features. Access details:
```yaml
access:
    path: files/demo.csv
```
List of relations to create as Graphoscope can't guess the logic behind random data structure:
```yaml
relations:
  -
    from:
        id: name
        group: name
        search: name
        attributes: [ "age" ]

    to:
        id: country
        group: country
        search: country

    edge:
        label: lives in
        attributes: []

  -
    from:
        id: name
        group: name
        search: name

    to:
        id: friend
        group: name
        search: name
```

For more parameters and details check example file `sources/source.yaml.example`.


## Fields autocomplete

How autocomplete works in a background:

1. Service is started
2. Connection is established to the each data source (if it's required)
3. Each data source returns a list of available fields to query
4. A global list of unique fields is being created and returned to the Web GUI

There are two ways to get a list of field names of the data source:

- Automatically. Such plugins know how to query a data source and return a list of fields
- Manually. As from some data sources there is no way to get all the possible fields - such list can be created manually by an administrator. Check plugin's README for more info. Then in a data source's YAML definition file use a `queryFields` setting to define all the possible fields:
```
queryFields:
    - address
    - domain
```


## Query auto-formatting rules

`format` button converts a list of comma or space separated indicators into a correct SQL query. To allow the service to understand the type (group) of each indicator formatting rules must be created: `files/formats.yaml.example` can be used as an example and shows the rules for some groups by default.

Syntax:
```yaml
indicator's group:
    - regexp 1 to detect it
    - regexp 2 to detect it
    - ...
```
Example:
```yaml
email:
    - ^[\%\w\.+-]+@[\%\w\.]+\.[\%a-zA-Z]{1,9}$
```

To add a non-default group - append it's name to the YAML file with a list of regexps to detect it and restart the service.


## Debug info

During queries several things happen in a background like SQL to Elasticsearch JSON query conversion, fields name adaptation, etc. Each plugin can save progress information and return to the user. Disabled by default, can be enabled in profile settings. Accessible in a browser's console.


## Custom graph elements style

It's possible to customize the style of graph nodes. The previous data source definition contained `group: name` and `group: country` - two styling groups, similar to the CSS classnames. To set your own styles change directory to the service's location and create a new file based on the example:
```sh
cd /opt/graphoscope/
ls files/groups.json
```
or in a dev. environment:
```sh
cd /opt/go/src/github.com/cert-lv/graphoscope
cp files/groups.json.example files/groups.json
```

Open `groups.json` in a text editor and insert:
```json
{
    "name": {
        "shape": "dot",
        "color": {
            "background": "#fc3",
            "border": "#da3"
        }
    },
    "country": {
        "shape": "diamond",
        "color": {
            "background": "#f22",
            "border": "#c22"
        },
        "font": {
            "color": "#04a"
        }
    },
    "cluster": {
        "shape": "hexagon",
        "size": 25,
        "color": {
            "background": "#777b7b",
            "border": "#566"
        },
        "font": {
            "color": "#ccc"
        }
    }
}
```

Here we describe both groups - shapes and all kinds of colors. `cluster` is a built-in group for the clusters - when you combine all the same type neighbors into a one cluster node, to make the picture cleaner. Restart the service, reload web page and see the difference:

![results-styled](assets/img/results-styled.png)

Possible shapes and more styling options at [https://visjs.github.io/vis-network/docs/network/nodes.html](https://visjs.github.io/vis-network/docs/network/nodes.html).

Font icons and images also can represent nodes. A JavaScript framework that is being used includes a complete port of **Font Awesome 5.13.0**. `groups.json` content example to use both image and font icon:
```json
...
"name": {
    "shape": "icon",
    "icon": {
      "face": "Icons",
      "weight": "bold",
      "code": "\uf0c0",
      "size": "30",
      "color": "#0d7"
    },
    "font": {
        "color": "#05b"
    }
},
"country": {
    "shape": "image",
    "image": "assets/img/logo.svg"
},
...
```
Result:

![nodes-images](assets/img/nodes-images.png)

A cheatsheet of possible icons codes: [https://fontawesome.com/cheatsheet](https://fontawesome.com/cheatsheet). Limitations and tips:

- Can use `circularImage` shape instead of `image` to make an image to be a circle. Example: [https://visjs.github.io/vis-network/examples/network/nodeStyles/circularImages.html](https://visjs.github.io/vis-network/examples/network/nodeStyles/circularImages.html)
- Images must be uploaded first. For example, to the `assets/img/icons`
- Selected nodes do NOT change their background color
- Size of the font icon will always stay the same, independently of the neighbors count


## Limit returned data

In a `graphoscope.yaml` there is a setting `limit: X` - max amount of returned entries from each data source. It prevents returning billions of entries and makes graph much cleaner. If any data source can return more entries - statistics info will be returned about these limited entries, so user is able to improve the query.


## Plugins development

Every new and unique data source is different from the previous - communication methods, data structure, etc. That means technical implementation will also be different. In Graphoscope it is done with the help of plugins - one for each unique data source. For example, a `MongoDB` plugin, or `Elasticsearch` - allow to connect to these specific databases.

Existing built-in plugins can be found in a `plugins/src/` directory. One plugin - one directory.

When there is a need to use a new data source - new plugin must be developed. Step-by-step workflow:

1. Move to the plugins directory:
```sh
cd $GOPATH/src/github.com/cert-lv/graphoscope/plugins/src
```
2. Make a copy of the template:
```sh
cp -r template <plugin-name>
cd <plugin-name>
```
... real plugin name should be instead of `<plugin-name>`.
3. Rename an entry point and testing files:
```sh
mv template.go <plugin-name>.go
mv template_test.go <plugin-name>_test.go
```
4. Follow the steps in the source files:

Edit `<plugin-name>.go`
  - **STEP 1** - validate required parameters or settings, given by the user
  - **STEP 2** - create a connection to the data source if needed, check whether it is established. For example, `MongoDB` requires an established connection, while `HTTP REST API` does not
  - **STEP 3** - store plugin settings, like "client" object, URL, database name, etc.
  - **STEP 4** - get a list of all known data source's fields for the Web GUI autocomplete
  - **STEP 5** - when new query is launched - an SQL statement conversion must be done, so the data source can understand what client is searching for. Created query should be added to the debug info, so admin or developer can see what happens in a background.

Edit `convert.go`
  - **STEP 6** - do the SQL conversion. Check, for example, a `MongoDB` plugin to see how SQL can be converted to the hierarchical object or an `HTTP` plugin where you get just a list of requested `field/value` pairs

Edit `<plugin-name>.go`
  - **STEP 7** - run the query and get the results. Implementation depends on the data source's methods. In an `HTTP` plugin it's just a GET/POST request
  - **STEP 8** - process data returned by the data source. Most of this loop content you shouldn't modify at all
  - **STEP 9** - gracefully stop the plugin when main service stops, drop all connections correctly

Edit `plugin.go`
  - **STEP 10** - define all the custom fields needed by the plugin, such as "client" object, database/collection name, etc. See the `STEP 3`
  - **STEP 11** - set plugin name and version

Edit `README.md`
  - **STEP 12** - white plugin description and documentation if needed

Edit `<plugin-name>_test.go`
  - **STEP 13** - test whether example SQL queries are correctly converted to the expected format

During all these steps you can use existing plugins as the working examples.


Now test and compile the source code:
```sh
cd $GOPATH/src/github.com/cert-lv/graphoscope
go test plugins/src/<plugin-name>/*.go
go build -buildmode=plugin -ldflags="-w" -o plugins/<plugin-name>.so plugins/src/<plugin-name>/*.go
```

For a prod. environment make sure `Makefile` is edited according to your needs (`REMOTE` variable) and append to the `Dockerfile`'s section `STEP 1`:
```sh
RUN go build -buildmode=plugin -ldflags="-w" -o /go/plugins/<plugin-name>.so plugins/src/<plugin-name>/*.go
```
... real plugin name should be instead of `<plugin-name>`, and run:
```sh
make compile
make update
```

Docker images will be created to compile plugins and the main service, after `update` remote files will be updated.


To make Golang plugins work and be compatible - all components must be compiled in the identical environments. So a specific Golang docker image is used. The same thing with `GOROOT`/`GOPATH` variables. `CGO_ENABLED=1` env. variable also is required.

When YAML description file is prepared, it is enough to restart the service.
