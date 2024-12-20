package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// Regex to validate a datetime value
	reDatetime = regexp.MustCompile(`^\d\d\d\d-\d\d-\d\dT\d\d:\d\d:\d\d\.\d\d\dZ$`)
)

/*
 * Structure to describe user's uploaded list of indicators
 * to be processed in a background
 */
type Upload struct {
	// Data source to query
	Source string

	// Datetime range to search in
	StartTime string
	EndTime   string

	// Output format
	Format string

	// Data source's field to check
	// in case is not specified in the uploaded file
	Field string
}

/*
 * Setup the indicators upload feature
 */
func setupUpload() error {
	err := os.MkdirAll(config.Upload.Path+"/queue/", 0755)
	if err != nil {
		return err
	}

	err = os.MkdirAll(config.Upload.Path+"/processed/", 0755)
	if err != nil {
		return err
	}

	// Process uploaded files again if any process was previously interrupted
	go reProcessUploads()

	// Delete old/expired files with the processing results
	go deleteUploadResults()

	return nil
}

/*
 * Process '/upload' request to upload a new list of indicators
 */
func uploadHandler(w http.ResponseWriter, r *http.Request) {

	// Get client's IP
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		_, e := w.Write([]byte("Can't get IP to upload file: " + err.Error()))
		if e != nil {
			log.Error().Str("ip", ip).Msg("Can't send error: " + e.Error())
		}

		log.Error().Msg("Can't get IP to upload file: " + err.Error())
		return
	}

	// Check whether user is signed in
	username, err := sessions.exists(w, r)
	if err != nil {
		_, e := w.Write([]byte("Can't validate user. Check server logs for more info!"))
		if e != nil {
			log.Error().Str("ip", ip).Msg("Can't send error: " + e.Error())
		}

		log.Error().
			Str("ip", ip).
			Msg(err.Error())
		return
	}

	// Get account from the online list
	account, ok := online[username]
	if !ok {
		log.Error().
			Str("ip", ip).
			Str("username", username).
			Msg("Upload attempt from not signed in user")
		return
	}

	// Check whether file is too large
	if r.ContentLength > config.Upload.MaxSize {
		_, e := w.Write([]byte(fmt.Sprintf("File is too large, max %d Bytes are allowed", config.Upload.MaxSize)))
		if e != nil {
			log.Error().
				Str("ip", ip).
				Str("username", username).
				Msg("Can't send warning: " + e.Error())
		}

		log.Error().
			Str("ip", ip).
			Str("username", username).
			Msgf("Upload file is too large, max %d Bytes are allowed", config.Upload.MaxSize)
		return
	}

	file, header, err := r.FormFile("file") // FormFile function takes in the POST input file
	if err != nil {
		_, e := w.Write([]byte("Can't detect uploaded file: " + err.Error()))
		if e != nil {
			log.Error().
				Str("ip", ip).
				Str("username", username).
				Msg("Can't send error: " + e.Error())
		}

		log.Error().
			Str("ip", ip).
			Str("username", username).
			Msg("Can't detect uploaded file: " + err.Error())
		return
	}

	prefix := time.Now().Format("20060102-150405-07")

	// Received file's object can't be NULL
	if file == nil {
		_, e := w.Write([]byte("File was not selected!"))
		if e != nil {
			log.Error().
				Str("ip", ip).
				Str("username", username).
				Msg("Can't send warning: " + e.Error())
		}

		log.Error().
			Str("ip", ip).
			Str("username", username).
			Str("filename", header.Filename).
			Msg("Upload file was not selected!")
		return
	}
	defer file.Close()

	uploaded, err := os.Create(config.Upload.Path + "/queue/" + prefix + "-" + header.Filename)
	if err != nil {
		_, e := w.Write([]byte("Can't upload file: " + err.Error()))
		if e != nil {
			log.Error().
				Str("ip", ip).
				Str("username", username).
				Msg("Can't send error: " + e.Error())
		}

		log.Error().
			Str("ip", ip).
			Str("username", username).
			Str("filename", header.Filename).
			Msg("Can't create file for writing: " + err.Error())
		return
	}
	defer uploaded.Close()

	// Write the content from POST to the local file
	_, err = io.Copy(uploaded, file)
	if err != nil {
		_, e := w.Write([]byte("Can't upload file: " + err.Error()))
		if e != nil {
			log.Error().
				Str("ip", ip).
				Str("username", username).
				Msg("Can't send error: " + e.Error())
		}

		log.Error().
			Str("ip", ip).
			Str("username", username).
			Str("filename", header.Filename).
			Msg("Can't store uploaded file: " + err.Error())
		return
	}

	// Get dynamic attributes defined by the user
	upload := &Upload{
		Source:    r.FormValue("source"),
		StartTime: r.FormValue("startTime"),
		EndTime:   r.FormValue("endTime"),
		Format:    r.FormValue("format"),
		Field:     r.FormValue("field"),
	}

	// Add file to the user's queue
	account.Uploads.In[prefix+"-"+header.Filename] = upload
	err = account.update("uploads.in", account.Uploads.In)
	if err != nil {
		_, e := w.Write([]byte("Can't add uploaded file to the user's in-queue: " + err.Error()))
		if e != nil {
			log.Error().
				Str("ip", ip).
				Str("username", username).
				Msg("Can't send error: " + e.Error())
		}

		log.Error().
			Str("ip", ip).
			Str("username", username).
			Str("filename", header.Filename).
			Msg("Can't add uploaded file to the user's in-queue: " + err.Error())

		// Clean the environment in case of an error
		account.cleanUploads(prefix + "-" + header.Filename)
		return
	}

	// Notify user that upload was successful
	_, err = w.Write([]byte("ok"))
	if err != nil {
		log.Error().
			Str("ip", ip).
			Str("username", username).
			Str("filename", header.Filename).
			Msg("Can't inform that upload is successful: " + err.Error())
	}

	log.Info().
		Str("ip", ip).
		Str("username", username).
		Str("filename", header.Filename).
		Msg("File uploaded")

	// Notify that the uploaded file is in a queue.
	// TODO: Merge "ok" and "upload" responses in just one
	account.send("uploaded", prefix+"-"+header.Filename, "")

	// Start processing the uploaded file
	go func() {
		err = account.processUploads(prefix+"-"+header.Filename, upload)
		if err != nil {
			log.Error().
				Str("ip", ip).
				Str("username", username).
				Str("filename", header.Filename).
				Msg(err.Error())

			// Notify user about the error
			account.addNotification("error", err.Error())
		}

		// Clean the environment whether the error was nil or not
		account.cleanUploads(prefix + "-" + header.Filename)
	}()
}

