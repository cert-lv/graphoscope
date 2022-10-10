/*
 * Graph canvas
 */
class Graph {
    constructor(application) {
        // Pointer to the main page core
        this.application = application;

        // Set graph styling & physics
        this.options = this.options();

        // HTML elements for the easier access:
        // Graph canvas container
        this.container =     document.getElementById('graph');
        // Button to center graph elements
        this.centerBtn =     document.getElementById('center');
        // Button to export canvas as an image
        this.imageBtn =      document.getElementById('image');
        // Button to export visible graph data as a file
        this.exportBtn =     document.getElementById('export');
        // Button to import graph data from a file
        this.importBtn =     document.getElementById('import');
        // Button to toggle fullscreen mode
        this.fullscreenBtn = document.getElementById('fullscreen');
        // Table with graph elements attributes
        this.attributes =    document.getElementById('attributes');
        // User notes for the clicked graph element
        this.notes =         document.getElementById('attributes-notes');
        // A list of selected nodes common attributes
        this.common =        document.getElementById('common-attributes');
        // Button to save custom notes for the clicked graph element
        this.saveNotesBtn =  $('.save.notes.button');

        // Top margin because of a separate top header bar
        this.topMargin = $('.top-header').height() + 20;  // Height + margin

        // Create a clean graph network ..
        this.network = new vis.Network(this.container, {}, this.options);
        // Change default network colors
        this.changeDefaultColors();

        // .. or a fast demo scene
        // this.nodes = this.initNodes();  // Demo nodes
        // this.edges = this.initEdges();  // Demo edges
        // this.network = new vis.Network(this.container, { nodes: this.nodes, edges: this.edges }, this.options);

        // Pointer to the created canvas
        this.canvas = this.container.childNodes[0].childNodes[0];

        // Bind some Web GUI buttons
        this.bind();

        // Show attributes when node/edge is clicked
        this.setupLeftClick();
        // Setup right click context menu
        this.setupRightClick();
        // Limit zoom levels
        // TODO: remove when 'https://github.com/visjs/vis-network/pull/629' is merged & new version released
        this.setupZoomLimit();
        // Select nodes with a rectangle by a mouse right button
        this.setupSelection();
        // Delete selected nodes with a 'Del' key
        this.setupNodeDeletion();

        // Prepare regex to detect country codes,
        // ISO 3166-1 (alfa-2)
        this.isCountry = new RegExp('^(A(D|E|F|G|I|L|M|N|O|R|S|T|Q|U|W|X|Z)|B(A|B|D|E|F|G|H|I|J|L|M|N|O|R|S|T|V|W|Y|Z)|C(A|C|D|F|G|H|I|K|L|M|N|O|R|U|V|' +
                                    'X|Y|Z)|D(E|J|K|M|O|Z)|E(C|E|G|H|R|S|T)|F(I|J|K|M|O|R)|G(A|B|D|E|F|G|H|I|L|M|N|P|Q|R|S|T|U|W|Y)|H(K|M|N|R|T|U)|I(D|' +
                                    'E|Q|L|M|N|O|R|S|T)|J(E|M|O|P)|K(E|G|H|I|M|N|P|R|W|Y|Z)|L(A|B|C|I|K|R|S|T|U|V|Y)|M(A|C|D|E|F|G|H|K|L|M|N|O|Q|P|R|S|' +
                                    'T|U|V|W|X|Y|Z)|N(A|C|E|F|G|I|L|O|P|R|U|Z)|OM|P(A|E|F|G|H|K|L|M|N|R|S|T|W|Y)|QA|R(E|O|S|U|W)|S(A|B|C|D|E|G|H|I|J|K|' +
                                    'L|M|N|O|R|T|V|Y|Z)|T(C|D|F|G|H|J|K|L|M|N|O|R|T|V|W|Z)|U(A|G|M|S|Y|Z)|V(A|C|E|G|I|N|U)|W(F|S)|Y(E|T)|Z(A|M|W))$');
    }

    /*
     * Bind Web GUI buttons
     */
    bind() {
        // Center graph elements
        this.centerBtn.addEventListener('click', (e) => {
            this.center();
        });

        // Export canvas as an image
        this.imageBtn.addEventListener('click', (e) => {
            this.saveAsImage();
        });

        // Export visible graph data as a file
        this.exportBtn.addEventListener('click', (e) => {
            this.export();
        });

        // Import graph data from a file
        this.importBtn.addEventListener('click', (e) => {
            this.import();
        });

        // Toggle fullscreen mode
        this.fullscreenBtn.addEventListener('click', (e) => {
            this.fullscreen();
        });

        // Save custom notes for the clicked graph element
        this.saveNotesBtn.on('click', (e) => {
            this.saveNotes();
        });

        // Reposition graph canvas when browser window is resized
        window.addEventListener('resize', (e) => {
            if (this.fullscreenBtn.className.indexOf('expand') !== -1)
                this.position();
        });

    }

