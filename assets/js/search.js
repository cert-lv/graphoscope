/*
 * Search functionality.
 *
 * Allows to format comma/space separated list of indicators,
 * display short usage info with the common queries examples,
 * query all connected data sources and display the results -
 * graph relations, statistics info or an error.
 */
class Search {
    constructor(application) {
        // Pointer to the main page core
        this.application = application;

        // Web GUI elements for the easier access
        this.input =      document.getElementById('search_input');
        this.source =     $('.source.dropdown');
        this.formatBtn =  $('.ui.format.button');
        this.searchBtn =  $('.ui.search.button');

        // SQL autocomplete
        this.autocomplete = new SQLAutocomplete(this, this.input);

        // Amount of currently running searches
        this.jobs = 0;

        // Bind Web GUI buttons
        this.bind();

        // Prepare regexps for the query formatting
        this.prepareFormatsRe();

        // Init Semantic UI dynamic elements
        this.prepareSemantic();
    }

    /*
     * Bind Web GUI buttons
     */
    bind() {
        // Usage button
        document.getElementById('usage-btn').addEventListener('click', (e) => {
            this.usage();
        });

        // Bind keyboard keys
        this.input.addEventListener('keydown', (e) => {
            var list = this.autocomplete.container.getElementsByTagName('div');

            // Run search with the Enter key
            // or select autocompleted field
            if (e.key === 'Enter') {
                // Simulate a click on the "active" autocomplete item
                // Or run the search query
                if (this.autocomplete.currentFocus > -1)
                    if (list.length !== 0) list[this.autocomplete.currentFocus].click();
                else
                    this.run();

            // Data sources fields autocomplete
            } else if (e.key === 'Tab') {
                this.autocomplete.build(e.target.selectionStart);
                e.preventDefault();

            } else if (e.key === 'ArrowDown') {
                this.autocomplete.setActive(list, 1);
                e.preventDefault();

            } else if (e.key === 'ArrowUp') {
                this.autocomplete.setActive(list, -1);
                e.preventDefault();
            }
        });

        // Format button
        this.formatBtn.on('click', (e) => {
            this.format();
        });

        // Search button
        this.searchBtn.on('click', (e) => {
            this.run();
        });

        // Button to clear all search related elements
        $('.ui.clear.button').on('click', (e) => {
            this.application.graph.clearAll();
        });
    }

    /*
     * Init Semantic UI dynamic elements
     */
    prepareSemantic() {
        $('.ui.dropdown').dropdown();

        // Datetime range selectors
        this.application.calendar.daterange.popup({
            position: 'bottom right',
            on: 'click'
        });
    }

    /*
     * Show queries examples
     */
    usage() {
        const text = '<span class="usage">' +
                         '<span class="header green_fg">Selection queries</span>' +

                         'ip=\'10.10.10.2\'&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span class="grey_fg">// Search for IP</span></br>' +
                         'domain=\'example.com\'&nbsp;&nbsp;&nbsp;<span class="grey_fg">// Search for a domain</span></br>' +
                         'ip LIKE \'8.8.8.%\'&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span class="grey_fg">// Wildcard search</span></br>' +
                         'feed.provider=\'ShadowServer\' AND source.ip=\'10.10.10.1\'&nbsp;&nbsp;<span class="grey_fg">// Multiple fields</span></br>' +
                         'domain IN (\'example.com\',\'google.com\')&nbsp;&nbsp;&nbsp;&nbsp;<span class="grey_fg">// Find any from the list</span></br>' +
                         'size BETWEEN 0 AND 15&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;' +
                         '<span class="grey_fg">// Search for sizes between the given values</span></br>' +
                         'source.institution.name<>\'SIA "Example"\'&nbsp;&nbsp;<span class="grey_fg">// Exclude results by field value</span></br>' +

                         '</br>' +

                         '<span class="header red_fg">Hide nodes</span>' +

                         'NOT name=\'10.10.1.32\'&nbsp;&nbsp;<span class="grey_fg">// By one specific field</span>' +
                     '</span>';

        this.application.modal.ok('USAGE', text);
    }

