/*
 * Modal window notifications
 */
class Modal {
    constructor() {
        // Pointer to the window HTML element
        this.container = $('.ui.basic.modal');
    }

    /*
     * Notify that everything is Ok
     */
    ok(header, text) {
        //console.log('ok', header, text);

        if (header) {
            // Remove previous message if exists
            if (!text)
                text = '';

            this.container.find('.header').html('<i class="exclamation circle icon"></i>' + header);
            this.container.find('.content').html(text);
            this.container.modal('show');
        }
    }

    /*
     * Notify that something is wrong
     */
    error(header, text) {
        //console.log(header, text);

        this.container.find('.header').html('<i class="exclamation triangle icon"></i>' + header);
        this.container.find('.content').html(text);
        this.container.modal('show');
    }
}
