/*
 * User filters to request new data or exclude existing
 */
class Filters {
    constructor(application) {
        // Pointer to the main page core
        this.application = application;

        // Container of filters HTML elements
        this.container = document.getElementById('filters');

        // Existing green (inclusion) and red (exclusion) filters.
        // Each entry is in the form:
        //
        // 'id': {
        //     'enabled': true/false,
        //     'query':   user's initial query,
        // }
        this.greenFilters = {};
        this.redFilters = {};
    }

    /*
     * UUID generator
     */
    uuid() {
        var dt = new Date().getTime(),
            uuid = 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
                var r = (dt + Math.random()*16)%16 | 0;
                dt = Math.floor(dt/16);
                return (c=='x' ? r :(r&0x3|0x8)).toString(16);
            });

        return uuid;
    }

    /*
     * Create green filter after a new search.
     * Receives user's query
     */
    addGreen(query) {
        //console.log(query);

        // Do not create dublicates
        for (const id in this.greenFilters)
            if (this.greenFilters[id].query === query) return id;

        var id = this.uuid();

        // Add a new filter
        this.new('green', query, id);

        this.greenFilters[id] = { 'enabled': true, 'query': query };

        if (this.application.graph.fullscreenBtn.className.indexOf('expand') !== -1)
            this.application.graph.position();

        this.application.modal.ok();

        // Save current user's filters in a database
        this.save();

        return id;
    }

    /*
     * Add red filter to hide necessary nodes.
     * Receives user's query
     */
    addRed(query) {
        //console.log(query);

        // Skip already existing filter
        for (const id in this.redFilters) {
            if (this.redFilters[id].query === query) {
                this.application.modal.error('Such <span class="red_fg">red</span> filter already exists!', 'Modify search input value and try again!');
                return;
            }
        }

        var id = this.uuid();

        // Add a new one
        this.new('red', query, id);

        this.redFilters[id] = { 'enabled': true, 'query': query };

        // Hide nodes
        this.hideRedNodes(id, query);
        // Update nodes count values
        this.application.tags.update();

        if (this.application.graph.fullscreenBtn.className.indexOf('expand') !== -1)
            this.application.graph.position();

        this.application.modal.ok();

        // Save current user's filters in a database
        this.save();

        return;
    }

    /*
     * Create a new filter's HTML object.
     *
     * Receives:
     *     color - green or red
     *     query - user's query for this filter
     *     id - filter's unique ID
     */
    new(color, query, id) {
        const div =   document.createElement('div'),
              input = document.createElement('div');

        div.className = 'ui action input ' + color + ' filter';

        // Set filter's ID
        div.setAttribute('data-id', id);

        if (query.length > 100)
            query = query.substring(0, 100) + ' ...';

        input.innerText = query;

        // Add filter management buttons to the filter's HTML element
        const editBtn =   document.createElement('button'),  // Copy user's query to the search input bar
              toggleBtn = document.createElement('button'),  // Enable/disable filter
              deleteBtn = document.createElement('button');  // Delete filter

        editBtn.className = 'ui icon small button';
        editBtn.innerHTML = '<i class="edit icon"></i>';
        editBtn.title =     'Edit';
        editBtn.onclick = () => {
            this.edit(editBtn);
        }

        toggleBtn.className = 'ui icon small button';
        toggleBtn.innerHTML = '<i class="eye icon"></i>';
        toggleBtn.title =     'Toggle';
        toggleBtn.onclick = () => {
            this.toggle(toggleBtn);
        }

        deleteBtn.className = 'ui icon small button';
        deleteBtn.innerHTML = '<i class="trash icon"></i>';
        deleteBtn.title =     'Delete';
        if (color === 'green') {
            deleteBtn.onclick = () => { this.deleteGreen(id); }
        } else {
            deleteBtn.onclick = () => { this.deleteRed(deleteBtn); }
        }

        div.appendChild(input);
        div.appendChild(editBtn);
        div.appendChild(toggleBtn);
        div.appendChild(deleteBtn);

        this.container.appendChild(div);

        return toggleBtn;
    }

    /*
     * Save all user's filters
     */
    save() {
        this.application.websocket.send('filters', JSON.stringify([this.greenFilters, this.redFilters]));
    }

    /*
     * Copy filter's query to the search input bar.
     * Receives an HTML element of the selected filter
     */
    edit(elem) {
        const id = elem.parentNode.getAttribute('data-id'),
              re = /FROM (.*) WHERE /g;

        var source, query;

        // Only green filters can contain data source name
        if (this.greenFilters.hasOwnProperty(id)) {
            query = this.greenFilters[id].query;
            source = re.exec(query);
        } else {
            query = this.redFilters[id].query;
        }

        // Set data sources dropdown value
        if (source) {
            if ($('.ui.source.dropdown').dropdown('get item', source[1])) {
                // Data source is known
                $('.ui.source.dropdown').dropdown('set exactly', source[1]);
            }

            // Set search input bar value
            this.application.search.input.value = query.replace(/FROM .* WHERE /g, '');

        // Data source is unknown and skip red filters
        } else {
                this.application.search.input.value = query;
        }
    }

    /*
     * Toggle filter - enable or disable,
     * hide/unhide related nodes/edges.
     *
     * Receives an HTML element of the selected filter
     */
    toggle(elem) {
        const filter = elem.parentNode,
              id = elem.parentNode.getAttribute('data-id');

        // Disable filter
        if (filter.className.indexOf('disabled') === -1) {
            $(filter).addClass('disabled');

            // Update button's icon
            elem.firstElementChild.className = 'eye slash icon';

            if (filter.className.indexOf('green') !== -1) {
                this.greenFilters[id].enabled = false;
                this.hideGreenNodes(id);

            } else if (filter.className.indexOf('red') !== -1) {
                this.redFilters[id].enabled = false;
                this.showRedNodes(id);
            }

        // Enable filter
        } else {
            $(filter).removeClass('disabled');

            // Update button's icon
            elem.firstElementChild.className = 'eye icon';

            if (filter.className.indexOf('green') !== -1) {
                this.greenFilters[id].enabled = true;

                if (this.greenFilters[id].empty)
                    this.application.search.query(this.greenFilters[id].query);
                else
                    this.showGreenNodes(id);

            } else if (filter.className.indexOf('red') !== -1) {
                this.redFilters[id].enabled = true;
                this.hideRedNodes(id, this.redFilters[id].query);
            }
        }

        // Update nodes count values
        this.application.tags.update();

        // Save current user's filters in a database
        this.save();
    }

    /*
     * Show nodes related to the green filter by its ID
     */
    showGreenNodes(id) {
        const updateArray = [],
              clusters = [];

        for (var n in this.application.graph.network.body.nodes) {
            const node = this.application.graph.network.body.nodes[n];

            if (!node.isCluster) {
                if (node.options.greenFiltersID[id] === false) {
                    node.options.greenFiltersID[id] = true;

                    // Hide node related to the red filters
                    updateArray.push({ 'id': node.id, 'hidden': this.checkDisplayRedNode(node) });
                }

            } else {
                clusters.push(node.id);
            }
        }

        this.application.graph.network.body.data.nodes.update(updateArray);

        for (var i = 0; i < clusters.length; i++) {
            const cluster = clusters[i];

            if (this.clusterIsVisible(cluster))
                this.application.graph.network.clustering.updateClusteredNode( cluster, { hidden:false });
        }
    }

    /*
     * Hide nodes related to the green filter by its ID
     */
    hideGreenNodes(id) {
        const updateArray = [],
              clusters = [];

        for (var n in this.application.graph.network.body.nodes) {
            const node = this.application.graph.network.body.nodes[n];

            if (!node.isCluster) {
                if (node.options.greenFiltersID[id] === true) {
                    node.options.greenFiltersID[id] = false;

                    if (!this.hasEnabled(node.options.greenFiltersID))
                        updateArray.push({ 'id': node.id, 'hidden': true });
                }

            } else {
                clusters.push(node.id);
            }
        }

        this.application.graph.network.body.data.nodes.update(updateArray);

        for (var i = 0; i < clusters.length; i++) {
            const cluster = clusters[i];

            if (!this.clusterIsVisible(cluster))
                this.application.graph.network.clustering.updateClusteredNode( cluster, { hidden:true });
        }
    }

    /*
     * Delete green filter and related nodes by its ID
     */
    deleteGreen(id) {
        const nodesToDelete = [],
              edgesToDelete = [];

        // Find filter to delete by its ID
        if (this.greenFilters.hasOwnProperty(id)) {
            const elem = document.querySelectorAll('[data-id="' + id + '"]')[0];
            elem.remove();

            delete this.greenFilters[id];
            delete FILTERS[0][id];
        }

        for (var n in this.application.graph.network.body.nodes) {
            const node = this.application.graph.network.body.nodes[n];

            if (node.options.greenFiltersID.hasOwnProperty(id)) {
                delete node.options.greenFiltersID[id];

                if (!this.hasEnabled(node.options.greenFiltersID)) {
                    nodesToDelete.push({ 'id': node.id });

                    for (var i = 0; i < node.edges.length; i++) {
                        edgesToDelete.push({ 'id': node.edges[i].id });
                    }
                }
            }
        }

        // Delete filter related nodes & edges
        this.application.graph.network.body.data.nodes.remove(nodesToDelete);
        this.application.graph.network.body.data.edges.remove(edgesToDelete);

        // Recalculate nodes size
        this.application.graph.recalculateSizes();

        // Actualize graph canvas Y position.
        // It will change when the amount of filters rows changes
        this.application.graph.position();

        // Update nodes count values
        this.application.tags.update();

        // Save current user's filters in a database
        this.save();
    }

    /*
     * Show all hidden nodes related to the red filter by its ID
     */
    showRedNodes(id) {
        const updateArray = [];

        for (var n in this.application.graph.network.body.nodes) {
            const node = this.application.graph.network.body.nodes[n];

            //if (node.options.redFiltersID && node.options.redFiltersID[id] === true) {
            if (node.options.redFiltersID[id] === true) {
                node.options.redFiltersID[id] = false;

                if (this.hasEnabled(node.options.redFiltersID))
                    updateArray.push({ 'id': node.id, 'hidden': true });
                else if (this.hasEnabled(node.options.greenFiltersID))
                    updateArray.push({ 'id': node.id, 'hidden': false });
            }
        }

        this.application.graph.network.body.data.nodes.update(updateArray);
    }

    /*
     * Hide all nodes with the given 'field=value'.
     *
     * Receives:
     *     id - unique filter's ID
     *     query - user's exclusion query. Example: "NOT ip='8.8.8.8'"
     */
    hideRedNodes(id, query) {
        const re = /NOT ([\w-_\.]*)=(?:"|')(.*)(?:"|')$/g,
              exclude = re.exec(query),  // Get field name and its value from user's query
              updateArray = [];

        for (var n in this.application.graph.network.body.nodes) {
            const node = this.application.graph.network.body.nodes[n];

            if (node.options.attributes[exclude[1]] === exclude[2]) {
                // if (!node.options.redFiltersID)
                //     node.options.redFiltersID = {};

                node.options.redFiltersID[id] = true;
                updateArray.push({ 'id': node.id, 'hidden': true });
            }
        }

        this.application.graph.network.body.data.nodes.update(updateArray);
    }

    /*
     * Apply existing red filters to the new received data.
     * Receives a single graph node
     */
    checkDisplayRedNode(node) {
        for (var id in this.redFilters) {
            const filter =  this.redFilters[id],
                  re =      /NOT ([\w-_\.]*)=(?:"|')(.*)(?:"|')$/g,
                  exclude = re.exec(filter.query),
                  attrs =   node.options.attributes;

            if (node.options.attributes[exclude[1]] === exclude[2]) {
                node.options.redFiltersID[id] = filter.enabled;

                if (filter.enabled)
                    return true;
            }
        }

        return false;
    }

    /*
     * Delete red filter and show related hidden nodes/edges again.
     * Receives an HTML element of red filter
     */
    deleteRed(elem) {
        const filter = elem.parentNode,
              id = elem.parentNode.getAttribute('data-id');

        elem.parentNode.remove();
        delete this.redFilters[id];
        delete FILTERS[1][id];

        if (filter.className.indexOf('disabled') === -1)
            this.showRedNodes(id);

        // Update nodes count values
        this.application.tags.update();

        // Actualize canvas Y position
        this.application.graph.position();

        // Save current user's filters in a database
        this.save();
    }

    /*
     * Check whether at least one filter is enabled for a single node.
     *
     * For the green filters:
     *     - if True  - node should be visible
     *     - if False - node shouldn't be visible
     *
     * For the red filters:
     *     - if True  - node shouldn't be visible
     *     - if False - node should be visible
     *
     * Receives a map of filters related to this single node
     */
    hasEnabled(obj) {
        for (var key in obj) {
            if (obj[key])
                return true;
        }

        return false;
    }

    /*
     * Check whether at least 1 cluster node stays visible,
     * so the cluster will be visible too.
     *
     * Receives a cluster's unique ID
     */
    clusterIsVisible(id) {
        const nodes = this.application.graph.network.clustering.getNodesInCluster(id);
        var i = nodes.length;

        while (i--) {
            const node = this.application.graph.network.body.nodes[nodes[i]];

            if (!node.options.hidden)
                return true;
        }

        return false;
    }

    /*
     * Restore created filters.
     * Enabled green filters queries will be launched again
     */
    restore() {
        // Skip empty
        if (FILTERS === '')
            return;

        // Add red filters first
        this.redFilters = FILTERS[1];

        for (const id in this.redFilters) {
            const toggleBtn = this.new('red', this.redFilters[id].query, id);

            if (!this.redFilters[id].enabled) {
                $(toggleBtn.parentNode).addClass('disabled');
                toggleBtn.firstElementChild.className = 'eye slash icon';
            }
        }

        // Launch enabled green filters queries.
        // Disabled filters can be simply created as HTML objects
        for (let id in FILTERS[0]) {
            const filter = FILTERS[0][id];

            if (!filter.enabled) {
                const toggleBtn = this.new('green', filter.query, id);

                $(toggleBtn.parentNode).addClass('disabled');
                toggleBtn.firstElementChild.className = 'eye slash icon';

                this.greenFilters[id] = {
                    enabled: false,
                    query:   filter.query,
                    empty:   true
                }

            } else {
                this.application.search.query(filter.query);
            }
        }

        // Adopt graph canvas position
        this.application.graph.position();
    }
}