    /*
     * Prepare regexps for the comma/space separated indicators formatting
     */
    prepareFormatsRe() {
        for (var group in FORMATS) {
            for (var i = 0; i < FORMATS[group].length; i++) {
                FORMATS[group][i] = new RegExp(FORMATS[group][i]);
            }
        }
    }

    /*
     * Format the comma/space separated indicators
     */
    format() {
        const input = this.input.value;

        var query = '',
            parts = input.split(/ *, */);

        // Split by spaces if no commas were found
        if (parts.length === 1)
            parts = input.split(/ +/);

        for (var i = 0; i < parts.length; i++) {
            const part = parts[i];

            // Skip empty values
            if (part === '')
                continue;

            loop: {
                for (var group in FORMATS) {
                    for (var j = 0; j < FORMATS[group].length; j++) {
                        if (FORMATS[group][j].test(part)) {
                            if (part.indexOf('%') === -1)
                                query += group + '=\'' + part + '\' OR ';
                            else
                                query += group + ' like \'' + part + '\' OR ';

                            break loop;
                        }
                    }
                }

                this.application.modal.error('Can\'t format the query!', 'Unknown type of the value: <strong>' + part + '</strong>');
                return;
            }
        }

        this.input.value = query.slice(0, -4);
    }

    /*
     * Build search query and execute
     * or create red filter if the query starts with "NOT "
     */
    run() {
        var query = this.input.value;

        // Skip empty input
        if (query === '')
            return;

        // Check whether it's a red filter
        if (query.substring(0, 4).toLowerCase() === 'not ') {
            this.application.filters.addRed(query);
            return;
        }

        // Add data source name if necessary
        if (query.substring(0, 5).toLowerCase() !== 'from ') {
            query = 'FROM ' + this.source.dropdown('get value') + ' WHERE ' + query;
        } else {
            var parts = query.split(' ');
            parts[0] = 'FROM';   // 'from' to uppercase because 'query()' expects uppercase,
            parts[2] = 'WHERE';  // 'where' to uppercase
            query = parts.join(' ');
        }

        // Hide charts if visible
        this.application.charts.close();

        // Do not run new query if such filter already exists
        for (var id in this.application.filters.greenFilters) {
            if (this.application.filters.greenFilters[id].query === query) {
                this.application.modal.error('Such <span class="green_fg">green</span> filter already exists!', 'Modify search input value and try again!');
                return;
            }
        }

        this.query(query);
    }

    /*
     * Build SQL request from the user's query and send it to the server
     */
    query(query) {
        const xmlHttp = new XMLHttpRequest(),
              search = this;  // Access 'this' from the 'onreadystatechange()'

        // Check whether such filter already exists
        for (var id in this.application.filters.greenFilters) {
            if (this.application.filters.greenFilters[id].query === query) {
                this.application.filters.greenFilters[id].empty = false;
            }
        }

        this.jobs += 1;

        const startTime = this.application.calendar.rangeStart.calendar('get date').toISOString().substr(0, 19) + '.000Z',
              endTime =   this.application.calendar.rangeEnd.calendar(  'get date').toISOString().substr(0, 19) + '.000Z';

        // Put all user fields in one scope
        // to make 'datetime' be always an independent filter.
        // 'replace' changes the first instance only
        var sql = query.replace(' WHERE ', ' WHERE (') + ') AND datetime BETWEEN \'' + startTime + '\' AND \'' + endTime +'\'';

        // Limit is optional
        if (OPTIONS.Limit !== 0)
            sql += ' LIMIT 0,' + OPTIONS.Limit;

        if (OPTIONS.Debug === true ) {
            console.log('%cUser query:', 'font-weight:bold');
            console.log(sql);
        }

        // Send resulting SQL query to the server
        this.application.websocket.send('sql', sql);

        // Disable search button and hide menus if visible
        this.searchBtn.addClass('disabled loading');
        this.application.graph.menu.style.visibility =  'hidden';
        this.application.charts.menu.style.visibility = 'hidden';
    }

