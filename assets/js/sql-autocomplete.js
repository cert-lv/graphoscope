/*
 * Generate valid autocomplete SQL suggestions for keywords, tables, or columns
 *
 * https://www.npmjs.com/package/sql-autocomplete
 * https://www.w3schools.com/howto/tryit.asp?filename=tryhow_js_autocomplete
 */
class SQLAutocomplete {
    constructor(input) {
        // Search input
        this.input = input;

        // Data sources dropdown
        this.source = $('.source.dropdown');

        // Container of the autocomplete fields
        this.container = document.getElementById('autocomplete');

        // Currently highlighted autocompleted field
        this.currentFocus = -1;

        // Close dropdown when user clicks in the document
        document.addEventListener('click', (e) => {
            this.closeList(e.target);
        });
    }

    /*
     * Build a dropdown of options.
     *
     * Accepts cursor position in the input to get the needed field,
     * not the last one in a string
     */
    build(pos) {
        // Get word under a cursor for the autocomplete
        var res = this.getFieldAt(pos),
            source = this.source.dropdown('get value'),
            left = res[0],
            right = res[1],
            word = res[2].trim();

        // Close any already open lists of autocompleted values
        this.closeList();

        // An array of unique fields
        var unique = [];
        // Currently highlighted dropdown option
        this.currentFocus = -1;

        // Build a data source related unique fields list
        if (source === 'global') {
            var map = {};

            for (var key in FIELDS) {
                var list = FIELDS[key],
                    len = list.length;

                for (var i = 0; i < len; i++) {
                    map[list[i]] = true;
                }
            }

            unique = Object.keys(map);
        } else {
            unique = FIELDS[source];
        }

        unique.sort();

        // For each item in the array...
        for (const field of unique) {
            // Check if the item starts with the same letters as the field value,
            // exclude item which already completes current field
            if (field.substr(0, word.length).toUpperCase() === word.toUpperCase() && field !== word) {
                // Create a DIV element for each matching element
                let option = document.createElement('div');

                // Make the matching letters bold
                option.innerHTML = '<strong>' + field.substr(0, word.length) + '</strong>';
                option.innerHTML += field.substr(word.length);

                // Insert data attribute that will hold current array item's value
                option.dataset.field = field;

                // Execute a function when someone clicks on the item value (DIV element)
                option.addEventListener('click', (e) => {
                    let value = option.dataset.field;

                    // If field name contains special characters - backticks must be added
                    if (value.includes('-') || value.includes('|') || value.includes('/'))
                        value = '`' + value + '`';

                    // Insert the value for the autocomplete text field
                    this.input.value = this.input.value.substring(0, left) +
                                       value +
                                       this.input.value.substring(right);

                    // Close the list of autocompleted values,
                    // or any other open lists of autocompleted values
                    this.closeList();
                    this.currentFocus = -1;

                    // Set caret position at the end of the selected field name
                    this.setCaretPosition((this.input.value.substring(0, left) + value).length);
                });

                this.container.appendChild(option);
            }
        }

        if (this.container.children.length > 0)
            this.container.style.display = 'block';
    }

    /*
     * Set caret position at the end of the selected field name
     */
    setCaretPosition(pos) {
        // Modern browsers
        if (this.input.setSelectionRange) {
            this.input.focus();
            this.input.setSelectionRange(pos, pos);

        // IE8 and below
        } else if (this.input.createTextRange) {
            var range = this.input.createTextRange();
            range.collapse(true);
            range.moveEnd('character', pos);
            range.moveStart('character', pos);
            range.select();
        }
    }

    /*
     * Classify an item as 'active'
     */
    setActive(list, i) {
        if (list.length === 0) return false;

        // Start by removing the 'active' class on all items
        this.removeActive(list);

        this.currentFocus += i;
        if (this.currentFocus >= list.length) this.currentFocus = 0;
        if (this.currentFocus < 0) this.currentFocus = (list.length - 1);

        // Add class 'autocomplete-active'
        var item = list[this.currentFocus];
        item.classList.add('autocomplete-active');
        item.scrollIntoView({block:'center'});
    }

    /*
     * Remove the 'active' class from all autocomplete items
     */
    removeActive(list) {
        for (var i = 0; i < list.length; i++) {
            list[i].classList.remove('autocomplete-active');
        }
    }

    /*
     * Close autocomplete list
     */
    closeList(elmnt) {
        if (elmnt != this.input) {
            this.container.innerHTML = '';
            this.container.style.display = 'none';
        }
    }

    getFieldAt(pos) {
        // Perform type conversions
        pos = Number(pos) >>> 0;

        // Search for the word's beginning and the end
        var left = this.input.value.slice(0, pos).search(/[\w\.]+$/),
            right = this.input.value.slice(pos).search(/[^\w.]/);

        // If no field name beginning is given -
        // 'Tab' was pressed to get all the possible fields
        if (left === -1)
            left = pos;

        // The last word in the string is a special case
        if (right < 0)
            return [left, this.input.value.length, this.input.value.slice(left)];

        // Return the word, using the located bounds to extract it from the string
        return [left, right + pos, this.input.value.slice(left, right + pos)];
    }
}
