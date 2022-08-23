/*
 * User's personal settings
 */
class Options {
    constructor(profile) {
        // Pointer to the profile page core
        this.profile = profile;

        // Init Semantic UI dynamic elements
        this.prepareSemantic();

        // Bind Web GUI buttons
        this.bind();
    }

    /*
    * Init Semantic UI dynamic elements
    */
    prepareSemantic() {
        $('.ui.dropdown').dropdown();
        $('.ui.secondary.menu .item').tab();
    }

    /*
     * Bind Web GUI buttons
     */
    bind() {
        // Button to save settings
        $('.ui.save.profile.button').on('click', (e) => {
            this.save();
        });
    }

    /*
     * Save settings
     */
    save() {
        const options = document.getElementById('stabilization').value + ',' +
                        document.getElementById('limit').value + ',' +
                        $('.ui.checkbox.debug').checkbox('is checked');

        // Send to the server
        this.profile.websocket.send('options', options);
    }
}
