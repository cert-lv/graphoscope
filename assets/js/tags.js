/*
 * Display the amount of nodes of each group
 */
class Tags {
    constructor(application) {
        // Pointer to the main page core
        this.application = application;

        // Container of tags HTML elements
        this.container = document.getElementById('count');
    }

    /*
     * Update nodes count for each group
     */
    update() {
        const count = {};

        // Clear old values
        this.container.innerHTML = '';

        // Count existing nodes
        for (var id in this.application.graph.network.body.nodes) {
            const node = this.application.graph.network.body.nodes[id];

            if (!node.options.hidden) {
                if (count[node.options.group])
                    count[node.options.group] += 1;
                else
                    count[node.options.group] = 1;
            }
        }

        // Add new tag labels
        for (var group in count) {
            const entry = document.createElement('div');
            entry.innerHTML = '<div class="ui label">' + group + '<div class="detail">' + count[group] + '</div></div>';

            this.container.appendChild(entry);
        }
    }
}