    /*
     * Create an array with demo nodes
     */
    initNodes() {
        const nodes = new vis.DataSet([
            { id: 'j', label: 'j', group: 'ip',     size: 20, greenFiltersID: {}, redFiltersID: {}, attributes: {} },
            { id: 'e', label: 'e', group: 'domain', size: 10, greenFiltersID: {}, redFiltersID: {}, attributes: {} },
            { id: 'h', label: 'h', group: 'domain', size: 10, greenFiltersID: {}, redFiltersID: {}, attributes: {} },
            { id: 'k', label: 'k', group: 'rtir',   size: 10, greenFiltersID: {}, redFiltersID: {}, attributes: {}, hidden: true },
            { id: 'g', label: 'g', group: 'rtir',   size: 10, greenFiltersID: {}, redFiltersID: {}, attributes: {} }
        ]);

        return nodes;
    }

    /*
     * Create an array with demo edges
     */
    initEdges() {
        const edges = new vis.DataSet([
            { from: 'j', to: 'e', label: 'resolves', group: 'pdns' },
            { from: 'j', to: 'h', label: 'resolves' },
            { from: 'j', to: 'k', label: ''         },
            { from: 'j', to: 'g', label: ''         },
            { from: 'e', to: 'k', label: ''         },
            { from: 'k', to: 'g', label: ''         }
        ]);

        return edges;
    }

    /*
     * Change default network colors.
     *
     * By default if node's group styling is not defined in "groups.json"
     * its highlight colors won't be affected by "options.nodes.color.highlight"
     */
    changeDefaultColors() {
        for (var i = 0; i < this.network.groups._defaultGroups.length; i++) {
            this.network.groups._defaultGroups[i].highlight = {
                background: SETTINGS.BGcolor,
                border: SETTINGS.BorderColor
            }
        }
    }

    /*
     * Graph engine options.
     * More info: https://visjs.github.io/vis-network/docs/network/#options
     */
    options() {
        const options = {
            nodes: {
                fixed: false,
                shape: 'dot',
                size: SETTINGS.NodeSize,
                borderWidth: SETTINGS.BorderWidth,
                //borderWidthSelected: 10,
                color: {
                    highlight: {
                        background: SETTINGS.BGcolor,
                        border: SETTINGS.BorderColor
                    }
                },
                font: {
                    size: SETTINGS.NodeFontSize,
                },
                shadow: {
                    enabled: SETTINGS.Shadow
                }
            },
            edges: {
                width: SETTINGS.EdgeWidth,
                color: {
                    highlight: SETTINGS.EdgeColor
                },
                font: {
                    size: SETTINGS.EdgeFontSize,
                    color: SETTINGS.EdgeFontColor
                },
                arrows: {
                    to: {
                        enabled: SETTINGS.Arrow
                    }
                },
                smooth: {
                    enabled: SETTINGS.Smooth
                }
            },
            groups: GROUPS,
            interaction: {
                hover: SETTINGS.Hover,
                multiselect: SETTINGS.MultiSelect,
                hideEdgesOnDrag: SETTINGS.HideEdgesOnDrag
            },
            physics: {
                //timestep: 0.75,
                barnesHut: {
                    avoidOverlap: 0.1,
                    gravitationalConstant: -10000,
                    //centralGravity: 1.5,
                    springConstant: 0.01,
                    springLength: 50, // Low value disables initial physics when 'SETTINGS.NodeSize' is too big
                    damping: 0.2
                }
            },
            layout: {
                //randomSeed: 1,
                improvedLayout: true
            }
        }

        return options;
    }

    /*
     * Setup graph manipulating with a left mouse button and
     * show attributes when node/edge is clicked
     */
    setupLeftClick() {
        this.network.on('hoverNode', (params) => {
            this.network.canvas.body.container.style.cursor = 'pointer'
        });
        this.network.on('blurNode', (params) => {
            this.network.canvas.body.container.style.cursor = 'default'
        });

        this.network.on('hoverEdge', (params) => {
            this.network.canvas.body.container.style.cursor = 'pointer'
        });
        this.network.on('blurEdge', (params) => {
            this.network.canvas.body.container.style.cursor = 'default'
        });

        this.network.on('dragStart', (params) => {
            // Skip if dragging canvas itself
            if (params.nodes.length === 0)
                return;

            for (var i = 0; i < params.nodes.length; i++) {
                const id = params.nodes[i];
                this.network.body.nodes[id].options.fixed = { x:false, y:false };
            }

            this.network.setOptions({
                physics: { enabled:true }
            });
        });

        this.network.on('dragEnd', (params) => {
            // Skip if dragging canvas itself
            if (params.nodes.length === 0)
                return;

            for (var i = 0; i < params.nodes.length; i++) {
                const id = params.nodes[i];
                this.network.body.nodes[id].options.fixed = { x:true, y:true };
            }

            // Stop all animations after dragging
            this.network.setOptions({
                physics: { enabled:false }
            });

            this.network.stopSimulation();
        });

        this.network.on('click', (params) => {
            // console.log(params);

            if (params.nodes.length !== 0) {
                const node = this.network.body.nodes[params.nodes[0]];
                this.showAttributes(node.id, node.options.attributes);
            }

            else if (params.edges.length !== 0) {
                // Must use '.get()' instead of:
                // this.network.body.edges[params.edges[0]]
                // or normal edges attributes are not visible

                const edge = this.network.body.data.edges.get(params.edges[0]);

                if (edge) {
                    const edge = this.network.body.data.edges.get(params.edges[0]);
                    this.showAttributes(edge.id, edge.attributes);
                }

                // Cluster edges appear only here
                else {
                    const edge = this.network.body.edges[params.edges[0]];
                    this.showAttributes(edge.fromId+'-'+edge.toId, edge.attributes);
                }
            }

            else {
                this.attributes.style.display = 'none';
                this.common.style.display = 'none';
                this.notes.removeAttribute('data-id');
                this.saveNotesBtn.removeClass('disabled loading');
                this.network.redraw();
            }

            this.menu.style.visibility = 'hidden';
            this.clusterMenu.style.visibility = 'hidden';
        });
    }

