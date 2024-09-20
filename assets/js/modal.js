/*
 * Modal window notifications
 */
class Modal {
    constructor() {
        // Pointers to the window HTML elements
        this.containerOk =    $('.ui.basic.ok.modal');
        this.containerEmpty = $('.ui.basic.empty.modal');
    }

    /*
     * Empty notification without actions
     */
    empty(header, text, icon) {
        //console.log('empty', header, text);

        if (header) {
            // Remove previous message if exists
            if (!text)
                text = '';

            this.containerEmpty.find('.header').html('<i class="' + icon + ' icon"></i>' + header);
            this.containerEmpty.find('.content').html(text);
            this.containerEmpty.modal('show');
        }
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

            this.containerOk.find('.header').html('<i class="exclamation circle icon"></i>' + header);
            this.containerOk.find('.content').html(text);
            this.containerOk.modal('show');
        }
    }

    /*
     * Notify that something is wrong
     */
    error(header, text) {
        //console.log(header, text);

        this.containerOk.find('.header').html('<i class="exclamation triangle icon"></i>' + header);
        this.containerOk.find('.content').html(text.replace(/^([\w -"']*):/, '<span class="red_fg">$1</span>:'));
        this.containerOk.modal('show');
    }

    /*
     * Close any modal window
     */
    close() {
        this.containerOk.modal('hide');
        this.containerEmpty.modal('hide');
    }
}
