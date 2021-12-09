/*
 * User's account management
 */
class Account {
    constructor(profile) {
        // Pointer to the profile page core
        this.profile = profile;

        // Bind Web GUI buttons
        this.bind();
    }

    /*
     * Bind Web GUI buttons
     */
    bind() {
        // Regenerate UUID
        $('.ui.uuid.button').on('click', (e) => {
            this.regenerateUUID();
        });

        // Save account data
        $('.ui.save.account.button').on('click', (e) => {
            this.save();
        });

        // Delete own account
        $('.ui.delete.account.button').on('click', (e) => {
            this.profile.websocket.send('account-delete');
            window.location.replace('/signin');
        });
    }

    /*
     * Regenerate auth UUID
     */
    regenerateUUID() {
        this.profile.websocket.send('uuid');
    }

    /*
     * Save account data
     */
    save() {
        const pass =       document.getElementById('newPassword').value,
              passRepeat = document.getElementById('newPasswordRepeat').value;

        // Compare both passwords
        if (pass !== passRepeat) {
            this.profile.modal.error('Unsuccessful request!', 'Passwords do not match!');
            return;
        }

        this.profile.websocket.send('account-save', pass);
    }
}
