/*
 * Statistics charts when returned nodes limit is exceeded
 */
class Charts {
    constructor(application) {
        // Pointer to the main page core
        this.application = application;

        // HTML elements container
        this.container = document.getElementById('charts');
        this.header =    document.getElementById('charts-header');

        // List of created charts,
        // to be able to delete them later
        this.charts = [];

        // Bind button to close charts window
        document.getElementById('charts-close').addEventListener('click', (e) => {
            this.close();
        });

        this.container.addEventListener('click', (e) => {
            this.menu.style.visibility = 'hidden';
        });

        // Right click context menu
        this.setupMenu();
    }

    /*
     * Create charts based on received stats.
     *
     * Receives:
     *     query - user's query that returned too much results
     *     data - statistics numbers
     *     filterID - filter's unique UUID
     */
    create(query, data, filterID) {
        //console.log(query, data, filterID);

        // Resize charts area when working in a fullscreen mode
        if (this.application.graph.fullscreenBtn.className.indexOf('expand') === -1)
            this.container.className = 'maximized';
        else
            this.container.removeAttribute('class');

        // Clear previous charts
        this.clear();

        this.header.innerHTML = '<strong>' + data.source.toUpperCase() + '</strong> has too many results, ' +
                                'add filters manually or use the charts (based on limited data) to reduce the amount of returned data. ' +
                                'Or close the charts to see the possible data from the other sources';
        this.container.style.display = 'block';

        for (var field in data) {
            if (field === 'source') continue;
            this.generate(query, field, data[field], filterID);
        }

        // Right click context menu
        this.setupContext();
    }

    /*
     * Create a new chart.
     *
     * Receives:
     *     query - user's query that returned too much results
     *     field - data source's field this single chart is related to
     *     data - statistics numbers
     *     filterID - filter's unique UUID
     */
    generate(query, field, data, filterID) {
        //console.log(query, field, data, filterID);

        // Single chart container
        const chart = document.createElement('div'),
              name =  document.createElement('div');

        chart.id = field;
        this.container.setAttribute('data-query', query);
        this.container.setAttribute('data-filter', filterID);
        this.container.appendChild(chart);

        // Generate chart
        c3.generate({
            bindto: chart,
            data: {
                json: data,
                type : 'donut',

                onclick: (d, i) => {
                    this.menu.style.visibility = 'hidden';
                },
                oncontext: (d, i) => {
                    this.menu.style.visibility = 'hidden';
                },
                // onmouseover: (d, i) => {
                //     console.log('onmouseover', d, i);
                // },
                // onmouseout: (d, i) => {
                //     console.log('onmouseout',  d, i);
                // }
            },
            // color: {
            //     pattern: ['#344', '#566', '#788', '#9aa']
            // },
            size: {
                height: 300
            },
            donut: {
                width:    60,
                padAngle: 0.03,
                label: {
                    format: (value, ratio, id) => {
                        return value;
                    }
                }
            },
            tooltip: {
                format: {
                    value: (value, ratio, id, index) => {
                        return value;
                    }
                }
            }
        });

        this.charts.push(chart);

        name.className = 'chart-name';
        name.innerText = field;
        chart.appendChild(name);
    }

    /*
     * Setup right click context menu
     */
    setupContext() {
        d3.select('#charts').selectAll('.c3-shape')
            .on('contextmenu', (d, i) => {

                this.expandingField =  d3.event.target.ownerSVGElement.parentNode.id;
                this.expandingQuery =  d3.event.target.ownerSVGElement.parentNode.parentNode.getAttribute('data-query');
                this.expandingFilter = d3.event.target.ownerSVGElement.parentNode.parentNode.getAttribute('data-filter');
                this.expandingValue =  d.data.id;

                const browserHeight = window.innerHeight || window.clientHeight;

                // Prevent menu be out of the screen's visible space
                if (d3.event.clientY + this.menu.offsetHeight < browserHeight)
                    this.menu.style.top = d3.event.clientY + 'px';
                else
                    this.menu.style.top = browserHeight - this.menu.offsetHeight - 10 + 'px';

                this.menu.style.left = d3.event.clientX + 'px';
                this.menu.style.visibility = 'visible';

                d3.event.preventDefault();
        });
    }

    /*
     * Setup right click menu
     */
    setupMenu() {
        this.menu = document.getElementById('charts-context-menu');

        // Bind actions
        const menuItems = this.menu.querySelectorAll('.item');

        for (var i = 0; i < menuItems.length; i++) {
            const item = menuItems[i];

            item.addEventListener('click', () => {
                this.action(item.getAttribute('data-action'));
                this.menu.style.visibility = 'hidden';
            });
        }
    }

    /*
     * Run new query based on user choice.
     * Receives: action, whether to include (=) or exclude (<>) selected value
     */
    action(action) {
        // Delete temp. filter
        this.application.filters.deleteGreen(this.expandingFilter);

        const query = this.expandingQuery + ' AND ' + this.expandingField + action + '\'' + this.expandingValue + '\'';

        this.container.style.display = 'none';
        this.application.search.query(query);
    }

    /*
     * Clear previous charts
     */
    clear() {
        for (var i = 0; i < this.charts.length; i++) {
            const chart = this.charts[i];
            chart.parentNode.removeChild(chart);
        }

        this.charts.length = 0;
        this.charts = [];
    }

    /*
     * Hide charts window and related elements
     */
    close() {
        this.container.style.display = 'none';
        this.menu.style.visibility = 'hidden';
    }
}