    /*
     * Setup right click context menu
     */
    setupRightClick() {
        // Show menu when node is selected
        this.network.on('oncontext', (params) => {
            params.event.preventDefault();

            const id = this.network.getNodeAt(params.pointer.DOM),
                  browserHeight = window.innerHeight || window.clientHeight,
                  browserWidth  = window.innerWidth  || window.clientWidth;

            if (!id) {
                this.menu.style.visibility = 'hidden';
                this.clusterMenu.style.visibility = 'hidden';
                return;
            }

            this.expandingNode = this.network.body.nodes[id];

            // Show normal node's menu when it's selected
            if (!this.expandingNode.isCluster) {
                // Build dynamic grouping options
                this.buildSubmenu(this.expandingNode);

                // Prevent menu be out of the screen's visible space
                if (params.event.clientY + this.menu.offsetHeight < browserHeight)
                    this.menu.style.top = params.event.clientY + 'px';
                else
                    this.menu.style.top = browserHeight - this.menu.offsetHeight - 10 + 'px';

                if (params.event.clientX + this.menu.offsetWidth < browserWidth)
                    this.menu.style.left = params.event.clientX + 'px';
                else
                    this.menu.style.left = browserWidth - this.menu.offsetWidth - 10 + 'px';

                this.menu.style.visibility = 'visible';
                this.clusterMenu.style.visibility = 'hidden';
            }

            // Show cluster's menu when it's selected
            else {
                // Prevent menu be out of the screen's visible space
                if (params.event.clientY + this.clusterMenu.offsetHeight < browserHeight)
                    this.clusterMenu.style.top = params.event.clientY + 'px';
                else
                    this.clusterMenu.style.top = browserHeight - this.clusterMenu.offsetHeight - 10 + 'px';

                if (params.event.clientX + this.clusterMenu.offsetWidth < browserWidth)
                    this.clusterMenu.style.left = params.event.clientX + 'px';
                else
                    this.clusterMenu.style.left = browserWidth - this.clusterMenu.offsetWidth - 10 + 'px';

                this.clusterMenu.style.visibility = 'visible';
                this.menu.style.visibility = 'hidden';
            }
        });

        this.menu =        document.getElementById('context-menu');
        this.clusterMenu = document.getElementById('cluster-context-menu');

        // Graph expanding actions
        const menuItems = this.menu.querySelectorAll('.item.expand');

        for (var i = 0; i < menuItems.length; i++) {
            const item = menuItems[i];

            item.addEventListener('click', () => {
                this.expand(item.getAttribute('data-action'), this.expandingNode.id);
                this.menu.style.visibility = 'hidden';
            });
        }

        // Find selected nodes common attributes/neighbors
        this.menu.querySelector('.item.common').addEventListener('click', () => {
            this.menuCommon();
        });

        // Delete selected nodes
        this.menu.querySelector('.item.delete').addEventListener('click', () => {
            this.menuDelete();
        });

        // Open cluster
        this.clusterMenu.querySelector('.item.open').addEventListener('click', () => {
            this.menuOpenCluster();
        });

        // Delete cluster
        this.clusterMenu.querySelector('.item.delete').addEventListener('click', () => {
            this.menuDeleteCluster();
        });
    }