/*
 * Start processing the uploaded file
 */
func (a *Account) processUploads(filename string, upload *Upload) error {

	// Generated report to send back to the user
	report := "Report for the uploaded file: " + filename[19:] + "\n"

	// Merged responses from different requests
	rRelations := ""
	rStats := ""
	rDebug := ""
	rError := ""

	// Validate user input
	if err := validUploads("source", upload.Source); err != nil {
		rError += "\n  - " + "Invalid 'source' value: " + err.Error()
	}
	if err := validUploads("datetime", upload.StartTime); err != nil {
		rError += "\n  - " + "Invalid 'start time' value: " + err.Error()
	}
	if err := validUploads("datetime", upload.EndTime); err != nil {
		rError += "\n  - " + "Invalid 'end time' value: " + err.Error()
	}
	if upload.Format != "json" && upload.Format != "table" {
		rError += "\n  - " + "Invalid 'output format' value: " + upload.Format + ", 'json' or 'table' extected"
	}

	// Start processing
	file, err := os.Open(config.Upload.Path + "/queue/" + filename)
	if err != nil {
		return fmt.Errorf("Can't open the uploaded file: " + err.Error())
	}

	field := strings.TrimSpace(upload.Field)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Some values shouldn't be quoted
		if field != "" {
			if valueIsNumeric(line) {
				line = field + "=" + line
			} else {
				line = field + "='" + line + "'"
			}
		}

		// Generate a resulting query
		sql := fmt.Sprintf("FROM %s WHERE (%s) AND datetime BETWEEN '%s' AND '%s'",
			upload.Source, strings.TrimSpace(line), upload.StartTime, upload.EndTime)

		// Query data sources for a new relations data
		response := querySources(upload.Source, sql, a.Options.ShowLimited, a.Options.Debug, a.Username)

		if len(response.Relations) != 0 {
			rRelations += "\n\nIndicator: " + line + "\n\n"

			// Format relations data
			str, err := formatTo(response.Relations, upload.Format)
			if err != nil {
				rError += "\n  - " + err.Error()
				break
			} else {
				rRelations += str
			}
		}

		if len(response.Stats) != 0 {
			rStats += "\n\nIndicator: " + line + "\n\n"

			// Format stats
			str, err := formatTo(response.Stats, upload.Format)
			if err != nil {
				rError += "\n  - " + err.Error()
				break
			} else {
				rStats += str
			}
		}

		if a.Options.Debug && response.Debug != nil {
			rDebug += "\n\nDebug info: " + line + "\n"

			for source, section := range response.Debug {
				if len(section.(map[string]interface{})) == 0 {
					continue
				}

				// Format debug info
				csv, err := formatTo(section, upload.Format)
				if err != nil {
					rError += "\n  - " + err.Error()
					break
				} else {
					rDebug += "\n" + source + "\n" + csv
				}
			}
		}

		if response.Error != "" {
			// Skip identical errors
			if !strings.Contains(rError, response.Error) {
				rError += "\n  - " + response.Error + ". Query: " + sql
			}
		}
	}

	err = scanner.Err()
	if err != nil {
		return fmt.Errorf("Can't scan the uploaded file: " + err.Error())
	}

	file.Close()

	// Fill report by the relations data
	if rRelations != "" {
		report += "\n" + rRelations
	}

	if rStats != "" {
		report += "\n\n\nThe amount of relations for some indicators exceeds the limit. Stats info based on limited info:"
		report += rStats
	}

	if rDebug != "" {
		report += "\n" + rDebug
	}

	if rError != "" {
		report += "\n\n\nSome errors have occurred during the process:\n"
		report += rError
	}

	// Write results to the file
	err = ioutil.WriteFile(config.Upload.Path+"/processed/"+filename, []byte(report), 0600)
	if err != nil {
		return fmt.Errorf("Can't save processed upload results: " + err.Error())
	}

	// Notify user that processing is completed
	a.send("upload-processed", filename, "")

	return nil
}

