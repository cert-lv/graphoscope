/*
 * Datetime range picker
 */
class Calendar {
    constructor() {
        // Web GUI elements for the easier access
        this.daterange =  $('.ui.daterange.button');
        this.rangeStart = $('#rangestart');
        this.rangeEnd =   $('#rangeend');

        // Bind Web GUI buttons
        this.bind();
        // Init Semantic UI dynamic elements
        this.prepareSemantic();
    }

    /*
     * Bind actions to the Web GUI elements
     */
    bind() {
        // Datetime custom ranges
        $('.ui.lastHour.button').on('click', (e) => {
            this.setDatetimeRange('lastHour');
        });
        $('.ui.last12hours.button').on('click', (e) => {
            this.setDatetimeRange('last12hours');
        });
        $('.ui.lastDay.button').on('click', (e) => {
            this.setDatetimeRange('lastDay');
        });
        $('.ui.lastWeek.button').on('click', (e) => {
            this.setDatetimeRange('lastWeek');
        });
        $('.ui.lastMonth.button').on('click', (e) => {
            this.setDatetimeRange('lastMonth');
        });
        $('.ui.last6months.button').on('click', (e) => {
            this.setDatetimeRange('last6months');
        });
    }

    /*
     * Init Semantic UI dynamic elements
     */
    prepareSemantic() {
        const calendarFormatter = {
            date: (date, settings) => {
                if (!date) return '';
                var day = date.getDate() + '';
                if (day.length < 2) {
                    day = '0' + day;
                }
                var month = (date.getMonth() + 1) + '';
                if (month.length < 2) {
                    month = '0' + month;
                }
                var year = date.getFullYear();
                return day + '.' + month + '.' + year;
            }
        }

        this.rangeStart.calendar({
            type: 'datetime',
            initialDate: new Date(Date.now() - 3600 * 24 * 1000),
            text: {
                days: ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa'],
            },
            firstDayOfWeek: 1,
            ampm: false,
            formatter: calendarFormatter,
            onSelect: (date, mode) => { // Without this both calendars are not always dynamically updating
                this.rangeEnd.calendar('set startDate', date);
                this.setDatetimeRangeStr();
            },
            endCalendar: this.rangeEnd
        });

        this.rangeEnd.calendar({
            type: 'datetime',
            initialDate: new Date(),
            text: {
                days: ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa'],
            },
            firstDayOfWeek: 1,
            ampm: false,
            formatter: calendarFormatter,
            onSelect: (date, mode) => { // Without this both calendars are not always dynamically updating
                this.rangeStart.calendar('set endDate', date);
                this.setDatetimeRangeStr();
            },
            startCalendar: this.rangeStart
        });

        // Set current & default datetime range string
        this.setDatetimeRange('lastDay');
    }

    /*
     * Set datetime custom ranges of calendars
     */
    setDatetimeRange(period) {
        var startDate = new Date(),
            endDate = new Date();

        switch(period) {
            case 'lastHour':
                startDate.setHours(startDate.getHours() - 1);
                break;

            case 'last12hours':
                startDate.setHours(startDate.getHours() - 12);
                break;

            case 'lastDay':
                startDate.setHours(startDate.getHours() - 24);
                break;

            case 'lastWeek':
                startDate.setHours(startDate.getHours() - 24*7);
                break;

            case 'lastMonth':
                startDate.setMonth(startDate.getMonth() - 1);
                break;

            case 'last6months':
                startDate.setMonth(startDate.getMonth() - 6);
                break;
        }

        // Set range
        this.rangeStart.calendar('set date', startDate);
        this.rangeEnd.calendar('set date', endDate);

        this.rangeStart.calendar('refresh');
        this.rangeEnd.calendar('refresh');

        this.setDatetimeRangeStr();
    }

    /*
     * Set datetime custom range string
     */
    setDatetimeRangeStr() {
        var startStr = this.rangeStart.calendar('get date'),
            endStr = this.rangeEnd.calendar('get date');

        // Skip if start date goes after end date
        if (endStr === null) { return }

        var datestring = ("0" + startStr.getDate()).slice(-2)    + "." +
                         ("0"+(startStr.getMonth()+1)).slice(-2) + "." +
                         startStr.getFullYear()                  + " " +
                         ("0" + startStr.getHours()).slice(-2)   + ":" +
                         ("0" + startStr.getMinutes()).slice(-2) +
                         '&nbsp;&nbsp;&nbsp;-&nbsp;&nbsp;&nbsp;' +
                         ("0" + endStr.getDate()).slice(-2)      + "." +
                         ("0"+(endStr.getMonth()+1)).slice(-2)   + "." +
                         endStr.getFullYear()                    + " " +
                         ("0" + endStr.getHours()).slice(-2)     + ":" +
                         ("0" + endStr.getMinutes()).slice(-2);

        this.daterange.html(datestring);
    }
}