    /*
     * Find selected nodes common attributes/neighbors
     */
    menuCommon() {
        const queries = [],
              attributes = [],
              selected = this.network.getSelectedNodes(),
              startTime = $('#rangestart').calendar('get date').toISOString().substr(0, 19) + '.000Z',
              endTime =   $('#rangeend').calendar(  'get date').toISOString().substr(0, 19) + '.000Z';

        for (var i = 0; i < selected.length; i++) {
            const node = this.application.graph.network.body.nodes[selected[i]];

            queries.push(node.options.search + '=\'' + node.options.attributes[node.options.group] + '\'');
            attributes.push(node.options.attributes);
        }

        // Send nodes to the server
        this.application.websocket.send('common', JSON.stringify(queries), '\'' + startTime + '\' AND \'' + endTime +'\'');

        // Disable search button and hide menus if visible
        this.application.search.searchBtn.addClass('disabled loading');
        this.menu.style.visibility = 'hidden';

        // Clear old attributes
        $('.ui.common-attributes.table').empty();
        $('.ui.common-neighbors.table').empty();
        this.attributes.style.display = 'none';
        this.common.style.display = 'block';

        // Search for common attributes
        var init = attributes[0],
            count = 0;

        if (attributes.length > 1) {
            for (var key in init) {
                var missing = false;

                for (var i = 1; i < attributes.length; i++) {
                    const attr = attributes[i];

                    if (!attr.hasOwnProperty(key) || attr[key] != init[key]) {
                        missing = true;
                        break
                    }
                }

                if (!missing) {
                    $('.ui.common-attributes.table').append('<tr><td>' + key + '</td><td>' + init[key] + '</td></tr>');
                    count++;
                }
            }
        }

        if (count === 0) {
            $('.ui.common-attributes.table').append('<tr><td class="grey_fg">Nothing found!</td><td class="empty"></td></tr>');
        }
    }

    /*
     * Delete selected nodes
     */
    menuDelete() {
        const edgesToDelete = [];

        for (var i = 0; i < this.expandingNode.edges.length; i++)
            edgesToDelete.push({ id: this.expandingNode.edges[i].id });

        this.network.body.data.nodes.remove([{ id: this.expandingNode.id }]);
        this.network.body.data.edges.remove(edgesToDelete);

        this.application.tags.update();
        this.menu.style.visibility = 'hidden';
    }

    /*
     * Open cluster
     */
    menuOpenCluster() {
        // Allow to select child nodes again
        const nodes = this.application.graph.network.clustering.getNodesInCluster(this.expandingNode.id);

        var i = nodes.length;
        while (i--) {
            this.application.graph.network.body.nodes[nodes[i]].options.inCluster = false;
        }

        // Open cluster
        this.network.openCluster(this.expandingNode.id);
            this.network.setOptions({
            physics: { enabled:true }
        });

        this.application.tags.update();
        this.clusterMenu.style.visibility = 'hidden';
    }

    /*
     * Delete cluster
     */
    menuDeleteCluster() {
        const nodes = this.application.graph.network.clustering.getNodesInCluster(this.expandingNode.id),
              nodesToDelete = [];

        var i = nodes.length;

        while (i--) {
            nodesToDelete.push({ id: this.application.graph.network.body.nodes[nodes[i]].id });
        }

        this.network.body.data.nodes.remove(nodesToDelete);
        this.application.tags.update();
        this.clusterMenu.style.visibility = 'hidden';
    }

    /*
     * Prevent unlimited zoom
     */
    setupZoomLimit() {
        this.network.on('zoom', (obj) => {

            const coef = 0.1 / obj.scale,
                  pos = this.network.getViewPosition();

            if (obj.scale < 0.2) {
                this.network.moveTo({
                    position: {
                        x: pos.x + (obj.pointer.x - this.canvas.width /2) * coef,
                        y: pos.y + (obj.pointer.y - this.canvas.height/2) * coef,
                    },
                    scale: 0.2,
                });
            }

            // vis-network already has some hard-coded max zoom level

            // if (this.network.getScale() > 2.0) {
            //     this.network.moveTo({
            //         position: {
            //             x: pos.x - (obj.pointer.x - this.canvas.width /2) * coef,
            //             y: pos.y - (obj.pointer.y - this.canvas.height/2) * coef,
            //         },
            //         scale: 2.0,
            //     });
            // }
        });
    }

    /*
     * Search for neighbors of the selected node
     */
    expand(source, id) {
        const selected = this.network.getSelectedNodes();
        if (selected.indexOf(id) === -1)
            selected.push(id);

        for (var i = 0; i < selected.length; i++) {
            const node = this.application.graph.network.body.nodes[selected[i]];
            this.application.search.query('FROM ' + source + ' WHERE ' + node.options.search + '=\'' + node.options.attributes[node.options.group] + '\'');

            console.log('Expanding by', node.options.search, '=', node.id, 'from', source);
        }
    }

    /*
     * Build dynamic grouping options.
     *
     * To avoid huge amount of hardcoded options, walk through the
     * selected node's neighbors and collect unique groups only
     */
    buildSubmenu(node) {
        const container = document.getElementById('context-submenu');

        // Clear old options
        container.innerHTML = '';

        for (var group in node.options.neighbors) {
            // Option template
            // <a class="item group" data-action="ip"><i class="gamepad icon"></i>ip</a>

            const a = document.createElement('a');

            a.className = 'item group';
            a.setAttribute('data-action', group);
            a.innerText = group;

            a.addEventListener('click', () => {
                this.group(a.getAttribute('data-action'), this.expandingNode.id);
                this.menu.style.visibility = 'hidden';
            });

            container.appendChild(a);
        }
    }

