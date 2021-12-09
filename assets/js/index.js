/*
 * Main webpage core.
 *
 * Allows to query data sources, create inclusion/exclusion filters,
 * manage dashboards, draw graph and statistics info,
 * show the latest service's features
 */
window.addEventListener('DOMContentLoaded', function() {

    class Application {
        constructor() {
            // Notifications
            this.notifications = new Notifications(this);

            // Query data sources
            this.search = new Search(this);

            // Nodes counts by groups
            this.tags = new Tags(this);

            // User filters to request new data or exclude existing
            this.filters = new Filters(this);

            // Dashboards saving & loading features
            this.dashboards = new Dashboards(this);

            // Graph canvas
            this.graph = new Graph(this);

            // Statistics charts when returned nodes limit is exceeded
            this.charts = new Charts(this);

            // Modal window
            this.modal = new Modal();

            // Notification about new service's features
            this.features = new Features(this);

            // Websocket connection for the client-server-client communication
            this.websocket = new Websocket(this);
        }
    }

    const application = new Application();

});