    /*
     * Process server's response.
     * Receives also user's initial query
     */
    processResults(query, response) {
        const results = JSON.parse(response),
              query_without_parenthesis = query.replace(' WHERE (', ' WHERE ').substring(0, query.length-2);

        this.jobs -= 1;

        // Wait until all running searches have finished
        if (this.jobs === 0)
            this.searchBtn.removeClass('disabled loading');

        // Skip if zero entries were returned
        if (Object.keys(results).length === 0) {
            this.application.filters.addGreen(query_without_parenthesis);

        } else {
            // Show an error along with the other relations data
            if (results.error !== undefined)
                this.application.modal.error('Server has returned an error!', results.error);

            // Show available results even if some data source has returned an error
            if (results.relations === undefined)
                results.relations = {};

            const id = this.application.filters.addGreen(query_without_parenthesis);
            this.processRelations(id, results.relations);

            // Show stats based on limited relations data
            // to be able to improve the query
            if (results.stats !== undefined)
                this.application.charts.create(query, results.stats, id);

            // Show debug info in browser's console
            if (results.debug !== undefined) {
                console.log('%cDebug info:', 'font-weight:bold');

                for (const source in results.debug) {
                    for (const key in results.debug[source])
                        console.log('%c' + source + ' ' + key + ': %c' + results.debug[source][key], 'color:#ff5f2d', 'color:default');
                }
            }
        }
    }

    /*
     * Process common nodes.
     *
     * Receives standard server's response with edges/stats/error
     * and a list of unique neighbors to display on a Web GUI right panel
     */
    processCommon(response, list) {
        const results = JSON.parse(response),
              values =  JSON.parse(list);

        this.searchBtn.removeClass('disabled loading');

        // Show an error along with the other relations data
        if (results.error !== undefined)
            this.application.modal.error('Server has returned an error!', results.error);

        // Inform that some nodes can't be processed
        if (results.stats !== undefined)
            this.application.modal.error('Too many entries!', '<strong>' + results.stats.source + '</strong> nodes can\'t be processed as data source contains too many entries!');

        // Show available results even if some data source has returned an error
        if (results.relations === undefined)
            results.relations = {};

        this.processRelations(null, results.relations);

        // Fill the right table
        if (values.length > 0) {
            for (var i = 0; i < values.length; i++) {
                const value = values[i]
                $('.ui.common-neighbors.table').append('<tr><td>' + value[0] + '</td><td>' + value[1] + '</td></tr>');
            }
        } else {
            $('.ui.common-neighbors.table').append('<tr><td class="grey_fg">Nothing found!</td><td class="empty"></td></tr>');
        }
    }

    /*
     * Show received nodes and edges.
     *
     * Receives unique filter's ID to create for the current query
     * and relations data
     */
    processRelations(id, relations) {
        //console.log(id, JSON.stringify(relations));

        // XXX: idea to add multiple nodes in 1 step:
        // https://github.com/almende/vis/issues/3278

        // this.application.graph.network.stopSimulation();
        // this.application.graph.network.setOptions({
        //     physics: { enabled: false }
        // });

        // This starts drawing the graph several times faster
        this.application.graph.network.setOptions({
            physics: { enabled:true }
        });

        this.application.graph.network.startSimulation();

        // Add new graph elements
        for (var i = 0; i < relations.length; i++) {
            const entry = relations[i],
                  from = entry.from,
                  to = entry.to,
                  existingEdge = this.application.graph.network.body.edges[from.id + '-' + to.id];

            // Create nodes which don't exist yet
            const nodeFrom = this.addNode(this.application.graph.network.body.nodes[from.id], from, id, entry.source),
                  nodeTo =   this.addNode(this.application.graph.network.body.nodes[to.id],   to,   id, entry.source);

            // Set neighbors group to be able to cluster them
            nodeFrom.options.neighbors[nodeTo.options.group] = true;
            nodeTo.options.neighbors[nodeFrom.options.group] = true;

            // Update the edge if already exists
            if (existingEdge) {
                // Merge attributes.
                // Forced to use 'get()' to access the attributes
                const edge = this.application.graph.network.body.data.edges.get(from.id + '-' + to.id);
                edge.attributes = this.merge(edge.attributes, entry.edge);

                // Increase edge size
                const size = existingEdge.options.width;
                existingEdge.options.width = size + 1 / size * 5;

            // Add an edge if doesn't exist yet
            } else {
                const edge = {};

                // Additional merge to convert all values to strings
                edge.attributes = this.merge({}, entry.edge)
                //edge.attributes = entry.edge || {};
                edge.attributes.source = entry.source;
                edge.id =    from.id + '-' + to.id;
                edge.from =  from.id;
                edge.to =    to.id;
                edge.label = edge.attributes.label || '';

                // TODO: Set edge's style
                // Config file can be used when a feature request is implemented:
                // https://github.com/visjs/vis-network/issues/1229
                //edge.group = entry.source;
                if (entry.source === 'pdns')
                    edge.color = { color: '#09f' };
                else if (entry.source === 'intelmq')
                    edge.color = { color: '#f75' };
                else if (entry.source === 'webapps')
                    edge.color = { color: '#d93' };
                else if (entry.source === 'pass')
                    edge.color = { color: '#086' };

                this.application.graph.network.body.data.edges.add(edge);
            }
        }

        // Recalculate nodes size
        this.application.graph.recalculateSizes();

        // This also auto scales & fits the graph if enabled
        if (OPTIONS.StabilizationTime != 0)
            this.application.graph.network.stabilize(OPTIONS.StabilizationTime);

        // Update tags value
        this.application.tags.update();
    }