    /*
     * Group neighbors of the selected node
     */
    group(name, id) {
        //console.log('Group', name, 'of', id);
        var children = '';

        const clusterOptions = {
            joinCondition: (childOptions) => {
                // Skip selected node & wrong groups
                if (childOptions.group !== name || childOptions.id === id)
                    return false;

                // Otherwise check neighbors
                const node = this.application.graph.network.body.nodes[childOptions.id];
                var i = node.edges.length;

                while (i--) {
                    const e = node.edges[i];

                    if (e.fromId === id || e.toId === id) {
                        children += childOptions.id + '\n';
                        return true;
                    }
                }

                return false;
            },

            processProperties: (clusterOptions, childNodes, childEdges) => {
                clusterOptions.label = name.toUpperCase() + 's';
                clusterOptions.mass =  2;
                clusterOptions.group = 'cluster';

                // Init object first, to prevent key name 'name'
                clusterOptions.attributes = {};
                clusterOptions.attributes[name] = children;

                return clusterOptions;
            },

            clusterNodeProperties: {
                id: 'cluster-' + id + '-' + name
            }
        };

        this.network.cluster(clusterOptions);
        this.network.setOptions({
            physics: { enabled:true }
        });

        // Set default empty attributes to the newly created adges
        const cluster = this.application.graph.network.body.nodes['cluster-' + id + '-' + name];
        // When trying to combine only 1 node - no cluster is created
        if (!cluster) return;

        for (var i = 0; i < cluster.edges.length; i++)
            cluster.edges[i].attributes = {};

        // Set default empty lists of filters
        // to make new node compatible with the other nodes
        cluster.options.greenFiltersID = {};
        cluster.options.redFiltersID = {};

        this.application.tags.update();

        const nodes = this.application.graph.network.clustering.getNodesInCluster('cluster-' + id + '-' + name);

        var i = nodes.length;
        while (i--) {
            this.application.graph.network.body.nodes[nodes[i]].options.inCluster = true;
        }
    }

    /*
     * Rectangle nodes selection
     *
     * Info from:
     * - https://github.com/almende/vis/issues/977
     * - https://github.com/almende/vis/issues/3594
     * - https://github.com/Loriowar/comindivion/blob/master/web/static/js/vis_interactive.js
     */
    setupSelection() {
        // Skip if disabled by administrator
        if (!SETTINGS.MultiSelect) return;

        const NO_CLICK = 0;
        const RIGHT_CLICK = 3;
        const topMargin = this.topMargin;

        // State
        let drag = false,
            graph = this,
            DOMRect = {};

        // Selector
        const canvasify = (DOMx, DOMy) => {
            const { x, y } = this.network.DOMtoCanvas({ x: DOMx, y: DOMy });
            return [x, y];
        };

        const correctRange = (start, end) =>
            start < end ? [start, end] : [end, start];

        const selectFromDOMRect = () => {
            const [sX, sY] = canvasify(DOMRect.startX, DOMRect.startY),
                  [eX, eY] = canvasify(DOMRect.endX, DOMRect.endY),
                  [startX, endX] = correctRange(sX, eX),
                  [startY, endY] = correctRange(sY, eY);

            const selected = [];

            for (var id in this.network.body.nodes) {
                const node = this.network.body.nodes[id],
                      { x, y } = this.network.getPositions(id)[id];

                if (startX <= x && x <= endX && startY <= y && y <= endY && !node.isCluster && !node.options.hidden && !node.options.inCluster)
                    selected.push(id);
            }

            this.network.selectNodes(selected);

            // this.network.selectNodes(this.network.body.data.nodes.get().reduce(
            //     (selected, { id }) => {
            //         const { x, y } = this.network.getPositions(id)[id];
            //         return (startX <= x && x <= endX && startY <= y && y <= endY && !this.network.body.data.nodes.get(id).hidden && !inCluster) ?
            //         selected.concat(id) : selected;
            //     }, []
            // ));
        };

        /*
         * Listeners
         */

        // When mousedown, save the initial rectangle state
        this.container.addEventListener('mousedown', function({ which, pageX, pageY }) {
            if (which === RIGHT_CLICK) {
                Object.assign(DOMRect, {
                    startX: pageX - this.offsetLeft - 10,
                    startY: pageY - this.offsetTop  - 10 - topMargin,
                    endX:   pageX - this.offsetLeft - 10,
                    endY:   pageY - this.offsetTop  - 10 - topMargin
                });

                const id = graph.network.getNodeAt({
                    x: pageX - this.offsetLeft - 10,
                    y: pageY - this.offsetTop  - 10 - topMargin
                });

                if (!id)
                    drag = true;
            }
        });

        // When mousemove, update the rectangle state
        this.container.addEventListener('mousemove', function({ which, pageX, pageY }) {
            // Make selection rectangle disappear when accidently mouseupped outside 'container'
            if (drag) {
                if (which === NO_CLICK) {
                    drag = false;
                    graph.network.redraw();
                } else {
                    Object.assign(DOMRect, {
                        endX: pageX - this.offsetLeft - 10,
                        endY: pageY - this.offsetTop  - 10 - topMargin
                    });

                    graph.network.redraw();
                }
            }
        });

        // When mouseup, select nodes in a rectangle
        this.container.addEventListener('mouseup', ({ which }) => {
            if (which === RIGHT_CLICK && drag) {
                drag = false;
                this.network.redraw();
                selectFromDOMRect();
            }
        });

        // Rectangle drawer
        this.network.on('afterDrawing', (ctx) => {
            if (drag) {
                const [startX, startY] = canvasify(DOMRect.startX, DOMRect.startY),
                      [endX,   endY]   = canvasify(DOMRect.endX, DOMRect.endY),
                      scale = this.network.getScale();

                ctx.lineWidth = 2 / scale;
                ctx.setLineDash([5 / scale]);
                ctx.strokeStyle = 'rgba(0, 150, 250, 1.0)';
                ctx.strokeRect(startX, startY, endX - startX, endY - startY);
                ctx.setLineDash([]);
                ctx.fillStyle = 'rgba(0, 130, 250, 0.2)';
                ctx.fillRect(startX, startY, endX - startX, endY - startY);
            }
        });
    }

