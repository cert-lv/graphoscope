/*
 * Notification about new service's features.
 * Appears once for each user
 */
class Features {
    constructor(application) {
        // Pointer to the main page core
        this.application = application;

        // Show new features modal window
        this.show();
    }

    /*
     * Show new features modal window.
     * FEATURES global variable is undefined if current user is already informed
     */
    show() {
        if (typeof FEATURES === 'undefined')
            return;

        var list = '<ul>';

        for (var i = 1; i < FEATURES.length; i++)
            list += '<li>' + FEATURES[i] + '</li>';

        list += '</ul><span class="features-date grey_fg">Date: ' + FEATURES[0] + '</span>';

        this.application.modal.ok('New features!', list);
    }
}