/*
 * Admin webpage core.
 *
 * Allows to set global graph settings and
 * manage registered users
 */
window.addEventListener('DOMContentLoaded', function() {

    class Admin {
        constructor() {
            // Prepare modal window first
            this.modal = new Modal();

            // Web GUI notifications
            this.notifications = new Notifications(this);

            // Websocket connection for the client-server-client communication
            this.websocket = new Websocket(this);

            // Global graph settings
            this.settings = new Settings(this);

            // Registered users management
            this.users = new Users(this);
        }
    }

    const admin = new Admin();

});