    /*
     * Delete selected nodes with a 'Delete' key
     */
    setupNodeDeletion() {
        $('.vis-network').keydown((e) => {
            if (e.key === 'Delete') {
                this.network.deleteSelected();
                this.recalculateSizes();

                // Update nodes count values
                this.application.tags.update();
            }
        });
    }

    /*
     * Recalculate nodes size after other nodes deletion
     */
    recalculateSizes() {
        for (const id in this.network.body.nodes) {
            const node = this.network.body.nodes[id];

            // Initial parameters
            var n = node.edges.length,
                size = SETTINGS.NodeSize,
                mass = 1;

            // Calculations based on edges count
            while (n--) {
                size = size + 1 / (size / 30);
                mass = size / 15;

                if (size > 100) {
                    size = 100;
                    mass = 15;
                    break;
                }
            }

            // Set resulting parameters
            node.options.size = size;
            node.options.mass = mass;
        }
    }

    /*
     * Show clicked node/edge attributes
     */
    showAttributes(id, attrs) {
        //console.log('attrs:', id, attrs);

        this.attributes.style.display = 'block';
        this.common.style.display = 'none';
        this.notes.value = '';
        this.notes.setAttribute('data-id', id);
        $('.ui.attributes.table').empty();

        for (const attr in attrs) {
            if (attr === 'source')
                continue;

            var value = attrs[attr].toString();

            if (value instanceof Array) {
                for (var i = 0; i < value.length; i++) {
                    value[i] = value[i].replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;');
                }

            } else {
                value = value.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;');
            }

            // Add a flag icon when the value is a country code
            if (this.isCountry.test(value.toUpperCase()))
                value += '<i class="' + value.toLowerCase() + ' flag"></i>';

            $('.ui.attributes.table').append('<tr><td>' + attr + '</td><td>' + value + '</td></tr>');
        }

        // Add source at the end of attributes table
        if (attrs['source'])
            $('.ui.attributes.table').append('<tr><td>source</td><td>' + attrs['source'] + '</td></tr>');

        // Search for selected element's notes
        this.showNotes(id);

        // Add right padding if vertical scrollbar appears
        if (this.attributes.scrollHeight > this.attributes.clientHeight)
            this.attributes.style.paddingRight = '10px';
        else
            this.attributes.style.paddingRight = '0';

        this.network.redraw();
    }

    /*
     * Show clicked node/edge user notes
     */
    showNotes(id) {
        this.application.websocket.send('notes', id);
        this.saveNotesBtn.addClass('disabled loading');
    }

    /*
     * Save user notes for the given graph element
     */
    saveNotes() {
        this.application.websocket.send('notes-save', encodeURI(this.notes.getAttribute('data-id')), this.notes.value);
        this.saveNotesBtn.addClass('disabled loading');
    }

    /*
     * Center graph elements
     */
    center() {
        this.network.fit();
    }

    /*
     * Save graph canvas as an image
     */
    saveAsImage() {
        const context = this.canvas.getContext('2d'),
              data = context.getImageData(0, 0, this.canvas.width, this.canvas.height),  // Get current ImageData for the canvas
              compositeOperation = context.globalCompositeOperation,                     // Store current globalCompositeOperation
              link = document.createElement('a');

        // Set to draw behind current content
        context.globalCompositeOperation = 'destination-over';
        // Set background color to white
        context.fillStyle = '#fff';
        // Draw background / rectangle on the entire canvas
        context.fillRect(0, 0, this.canvas.width, this.canvas.height);

        const dataURL = this.canvas.toDataURL('image/png');

        // Restore canvas state

        // Clear the canvas
        context.clearRect(0, 0, this.canvas.width, this.canvas.height);
        // Restore it with original / cached ImageData
        context.putImageData(data, 0, 0);
        // Reset the globalCompositeOperation to what it was
        context.globalCompositeOperation = compositeOperation;

        // Download the image
        link.setAttribute('download', 'graph.png');
        link.setAttribute('href', dataURL);
        link.click();
    }


