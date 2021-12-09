/*
 * Global graph UI settings management
 */
class Settings {
    constructor(profile) {
        // Pointer to the admin page core
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
        $('.ui.save.settings.button').on('click', (e) => {
            this.save();
        });
    }

    /*
     * Save settings
     */
    save() {
        const settings = document.getElementById('nodesize').value + ',' +
                         document.getElementById('borderwidth').value + ',' +
                         document.getElementById('bgcolor').value.toLowerCase() + ',' +
                         document.getElementById('bordercolor').value.toLowerCase() + ',' +
                         document.getElementById('nodefontsize').value + ',' +
                         $('.ui.checkbox.shadow').checkbox('is checked') + ',' +
                         document.getElementById('edgewidth').value + ',' +
                         document.getElementById('edgecolor').value.toLowerCase() + ',' +
                         document.getElementById('edgefontsize').value + ',' +
                         document.getElementById('edgefontcolor').value.toLowerCase() + ',' +
                         $('.ui.checkbox.arrow').checkbox('is checked') + ',' +
                         $('.ui.checkbox.smooth').checkbox('is checked') + ',' +
                         $('.ui.checkbox.hover').checkbox('is checked') + ',' +
                         $('.ui.checkbox.multiselect').checkbox('is checked') + ',' +
                         $('.ui.checkbox.hideedgesondrag').checkbox('is checked');

        // Send settings to the server
        this.profile.websocket.send('settings', settings);
    }
}
