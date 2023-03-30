/*
 * Dashboards saving & loading features
 */
class Dashboards {
    constructor(application) {
        // Pointer to the main page core
        this.application = application;

        // HTML elements for the easier access
        this.dashboards = $('.ui.load.dashboard.dropdown');
        this.name = document.getElementById('dashboard-name');
        this.shared = $('.ui.shared.dashboard.checkbox');
        this.shared_header = document.getElementById('shared-dashboards');

        // Bind related Web GUI buttons
        this.bind();

        // Show shared dashboards header in a dropdown
        // only when such dashboards exist
        this.toggleSharedHeader();
    }

    /*
     * Bind dashboards related Web GUI buttons
     */
    bind() {
        // Button to save dashboard
        $('.ui.save.dashboard.button').on('click', (e) => {
            this.save();
        });

        // Button to load dashboard
        $('.ui.load.dashboard.button').on('click', (e) => {
            this.load();
        });

        // Button to delete dashboard
        $('.ui.delete.dashboard.button').on('click', (e) => {
            this.delete();
        });
    }

    /*
     * Save dashboard
     */
    save() {
        const data = {
                name:     this.name.value,
                shared:   this.shared.checkbox('is checked'),
                filters:  [this.application.filters.greenFilters, this.application.filters.redFilters],
                datetime: this.application.calendar.rangeStart.calendar('get date').toISOString().substr(0, 19) + '.000Z' + '..' +
                          this.application.calendar.rangeEnd.calendar(  'get date').toISOString().substr(0, 19) + '.000Z'
              }

        this.application.websocket.send('dashboard-save', JSON.stringify(data));
    }

    /*
     * Actions when dashboard is saved
     */
    saved(data) {
        const dashboard = JSON.parse(data);

        this.addDropdownEntry(dashboard.name, dashboard.shared);
        this.name.value = '';
        this.shared.checkbox('uncheck');

        if (dashboard.shared) {
            SHARED[dashboard.name] =     { filters:dashboard.filters, datetime:dashboard.datetime }
            this.toggleSharedHeader();
        } else {
            DASHBOARDS[dashboard.name] = { filters:dashboard.filters, datetime:dashboard.datetime }
        }
    }

    /*
     * Load selected dashboard
     */
    load() {
        const name =   this.dashboards.dropdown('get value'),
              option = this.dashboards.dropdown('get item', name)[0];

        // Skip if none is selected
        if (!option) return;

        const ds = (option.dataset.shared === 'true') ? SHARED[name] : DASHBOARDS[name],
              ts = ds.datetime.split('..');

        // Remove all existing graph elements
        this.application.graph.clearAll();

        // Restore saved filters
        FILTERS = ds.filters;
        this.application.filters.restore();
        this.application.graph.position();

        // Restore datetime range
        this.application.calendar.rangeStart.calendar('set date', new Date(ts[0]));
        this.application.calendar.rangeEnd.calendar(  'set date', new Date(ts[1]));
        this.application.calendar.setDatetimeRangeStr();
    }

    /*
     * Delete saved dashboard
     */
    delete() {
        const name =   this.dashboards.dropdown('get value'),
              option = this.dashboards.dropdown('get item', name)[0];

        // Skip if none is selected
        if (!option) return;

        this.application.websocket.send('dashboard-delete', name, option.dataset.shared);
    }

    /*
     * Actions after saved dashboard is deleted.
     * Receives darboard name and whether it's shared
     */
    deleted(name, shared) {
        // String to bool
        shared = (shared === 'true');

        // Remove item as dropdown's available option ..
        const item = this.findDropdownEntry(name, shared);
        if (item) item.remove();

        // .. and remove selection info
        this.dashboards.dropdown('remove selected', name);
        this.dashboards.dropdown('set exactly', '');

        // Remove from memory
        if (shared) {
            delete SHARED[name];
            this.toggleSharedHeader();
        } else {
            delete DASHBOARDS[name];
        }
    }

    /*
     * Add new loadable dashboard name to the dropdown's (not) shared group
     */
    addDropdownEntry(name, shared) {
        // Check whether such dasboard already exists
        const item = this.findDropdownEntry(name, shared);
        if (item) return;

        // Add new dropdown's option
        var option = document.createElement('div');

        option.className = 'item';
        option.innerText = name;

        if (shared) {
            option.setAttribute('data-value', name);
            option.setAttribute('data-shared', shared);
            this.dashboards.children('.menu').append(option);

        } else {
            option.setAttribute('data-value', name);
            this.dashboards.children('.menu').prepend(option);
        }

        this.dashboards.dropdown('refresh');
    }

    /*
     * Get dropdown's entry.
     * Receives darboard name and whether it's shared
     */
    findDropdownEntry(name, shared) {
        var item;

        this.dashboards.find('.menu .item').each(function() {
            // 'this' is current dropdown element in a loop
            if (this.getAttribute('data-value') === name && this.hasAttribute('data-shared') === shared) {
                item = this;
                return false;  // Breaking from 'each' loop
            }
        });

        return item;
    }

    /*
     * Show shared dashboards header in a dropdown
     * only when such dashboards exist
     */
    toggleSharedHeader() {
        if (Object.keys(SHARED).length === 0)
            this.shared_header.style.display = 'none';
        else
            this.shared_header.style.display = 'block';
    }
}
