/*
 * Websocket connection for a client-server-client communication.
 *
 * For the incoming messages checks its type and executes related actions.
 * For the outgoing messages builds a JSON object and sends it to the server
 */
class Websocket {
    constructor(application) {
        // Pointer to the current page core
        this.application = application;

        // Attempts to reestablish a Websocket connection
        this.attempts = 0;

        // Create a Websocket connection
        this.init();
    }

    /*
     * Init a Websocket connection
     */
    init() {
        if (!'WebSocket' in window) {
            this.application.modal.error('Upgrade is required!', 'Browser does not support Websockets!');
            return;
        }

        this.ws = new WebSocket('wss://' + window.location.host + '/ws');

        this.ws.onopen =  this.onOpen.bind(this);
        this.ws.onclose = this.onClose.bind(this);

        // Process incoming messages
        this.ws.onmessage = this.onMessage.bind(this);
    }

    /*
     * Websocket connection is established
     */
    onOpen() {
        //console.log('Websocket connection established');

        // Request upload lists if a "Profile" page is open
        if (document.getElementById('newPassword'))
            this.send('upload-lists');

        // Some features are search page related only
        if (this.application.search) {
            // Prevent searching while Websocket connection is not established yet
            this.application.search.searchBtn.removeClass('disabled');

            // Run user's enabled filters when Websocket is established first time only
            if (this.attempts === 0)
                this.application.filters.restore();
        }

        // Reset the tries back to 1 since we have a new connection opened
        this.attempts = 0;
    }

    /*
     * Check incoming message type and execute related actions
     */
    onMessage(e) {
        console.log(e.data);
        const message = JSON.parse(e.data);

        switch(message.type) {
            // Result of the search query
            case 'results':
                this.application.search.processResults(message.extra, message.data);
                break;

            // Response to the user's request to find selected nodes common attributes and neighbors
            case 'common':
                this.application.search.processCommon(message.data, message.extra);
                break;

            // Notification from server
            case 'notification':
                this.application.notifications.appendNow(message.extra, message.data);
                break;

            // Regenerated user's auth UUID
            case 'uuid':
                document.getElementById('uuid').innerText = message.data;
                break;

            // Notes for the selected graph element
            case 'notes':
                this.application.graph.saveNotesBtn.removeClass('disabled loading');
                this.application.graph.notes.value = message.data;
                break;

            // Notes for the selected graph element saved
            case 'notes-set':
                this.application.graph.saveNotesBtn.removeClass('disabled loading');
                break;

            // Notes for the selected graph element where NOT saved
            case 'notes-error':
                this.application.graph.saveNotesBtn.removeClass('disabled loading');
                this.application.graph.application.modal.error('Error!', message.data);
                break;

            // User password was deleted
            case 'account-reset':
                this.application.modal.ok('Done!', 'User password was deleted. <strong>Sign up</strong> again with a new one!');
                break;

            // Account deleted by an administrator
            case 'account-deleted':
                document.getElementById('username-'+message.data).remove();
                break;

            // Dashboard saved
            case 'dashboard-saved':
                this.application.dashboards.saved(message.data);
                break;

            // Dashboard deleted
            case 'dashboard-deleted':
                this.application.dashboards.deleted(message.data, message.extra);
                break;

            // User's upload queue to be processed and downloads list
            case 'upload-lists':
                this.application.actions.lists(message.data, message.extra);
                break;

            // List of indicators was uploaded
            case 'uploaded':
                if (this.application.actions)
                    this.application.actions.uploaded(message.data);
                break;

            // Processing of the list of indicators completed
            case 'upload-processed':
                if (this.application.actions)
                    this.application.actions.uploadProcessed(message.data);
                break;

            // Notify that the latest action finished without errors
            case 'ok':
                this.application.modal.ok('Done!');
                break;

            // Notify about some error
            case 'error':
                this.application.modal.error(message.extra, unescape(message.data));
                break;

            // Ping from time to time for the Websocket connection to stay alive
            case 'ping':
                break;
        }
    }

    /*
     * Websocket connection is closed
     */
    onClose(e) {
        this.ws = null;

        // Some features are search related only
        if (this.application.search) {
            // Prevent searching until Websocket connection is established
            this.application.search.searchBtn.removeClass('loading');
            this.application.search.searchBtn.addClass('disabled');
        }

        if (this.attempts === 0)
            this.application.notifications.appendNow('error', 'WebSocket connection was interrupted, reconnect will be attempted automatically.');

        // Connection was interrupted, try to reconnect
        if (this.attempts < 60) {
            setTimeout(() => {
                this.attempts++;
                this.init();
            }, 5000);

        } else {
            this.application.notifications.appendNow('error', 'Reconnection to the server has stopped, refresh the page to try again.');
        }
    }

    /*
     * Send message as a JSON-formatted string to the server.
     *
     * Receives:
     *     type  - message type: sql, notes-save, etc.
     *     data  - any data to send to the server
     *     extra - possible additional data
     */
    send(type, data, extra) {
        const message = {
            type:  type,
            data:  data,
            extra: extra
        }

        if (this.ws)
            this.ws.send(JSON.stringify(message));
    }
}