/*
 * Process uploaded files again if any process was interrupted,
 * for example service was stopped
 */
func reProcessUploads() {
	accounts, err := db.getAccounts()
	if err != nil {
		log.Error().Msg("Can't get all accounts to restart processing of the uploaded files: " + err.Error())
		return
	}

	for _, account := range accounts {
		if len(account.Uploads.In) > 0 {
			for filename, params := range account.Uploads.In {
				log.Info().
					Str("username", account.Username).
					Str("filename", filename[19:]).
					Msg("Restart processing of the uploaded file")

				// Start processing again
				err = account.processUploads(filename, params)
				if err != nil {
					log.Error().
						Str("username", account.Username).
						Str("filename", filename).
						Msg(err.Error())

					// Notify user about the error
					account.addNotification("error", err.Error())
				}

				// Clean the environment whether the error was nil or not
				account.cleanUploads(filename)
			}
		}
	}
}

/*
 * Validate dynamic parameters of the upload
 * TODO: call this function only once and return a single combined error
 */
func validUploads(field, value string) error {
	// Validate 'source' value
	if field == "source" {
		if value == "" {
			return fmt.Errorf("Can't be empty")
		} else if len(value) > 100 {
			return fmt.Errorf("Length is too long: %d characters", len(value))
		}
	}

	// Validate 'datetime' value
	if field == "datetime" {
		if value == "" {
			return fmt.Errorf("Can't be empty")
		} else if !reDatetime.MatchString(value) {
			return fmt.Errorf("Format is incorrect")
		}
	}

	return nil
}

/*
 * Clean the environment when the uploaded file is processed
 */
func (a *Account) cleanUploads(filename string) {
	err := os.Remove(config.Upload.Path + "/queue/" + filename)
	if err != nil {
		log.Error().
			Str("username", a.Username).
			Str("filename", filename[19:]).
			Msg("Can't remove the uploaded file: " + err.Error())
	} else {
		log.Info().
			Str("username", a.Username).
			Str("filename", filename[19:]).
			Msg("Uploads file deleted")
	}

	// Remove file from the user's in-queue
	delete(a.Uploads.In, filename)

	err = a.update("uploads.in", a.Uploads.In)
	if err != nil {
		log.Error().
			Str("username", a.Username).
			Str("filename", filename).
			Msg("Can't remove uploaded file from the in-queue: " + err.Error())
		return
	}

	log.Info().
		Str("username", a.Username).
		Str("filename", filename).
		Msg("Uploaded file removed from the in-queue")

	// Add processed file to the downloads list
	a.Uploads.Out = append(a.Uploads.Out, filename)

	err = a.update("uploads.out", a.Uploads.Out)
	if err != nil {
		log.Error().
			Str("username", a.Username).
			Str("filename", filename).
			Msg("Can't add processed file to the downloads list: " + err.Error())

		// Notify user about the error
		a.addNotification("error", "Can't add processed file <span class=\"ui orange text\">"+filename[19:]+"</span> to the downloads list: "+err.Error())
	}

	log.Info().
		Str("username", a.Username).
		Str("filename", filename).
		Msg("Processed file added to the downloads list")
}

/*
 * Process '/download' request to get a file with results
 * of the uploaded file processing
 */