    /*
     * Export visible graph elements as a file
     */
    export() {
        const data = [];

        for (var id in this.network.body.nodes) {
            const node = this.network.body.nodes[id];

            // Skip hidden nodes
            if (node.options.hidden)
                continue;

            // Nodes without neighbors
            if (node.edges.length === 0) {
                const item = {
                    from: {},
                    to:   {},
                    edge: {}
                }

                item.from.attributes = JSON.parse(JSON.stringify(node.options.attributes));  // Prevent modifying original object
                item.from.search = node.options.search;
                item.from.group = node.options.group;
                item.from.id = node.id;

                if (node.source) {
                    item.source = node.source;

                } else if (node.options.attributes.source) {
                    node.source = node.options.attributes.source;
                    item.source = node.options.attributes.source;
                    delete node.options.attributes.source;

                } else {
                    item.source = node.source;
                }

                delete item.from.attributes[item.from.group];

                if (Object.keys(item.from.attributes).length === 0)
                    delete item.from.attributes;

                data.push(item);
                continue;
            }

            // Check whether at least 1 visible neighbor exists
            var linked = false;

            for (var i = 0; i < node.edges.length; i++) {
                const edge = node.edges[i];

                if (!edge.from.options.hidden && !edge.to.options.hidden) {
                    linked = true;
                    break;
                }
            }

            for (var i = 0; i < node.edges.length; i++) {
                const edge = node.edges[i];

                // Skip double 'data' entries as every edge links 2 nodes
                if (edge.fromId === node.id && !edge.to.options.hidden) {
                    const attrs = this.network.body.data.edges.get(edge.id);

                    // Skip virtual cluster nodes
                    if (!attrs) continue;

                    const item = {
                        from: {},
                        to:   {},
                        edge: attrs.attributes
                    }

                    item.from.attributes = JSON.parse(JSON.stringify(edge.from.options.attributes));  // Prevent modifying original object
                    item.from.search = edge.from.options.search;
                    item.from.group = edge.from.options.group;
                    item.from.id = edge.from.id;

                    item.to.attributes = JSON.parse(JSON.stringify(edge.to.options.attributes));  // Prevent modifying original object
                    item.to.search = edge.to.options.search;
                    item.to.group = edge.to.options.group;
                    item.to.id = edge.to.id;

                    if (item.edge.source) {
                        edge.from.source = item.edge.source;
                        edge.to.source = item.edge.source;
                        item.source = item.edge.source;

                    } else if (edge.from.options.attributes.source) {
                        edge.from.source = edge.from.options.attributes.source;
                        edge.to.source = edge.from.options.attributes.source;
                        item.source = edge.from.options.attributes.source;
                        delete edge.from.options.attributes.source;

                    } else {
                        item.source = edge.from.source;
                    }

                    delete item.from.attributes.source;
                    delete item.from.attributes[item.from.group];
                    delete item.to.attributes.source;
                    delete item.to.attributes[item.to.group];
                    delete item.edge.source;

                    if (Object.keys(item.from.attributes).length === 0)
                        delete item.from.attributes;
                    if (Object.keys(item.to.attributes).length === 0)
                        delete item.to.attributes;

                    data.push(item);

                } else if (edge.fromId === node.id && edge.to.options.hidden && !linked) {
                    const item = {
                        from: {},
                        to:   {},
                        edge: {}
                    }

                    item.from.attributes = JSON.parse(JSON.stringify(edge.from.options.attributes));  // Prevent modifying original object
                    item.from.search = edge.from.options.search;
                    item.from.group = edge.from.options.group;
                    item.from.id = edge.from.id;

                    if (edge.from.source) {
                        item.source = edge.from.source;

                    } else if (edge.from.options.attributes.source) {
                        edge.from.source = edge.from.options.attributes.source;
                        item.source = edge.from.options.attributes.source;
                        delete edge.from.options.attributes.source;

                    } else {
                        item.source = edge.from.source;
                    }

                    delete item.from.attributes[item.from.group];

                    if (Object.keys(item.from.attributes).length === 0)
                        delete item.from.attributes;

                    data.push(item);

                } else if (edge.toId === node.id && edge.from.options.hidden && !linked) {
                    const item = {
                        from: {},
                        to:   {},
                        edge: {}
                    }

                    item.to.attributes = JSON.parse(JSON.stringify(edge.to.options.attributes));  // Prevent modifying original object
                    item.to.search = edge.to.options.search;
                    item.to.group = edge.to.options.group;
                    item.to.id = edge.to.id;

                    if (edge.to.source) {
                        item.source = edge.to.source;

                    } else if (edge.to.options.attributes.source) {
                        edge.to.source = edge.to.options.attributes.source;
                        item.source = edge.to.options.attributes.source;
                        delete edge.to.options.attributes.source;
                    } else {
                        item.source = edge.to.source;
                    }

                    delete item.to.attributes[item.to.group];

                    if (Object.keys(item.to.attributes).length === 0)
                        delete item.to.attributes;

                    data.push(item);
                }
            }
        }

        // Prevent download of empty data
        if (data.length === 0) {
            this.application.modal.error('No data!', 'Add some data to the graph and try again!');
            return;
        }

        // Download object as a file
        const dataStr = 'data:text/json;charset=utf-8,' + encodeURIComponent(JSON.stringify(data, undefined, 4)),
              downloadAnchorNode = document.createElement('a');

        downloadAnchorNode.setAttribute('href', dataStr);
        downloadAnchorNode.setAttribute('download', 'export.json');
        document.body.appendChild(downloadAnchorNode);  // Required for firefox
        downloadAnchorNode.click();
        downloadAnchorNode.remove();
    }

