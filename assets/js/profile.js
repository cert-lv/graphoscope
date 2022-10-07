/*
 * Profile webpage core.
 *
 * Allows to save personal and account settings, launch long term actions
 */
window.addEventListener('DOMContentLoaded', function() {

    class Profile {
        constructor() {
            // Prepare modal window first
            this.modal = new Modal();

            // Notifications
            this.notifications = new Notifications(this);

            // Websocket connection for the client-server-client communication
            this.websocket = new Websocket(this);

            // User's personal settings management
            this.options = new Options(this);

            // User's account management
            this.account = new Account(this);

            // Datetime range picker
            this.calendar = new Calendar();

            // Long-term user actions
            this.actions = new UserActions(this);
        }
    }

    const profile = new Profile();

});
