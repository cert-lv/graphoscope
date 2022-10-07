/*
 * User management.
 * Administrators can reset passwords, delete users, etc.
 */
class Users {
    constructor(admin) {
        // Pointer to the admin page core
        this.admin = admin;

        // Bind Web GUI buttons
        this.bind();

        // Check whether server has returned an error during the page load
        this.checkLoadError();
    }

    /*
     * Bind Web GUI buttons
     */
    bind() {
        // Access 'this' from 'function()'
        const users = this;

        // Button to clear user's password
        $('.ui.button.reset').on('click', (e) => {
            users.apply(e.target.dataset.username, 'reset-password');
        });

        // Button to delete account
        $('.ui.button.delete').on('click', (e) => {
            users.apply(e.target.dataset.username, 'delete');
        });

        // Toggle admin rights
        $('.ui.checkbox.admin').checkbox({
            onChange: function() {
                users.apply(this.dataset.username, 'admin-' + this.checked);
            }
        });
    }

    /*
     * Check whether server has returned an error during the page load
     */
    checkLoadError() {
        if (MSG !== '')
            this.admin.modal.error('Something went wrong!', MSG);
    }

    /*
     * Modify selected user.
     * Receives a user name and an action to apply
     */
    apply(username, action) {
        this.admin.websocket.send('users', username+','+action);
    }
}