    /*
     * Import graph elements from a file
     */
    import() {
        const text = 'Select a JSON file with a graph data to display:' +
                        '<div class="ui import input" id="file">' +
                            '<input type="file" id="inputFile">' +
                            '<div class="ui icon grey button">' +
                                '<i class="attach icon"></i>' +
                                'File' +
                            '</div>' +
                        '</div>';

        this.application.modal.empty('Import data', text, 'download');

        // File selection for the import
        $('.ui.import.input > .ui.icon.button').on('click', () => {
            document.getElementById('inputFile').click();
        });

        // Show selected file's name
        $('input:file', '.ui.input').on('change', (e) => {
            if (e.target.files.length > 0) {
                var file = e.target.files[0],
                reader = new FileReader();

                reader.readAsText(file, 'UTF-8');
                reader.onload = (e) => {
                    try {
                        const result = JSON.parse(e.target.result);
                        this.application.search.processRelations(null, result);
                        this.application.modal.close();
                    } catch (e) {
                        this.application.modal.error('Error!', 'Invalid JSON file selected!');
                    }
                }
                reader.onerror = (e) => {
                    this.application.modal.error('Error!', 'Can\'t read file!');
                }
            }
        });
    }

    /*
     * Toggle canvas fullscreen state
     */
    fullscreen() {
        // Enable fullscreen
        if (this.fullscreenBtn.className.indexOf('expand') !== -1) {
            $(this.centerBtn).css({
                'position': 'fixed',
                'top':      10,
                'right':    116,
            });

            $(this.imageBtn).css({
                'position': 'fixed',
                'top':      10,
                'right':    85,
            });

            $(this.exportBtn).css({
                'position': 'fixed',
                'top':      10,
                'right':    59,
            });

            $(this.importBtn).css({
                'position': 'fixed',
                'top':      10,
                'right':    33,
            });

            $(this.fullscreenBtn).removeClass('expand').addClass('compress').css({
                'position': 'fixed',
                'top':      10,
                'right':    7,
            });

            $(this.container).css({
                'position': 'fixed',
                'width':    'auto',
                'top':      0,
                'right':    0,
                'z-index':  2,
            });

            this.network.redraw();

        // Disable fullscreen
        } else {
            $(this.centerBtn).css({
                'position': 'absolute',
                'right':    133,
            });

            $(this.imageBtn).css({
                'position': 'absolute',
                'right':    102,
            });

            $(this.exportBtn).css({
                'position': 'absolute',
                'right':    76,
            });

            $(this.importBtn).css({
                'position': 'absolute',
                'right':    50,
            });

            $(this.fullscreenBtn).removeClass('compress').addClass('expand').css({
                'position': 'absolute',
                'right':    24,
            });

            $(this.container).css({
                'position': 'absolute',
                'width':    'calc(100% - 10px)',
                'z-index':  0,
            });

            this.position();
        }
    }

    /*
     * Clear all graph related elements
     */
    clearAll() {
        // Remove nodes & edges
        const nodesToDelete = [],
              edgesToDelete = [];

        for (var id in this.network.body.nodes)
            nodesToDelete.push({ id: id });

        for (var id in this.network.body.edges)
            edgesToDelete.push({ id: id });

        this.network.body.data.nodes.remove(nodesToDelete);
        this.network.body.data.edges.remove(edgesToDelete);

        // Remove tags
        this.application.tags.container.innerHTML = '';

        // Remove filters
        this.application.filters.container.innerHTML = '';
        this.application.filters.greenFilters = {};
        this.application.filters.redFilters = {};

        // Clear search input
        this.application.search.input.value = '';

        this.application.modal.ok();

        // Clear charts
        this.application.charts.close();

        // Clear attributes
        $('.attributes-header').hide();
        $('.ui.attributes.table').empty();
    }

    /*
     * Set actual graph canvas Y position.
     * It will be changed when the amount of filters rows changes
     */
    position() {
        var y = this.application.filters.container.getBoundingClientRect().bottom;

        // Sometimes, when there is a too large amount of attributes
        // and window is scrolled down - negative value is returned
        // and graph container overlaps other HTML elements
        if (y < 0) return;

        y = y-this.topMargin + 'px';

        this.container.style.top = y;
        this.application.tags.container.style.top = y;
        this.centerBtn.style.top = y;
        this.imageBtn.style.top = y;
        this.exportBtn.style.top = y;
        this.importBtn.style.top = y;
        this.fullscreenBtn.style.top = y;

        //this.network.fit();
        this.network.redraw();
    }
}