    /*
     * Create node from new data.
     *
     * Receives:
     *     existing - node if its value already exists on a graph, 'undefined' otherwise
     *     data     - single node's parameters
     *     filterID - filter's ID this node is related to
     *     source   - data source's name where data comes from
     */
    addNode(existing, data, filterID, source) {
        data.attributes = data.attributes || {};
        data.attributes['source'] = source;

        // Update existing one
        if (existing) {
            existing.options.attributes = this.merge(existing.options.attributes, data.attributes);

            // Nodes as common neighbors do not belong to any filter
            if (filterID)
                existing.options.greenFiltersID[filterID] = true;

            this.application.graph.network.body.data.nodes.update([{ id: existing.id, hidden: this.application.filters.checkDisplayRedNode(existing) }]);

            return existing;

        // Add new node if doesn't exist
        } else {
            // Additional merge to convert all values to strings
            data.attributes = this.merge({}, data.attributes)
            data.attributes[data.group] = data.id;

            data.neighbors = {};
            data.id =    data.id.toString();
            data.label = data.id.toString();
            data.redFiltersID = {};
            data.greenFiltersID = {};

            // Nodes as common neighbors do not belong to any filter
            if (filterID)
                data.greenFiltersID[filterID] = true;

            // Shorten too long string values on the graph
            if (data.label.length > 30)
                data.label = data.label.substring(0, 30) + ' ...';

            this.application.graph.network.body.data.nodes.add(data);

            // Hide node related to red filters
            const node = this.application.graph.network.body.nodes[data.id];
            this.application.graph.network.body.data.nodes.update([{ id: node.id, hidden: this.application.filters.checkDisplayRedNode(node) }]);

            return node;
        }
    }

    /*
     * Merge a 'source' object to a 'target' recursively.
     * Different values of the same key will be concatenated with a comma
     */
    merge(target, source) {
        if (!source)
            return target;

        // Iterate through the 'source' properties,
        // if an 'Object' - set property to merge of 'target' and 'source' properties
        for (const key of Object.keys(source)) {
            if (source[key] instanceof Object &&
              !(source[key] instanceof Array) &&
                key in target) {

                Object.assign(source[key], this.merge(target[key], source[key]));
            }
        }

        // Join 'target' and modified 'source'
        for (const key in source) {
            if (target.hasOwnProperty(key)) {
                if (!target[key].match(new RegExp('(^|\\W)'+this.escapeRegex(source[key].toString())+'(\\W|$)'))) {
                    if (Array.isArray(source[key]))
                        target[key] += ', ' + source[key].join(', ');
                    else
                        target[key] += ', ' + source[key].toString();
                }

            } else if (source[key]) {  // Value sometimes is Null
                // Separate arrays and strings
                if (Array.isArray(source[key]))
                    target[key] = source[key].join(', ');
                else
                    target[key] = source[key].toString();
            }
        }

        return target;
    }

    /*
     * Escape special characters in a string for use inside of a Regular Expression
     */
    escapeRegex(string) {
        return string.replace(/[-\/\\^$*+?.()|[\]{}]/g, '\\$&');
    }
}
