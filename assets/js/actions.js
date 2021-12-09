/*
 * Long term actions
 */
class Actions {
    constructor(profile) {
        // Pointer to the profile page core
        this.profile = profile;

        // Web GUI elements for the easier access
        this.uploadBtn = $('.ui.upload.button');
        this.inputFile = document.getElementById('inputFile');
        this.inputText = document.getElementById('inputText');

        // Upload In-Out queues
        this.queue =           document.getElementById('queue');
        this.queueHeader =     document.getElementById('queueHeader');
        this.downloads =       document.getElementById('downloads');
        this.downloadsHeader = document.getElementById('downloadsHeader');
        // Numbers of files
        this.queueN = 0;
        this.downloadsN = 0;

        // Bind Web GUI buttons
        this.bind();
        // Init Semantic UI dynamic elements
        this.prepareSemantic();
    }

    /*
     * Bind actions to the Web GUI elements
     */
    bind() {
        // File selection for the upload
        $('#inputText').on('click', () => {
            this.inputFile.click();
        });

        $('.ui.icon.button.select').on('click', () => {
            this.inputFile.click();
        });

        // Show selected file's name
        $('input:file', '.ui.action.input').on('change', (e) => {
            if (e.target.files.length > 0) {
                var name = e.target.files[0].name;
                this.inputText.value = name;
            }
        });

        // Upload file with indicators
        $('.ui.upload.button').on('click', () => {
            this.upload();
        });
    }

    /*
     * Init Semantic UI dynamic elements
     */
    prepareSemantic() {
        $('.ui.dropdown').dropdown();

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

        $('#rangestart').calendar({
            type: 'datetime',
            initialDate: new Date(Date.now() - 3600 * 24 * 1000),
            firstDayOfWeek: 1,
            ampm: false,
            formatter: calendarFormatter,
            endCalendar: $('#rangeend')
        });
        $('#rangeend').calendar({
            type: 'datetime',
            initialDate: new Date(),
            firstDayOfWeek: 1,
            ampm: false,
            formatter: calendarFormatter,
            startCalendar: $('#rangestart')
        });
    }

    /*
     * Build user's upload queue to process and download list.
     * 
     * Recieves:
     *   i - list of input files to be processed,
     *   o - list of processed files available for the download
     */
    lists(i, o) {
        const im = JSON.parse(i),
              om = JSON.parse(o);

        if (im.length > 0) {
            for (var n = 0; n < im.length; n++) {
                this.uploaded(im[n]);
            }
        }

        if (om.length > 0) {
            this.downloadsHeader.style.display = 'block';

            for (var n = 0; n < om.length; n++) {
                this.appendDownloadLink(om[n]);
            }
        }
    }

    /*
     * Upload chosen file
     */
    upload() {
        const actions =   this,
              source =    $('.ui.source.dropdown').dropdown('get value'),
              startTime = $('#rangestart').calendar('get date').toISOString().substr(0, 19) + '.000Z',
              endTime =   $('#rangeend').calendar('get date').toISOString().substr(0, 19) + '.000Z',
              format =    $('.ui.format.dropdown').dropdown('get value'),
              field =     document.getElementById('default_field').value;

        if (this.inputFile.files.length > 0) {

            // Check user input
            if (this.inputFile.files[0].size > MAXUPLOADSIZE) {
                this.profile.modal.error('Can\'t upload!', 'File size is too large, max <strong>' + (MAXUPLOADSIZE/1048576).toFixed(1) + ' MB</strong> are allowed!');
                return;
            }

            this.uploadBtn.addClass('disabled loading');

            var reader = new FileReader();

            // Closure to capture the file information
            reader.onload = (function(theFile) {
                return function(e) {
                    var formData = new FormData(),
                        xhttp = new XMLHttpRequest();

                    formData.append('file', theFile);

                    // Set POST method and ajax file path
                    xhttp.open('POST', '/upload?source='+encodeURIComponent(source)+'&startTime='+startTime+'&endTime='+endTime+'&format='+format+'&field='+encodeURIComponent(field), true);

                    // Call on request change state
                    xhttp.onreadystatechange = function() {
                        if (this.readyState === 4) {
                            if (this.status === 200) {
                                if (this.responseText === 'ok') {
                                    actions.profile.notifications.appendNow('info', 'File uploaded! Wait for a notification that file is processed, download link will be added below.');
                                    document.getElementById('inputFile').value = '';
                                    document.getElementById('inputText').value = '';

                                } else {
                                    actions.profile.modal.error('File not uploaded!', this.responseText);
                                }

                            } else {
                                if (this.responseText)
                                    actions.profile.modal.error('File not uploaded!', this.responseText + ' (' + this.statusText + ')');
                                else
                                    actions.profile.modal.error('System internal error', 'Check browser\'s console for the possible details.');
                            }

                            actions.uploadBtn.removeClass('disabled loading');
                        }
                    }

                    // Send request with data
                    xhttp.send(formData);
                }
            })(this.inputFile.files[0]);

            // Read the file as a data URL
            reader.readAsText(this.inputFile.files[0]);

        } else {
            this.profile.modal.error('No file is chosen!', '');
            this.uploadBtn.removeClass('disabled loading');
        }
    }

    /*
     * Append filename to the queue when it's uploaded
     */
    uploaded(filename) {
        const el = document.createElement('div');

        el.id = encodeURIComponent(filename);
        el.className = 'ui orange message';
        el.innerHTML = '<strong>'+this.formatName(filename)+'</strong>, upload date: <strong>'+this.formatDate(filename)+'</strong>';

        this.queue.appendChild(el);
        this.queueHeader.style.display = 'block';

        // Increase counter
        this.queueN += 1;
    }

    /*
     * Move filename to the download list when it's processed
     */
    uploadProcessed(filename) {
        const inQueue = document.getElementById(encodeURIComponent(filename));

        // Remove from the queue and add to the downloads list
        inQueue.parentNode.removeChild(inQueue);
        this.appendDownloadLink(filename);

        this.profile.notifications.appendNow('info', 'Processing of the <span class="ui orange text">'+this.formatName(filename)+'</span> is complete!');

        // Decrease counter and hide the header if it's zero
        this.queueN -= 1;

        if (this.queueN === 0)
            this.queueHeader.style.display = 'none';
    }

    /*
     * Create a new link to download the results of the frocessd file
     */
    appendDownloadLink(filename) {
        const el = document.createElement('div');

        // Append to the downloads list
        el.className = 'ui blue message';
        el.innerHTML = '<strong>'+this.formatName(filename)+'</strong>, ' +
                       'upload date: <strong>'+this.formatDate(filename)+'</strong>, ' +
                       'download link: <a href="/download?file='+encodeURIComponent(filename)+'" download="'+filename+'"><i class="download icon"></i></a>';

        this.downloads.appendChild(el);
        this.downloadsHeader.style.display = 'block';

        // Increase counter
        this.downloadsN += 1;
    }

    /*
     * Format date from the uploaded file name.
     *
     * Example: 0211126-130531+02-action.txt -> 2021-11-26 13:05:31 +02:00
     */
    formatDate(filename) {
        return filename.substring(0, 4)   + '-' +
               filename.substring(4, 6)   + '-' +
               filename.substring(6, 8)   + ' ' +
               filename.substring(9, 11)  + ':' +
               filename.substring(11, 13) + ':' +
               filename.substring(13, 15) + ' ' +
               filename.substring(15, 18) + ':00';
    }

    /*
     * Get initial uploaded file name.
     *
     * Example: 20211126-130531+02-action.txt -> action.txt
     */
    formatName(filename) {
        return filename.substring(19, filename.length);
    }
}
