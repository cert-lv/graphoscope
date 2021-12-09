/*
 * Built-in documentation webpage core.
 * Allows to render Markdown code, one source for each section
 */
window.addEventListener('DOMContentLoaded', function() {

    class Docs {
        constructor() {
            // Prepare modal window first
            this.modal = new Modal();

            // Web GUI notifications
            this.notifications = new Notifications(this);

            // Init Semantic UI dynamic elements
            this.prepareSemantic();

            // Init Markdown rendering engine
            const md = window.markdownit().
                use(window.markdownItAnchor).
                use(window.markdownItTocDoneRight);

            // Render UI docs
            var result = md.render(document.getElementById('ui-md').innerText);
            document.getElementById('ui-md').innerHTML = result;

            // Render search docs
            result = md.render(document.getElementById('search-md').innerText);
            document.getElementById('search-md').innerHTML = result;

            // Render administration section
            result = md.render(document.getElementById('admin-md').innerText);
            document.getElementById('admin-md').innerHTML = result;
        }

        /*
         * Init Semantic UI dynamic elements
         */
        prepareSemantic() {
            $('.ui.dropdown').dropdown();
            $('.ui.secondary.menu .item').tab();
        }
    }

    const docs = new Docs();

});
