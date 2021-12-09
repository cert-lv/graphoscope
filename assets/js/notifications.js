/*
 * Notifications.
 *
 * May appear multiple windows grouped in a column.
 * Mouse click hides a single notification
 */
class Notifications {
    constructor(section) {
        // Pointer to the current webpage core
        this.section = section;

        // Notification windows container
        this.container = document.getElementById('notifications-container');

        // The amount of available notifications,
        // created when user was offline
        this.count = document.getElementById('notifications-count');

        // Bind Web GUI buttons
        this.bind();
    }

    /*
     * Bind Web GUI buttons
     */
    bind() {
        // Show notifications that were created when user was offline
        $('.ui.label.count').on('click', () => {
            this.show();
        });
    }

    /*
     * Show notifications that were created when user was offline
     */
    show() {
        // Create notification windows
        this.insert();
        // Clean notifications to prevent them showing again
        this.clean();
    }

    /*
     * Create a new notification.
     *
     * Receives:
     *     - type of event. 'error', 'info', etc.
     *     - timestamp of the event
     *     - text to display
     */
    append(type, timestamp, text) {
        var el = document.createElement('div'),
            ic = document.createElement('i'),
            he = document.createElement('div'),
            co = document.createElement('div'),
            ts = document.createElement('div');

        el.className = 'ui segment notification';
        he.className = 'ui header';
        co.className = 'content';
        ts.className = 'ts grey_fg';

        // Set dynamic icon based on notification's type
        if (type === 'error') {
            ic.className = 'bolt icon red_fg';
            he.innerText = 'Error!';
        } else if (type === 'info') {
            ic.className = 'envelope open outline icon green_fg';
            he.innerText = 'Info!';
        }

        // Set content
        co.innerHTML = text;
        ts.innerText = timestamp;

        el.appendChild(ic);
        el.appendChild(he);
        el.appendChild(co);
        el.appendChild(ts);

        // Allow to hide notification on click.
        // Use additional function to create a closure for the 'el' variable
        (function(el) {
            el.onclick = function() {
                el.parentNode.removeChild(el);
            }
        })(el);

        // Insert notification in a container
        this.container.appendChild(el);
    }

    /*
     * Append a new notification with a current date/time.
     *
     * Receives:
     *     - type of event. 'error', 'info', etc.
     *     - text to display
     */
    appendNow(type, text) {
        this.append(type, this.formatDate(new Date()), text);
    }

    /*
     * Format JavaScript default datetime object.
     * Example: 'Fri Dec 03 2021 14:21:09 GMT+0200 (Eastern European Standard Time)' -> '3 Dec 2021 14:21:09'
     */
    formatDate(date) {
        const strArray = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'],
              D = date.getDate(),
              M = strArray[date.getMonth()],
              Y = date.getFullYear(),
              h = (date.getHours() < 10 ? '0' : '') + date.getHours(),
              m = (date.getMinutes() < 10 ? '0' : '') + date.getMinutes(),
              s = (date.getSeconds() < 10 ? '0' : '') + date.getSeconds();


        return '' + D + ' ' + M + ' ' + Y + ' ' + h + ':' + m + ':' + s;
    }

    /*
     * UTC to localtime
     */
    // UTCtoLocal(date) {
    //     const localOffset = date.getTimezoneOffset() * 60000,
    //           localTime = date.getTime();

    //     return localTime - localOffset;
    // }

    /*
     * Show notifications that were created when user was offline
     */
    insert() {
        for (var i=0; i < NOTIFICATIONS.length; i++) {
            const notification = NOTIFICATIONS[i];

            var el = document.createElement('div'),
                ic = document.createElement('i'),
                he = document.createElement('div'),
                co = document.createElement('div'),
                ts = document.createElement('div');

            el.className = 'ui segment notification';
            he.className = 'ui header';
            co.className = 'content';
            ts.className = 'ts grey_fg';

            // Set dynamic icon based on notification's type
            if (notification.type === 'error') {
                ic.className = 'bolt icon red_fg';
                he.innerText = 'Error!';
            } else if (notification.type === 'info') {
                ic.className = 'envelope open outline icon green_fg';
                he.innerText = 'Info!';
            }

            // Set content
            co.innerHTML = notification.message;
            ts.innerText = notification.ts;

            el.appendChild(ic);
            el.appendChild(he);
            el.appendChild(co);
            el.appendChild(ts);

            // Allow to hide notification on click.
            // Use additional function to create a closure for the 'el' variable
            (function(el) {
                el.onclick = function() {
                    el.parentNode.removeChild(el);
                }
            })(el);

            // Insert notification in a container
            this.container.appendChild(el);
        }
    }

    /*
     * Clean notifications on a server-side to prevent them showing again
     */
    clean() {
        this.section.websocket.send('notifications');

        // Clean the amount of new notifications on a Web GUI
        this.count.innerHTML = '';
        // Clean a global variable
        NOTIFICATIONS = [];
    }
}
