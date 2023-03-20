/*
 * Administrators only actions
 */
class AdminActions {
    constructor(admin) {
        // Pointer to the admin page core
        this.admin = admin;

        // Bind Web GUI buttons
        this.bind();
    }

    /*
     * Bind actions to the Web GUI elements
     */
    bind() {
        // Refresh the list of fields to query for the Web GUI autocomplete
        $('.ui.reload.button').on('click', (e) => {
            this.reloadPlugins();
        });
    }

    /*
     * Reload collectors and processors
     */
    reloadPlugins() {
        this.admin.websocket.send('reload');
    }
}
