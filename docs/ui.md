1. [Top bar](#top-bar)
2. [Search bar](#search-bar)
3. [Search actions](#search-actions)
4. Data source selection dropdown
5. Buttons to search, clear input, show some query examples
6. [Time period](#time-period)
7. [Dashboards](#dashboards)
8. Input to save a new dashboard
9. [Filters](#filters)
10. [Graph area](#graph-area)
11. Graph image manipulating
12. List of visible nodes groups
13. Graph extending and manipulating
14. [Elements attributes](#elements-attributes)
15. [Users notes](#users-notes)
16. [Main menu](#main-menu)


![ui](assets/img/ui-elements.png#left)


## Top bar

`1` is the same for all screens. Contains a link to the main screen on the left side (logo & project name), username with a site map can be found on the right side.


## Search bar

`2` is the main way to interact with the background data sources. Just like with other search engines, type what you are looking for, press `Enter` or the search button and get results from the chosen databases. Press `Tab` for the current field name autocomplete.

More info in the `Search` documentation section.


## Search actions

`3` allows to skip manual request formatting. Just type in a list of comma or space separated indicators, press the `format` button and it will be automatically converted into a correct SQL query. Usage info in the `Search` documentation section, creation of formatting rules in the `Administration` section.

`4` is a data source selection dropdown, the source to collect data from:
![datasources](assets/img/datasources.png)

`global` is a special keyword to request all allowed data sources. In the `Administration` documentation section it will be explained, that some sources can be extremely slow, so to prevent every single request being slow - some data sources are excluded from the `global` space.

Two informative icons on the right side:
- `SQL not supported` - means this data source accepts basic query only, most likely `field=value`,
- `Datetime range is ignored` - means selected date range does not affect search results.

`5` contains some additional actions:

- The search button
- `2`'s input clearing button
- `Usage` to show some examples of the search requests with a correct SQL syntax. Useful for new users


## Time period

`6` allows the user to select a time period to request data from. Sometimes you are interested in the last hour only, sometimes in the last 6 months. The default value is to search within the last `24 hours`.

Note that larger time period causes larger background data sources load!


## Dashboards

Sometimes you want to save the specific state of the window or requested data, to be able to open it later. By default service will remember your last used filters.

`7` contains all saved dashboards. Both personal and shared. Select from the dropdown and press `Load` or `Delete` to remove the unnecessary one. When loading - user is redirected to the unique URL for a direct access to the dashboard.

`8` allows the user to save a new dashboard. Type in a unique name, set the `Share` checkbox if you want it to be publicly available and press the `Save` button.


## Filters

`9` is the place where all new filters appear after something was requested. Each one has 3 action buttons:

- Copy filter's request to the `2` input. Useful when the new request is similar to the existing one
- Hide all graph elements received by this filter. Skips elements attached to the other filters too
- Delete filter and related graph elements. Also skips elements attached to the other filters too

![filters](assets/img/filters.png)

... here the left filter is disabled and the right one is enabled.


## Graph area

`10` is the main area to use - the graph area. When the request is successful, received data is being visualized as graph nodes with edges between them.

Every element is interactive and can contain attributes. `Left click` on any one to get more info about it or move it. The whole graph area is movable also. Releasing left click stops all animations - useful when positions calculations/animations are taking too much time.

Any graph element can be removed or extended. `Right click` gives you additional graph manipulating options `13`:

- Search by a selected node(s) to try to find its neighbors
- Cluster neighbors by group. List of available groups is being generated automatically. Allows to hide a large amount of nodes of the same type. Useful when graph looks too heavy. Resulting cluster can be opened or deleted later. Left clicking it shows all the internal elements on the right side of the screen
- Find selected nodes common attributes or neighbors
- Delete selected node(s)

It is also possible to delete nodes with a `Delete` key, just make sure the graph area is in focus, but not some input field. Click on the graph area if it was not in focus.

Example of the cluster `emails` (red) and the list of contained emails (blurred on the right side):

![cluster](assets/img/cluster.png)


`11` buttons allow to:

- Center graph. Useful when it was accidentally moved out of the visible area.
- Save the graph as an image.
- Export not hidden graph elements as a text file. Useful for providing as a report.
- Set to full screen. Sometimes using the whole browser area is more comfortable.


`12` is a passive list of not hidden nodes groups. Every tag shows the number of related nodes.


## Elements attributes

Every graph element is interactive and can contain attributes. To find more info about graph node or edge - left click on it, a table of attributes will appear on the right side of the screen `14`.


## Users notes

Users can set custom comments for the graph elements. Click on any node/edge - **Notes** section `15` below the attributes will appear. Enter any text and press `Save` button. Saved notes will be visible to all the users. To remove the note - save an empty value.


## Main menu

`16` is a main menu, where user can access his profile settings, service administration, user management and a built-in documentation. Some options are available to the administrators only.