func downloadHandler(w http.ResponseWriter, r *http.Request) {

	// Get client's IP
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		_, e := w.Write([]byte("Can't get IP to download processed file: " + err.Error()))
		if e != nil {
			log.Error().Msg("Can't send error: " + e.Error())
		}

		log.Error().
			Str("ip", ip).
			Msg("Can't get IP to download processed file: " + err.Error())
		return
	}

	// Check whether user is signed in
	username, err := sessions.exists(w, r)
	if err != nil {
		_, e := w.Write([]byte("Can't validate user. Check server logs for more info!"))
		if e != nil {
			log.Error().
				Str("ip", ip).
				Msg("Can't send error: " + e.Error())
		}

		log.Error().
			Str("ip", ip).
			Msg(err.Error())
		return
	}

	file := r.FormValue("file")

	// Validate file name
	if strings.Contains(file, "..") || strings.Contains(file, "/") {
		log.Error().
			Str("ip", ip).
			Str("username", username).
			Str("filename", file).
			Msg("Unexpected processed file requested to download")
		return
	}

	http.ServeFile(w, r, config.Upload.Path+"/processed/"+file)

	log.Info().
		Str("ip", ip).
		Str("username", username).
		Str("filename", file).
		Msg("Processed file downloaded")
}

/*
 * Delete old files with results of the uploaded files processing
 */
func deleteUploadResults() {
	for now := range time.Tick(time.Duration(config.Upload.DeleteInterval) * time.Second) {
		log.Debug().Msg("Cleaning users downloads list")

		accounts, err := db.getAccounts()
		if err != nil {
			log.Error().Msg("Can't get all accounts to delete the results of the uploaded files: " + err.Error())
			continue
		}

		// Check all the accounts first
		for _, account := range accounts {
			if len(account.Uploads.Out) > 0 {
				// Do not use 'range' to be able to change 'i'
				for i := 0; i < len(account.Uploads.Out); i++ {
					filename := account.Uploads.Out[i]

					dt, err := time.Parse("20060102-150405-07", filename[:18])
					if err != nil {
						log.Error().Msg("Can't parse datetime to delete the results of the uploaded file: " + err.Error())
						continue
					}

					if dt.Add(time.Duration(config.Upload.DeleteExpiration) * time.Second).Before(now) {

						err := os.Remove(config.Upload.Path + "/processed/" + filename)
						if err != nil {
							log.Error().
								Str("username", account.Username).
								Str("filename", filename).
								Msg("Can't delete the results of the uploaded file: " + err.Error())
						}

						// Remove file from the user's downloads list
						account.Uploads.Out[i] = account.Uploads.Out[len(account.Uploads.Out)-1]
						account.Uploads.Out = account.Uploads.Out[:len(account.Uploads.Out)-1]

						err = account.update("uploads.out", account.Uploads.Out)
						if err != nil {
							log.Error().
								Str("username", account.Username).
								Str("filename", filename).
								Msg("Can't remove file from the downloads list: " + err.Error())
							break
						}

						// Decrease counter to check the new file name again
						// after cleaning the slice
						i--

						log.Info().
							Str("username", account.Username).
							Str("filename", filename).
							Msg("Results of the uploaded file deleted")
					}
				}
			}
		}

		// Check the file system just in case some old files are stuck
		files, _ := ioutil.ReadDir(config.Upload.Path + "/processed/")
		for _, file := range files {
			if !file.IsDir() {
				filename := file.Name()

				dt, err := time.Parse("20060102-150405-07", filename[:18])
				if err != nil {
					log.Error().Msg("Can't parse datetime to delete unassigned results of the uploaded file: " + err.Error())
					continue
				}

				if dt.Add(time.Duration(config.Upload.DeleteExpiration) * time.Second).Before(now) {
					err := os.Remove(config.Upload.Path + "/processed/" + filename)
					if err != nil {
						log.Error().
							Str("filename", filename).
							Msg("Can't delete unassigned results of the uploaded file: " + err.Error())
					}

					log.Info().
						Str("filename", filename).
						Msg("Unassigned results of the uploaded file deleted")
				}
			}
		}
	}
}

/*
 * Send user's upload queue and download lists
 * to be displayed as Web GUI items
 */
func (a *Account) getUploadLists() {
	// Use a list instead of a concatenated string
	// to avoid possible splitting errors on a JavaScript side
	listIn := []string{}
	for k := range a.Uploads.In {
		listIn = append(listIn, k)
	}

	bIn, err := json.Marshal(listIn)
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't marshal upload queue list: " + err.Error())

		a.send("error", "Check server logs for more info!", "Can't get upload queue and download lists.")
		return
	}

	bOut, err := json.Marshal(a.Uploads.Out)
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't marshal download list: " + err.Error())

		a.send("error", "Check server logs for more info!", "Can't get upload queue and download lists.")
		return
	}

	a.send("upload-lists", string(bIn), string(bOut))
}

/*
 * Check whether string value is numeric
 */
func valueIsNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}
