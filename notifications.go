package main

import (
	"time"
)

/*
 * Structure of one Web UI notification
 */
type Notification struct {
	// Type of notification.
	// Currently possible values: "err", "info"
	Type string `bson:"type" json:"type"`

	// Message text
	Message string `bson:"message" json:"message"`

	// Timestamp of the creation time
	Ts string `bson:"ts" json:"ts"`
}

/*
 * Add new notification to the user.
 *
 * If user is online - notification will be received in a browser,
 * If offline - it will be stored in a database.
 *
 * Receives its type and message text
 */
func (a *Account) addNotification(typ, message string) {
	n := &Notification{
		Type:    typ,
		Message: message,
		Ts:      time.Now().Format("2 Jan 2006 15:04:05"),
	}

	// Check whether user is online first
	if a.Session != nil {
		a.send("notification", message, typ)
		return
	}

	// Store the notification in a database otherwise
	a.Notifications = append(a.Notifications, n)

	// Leave the last 5 notifications only
	if len(a.Notifications) > 5 {
		a.Notifications = a.Notifications[len(a.Notifications)-5:]
	}

	err := a.update("notifications", a.Notifications)
	if err != nil {
		log.Error().
			Str("username", a.Username).
			Msg("Can't add a notification to the user: " + err.Error())
	}

	log.Info().
		Str("ip", a.Session.IP).
		Str("username", a.Username).
		Msg("Notification added")
}

/*
 * Handle 'notifications' websocket command
 * to clean user's notifications
 */
func (a *Account) notificationsHandler() {

	a.Notifications = []*Notification{}

	err := a.update("notifications", a.Notifications)
	if err != nil {
		log.Error().
			Str("ip", a.Session.IP).
			Str("username", a.Username).
			Msg("Can't clean notifications: " + err.Error())

		a.send("error", err.Error(), "Can't clean notifications!")
		return
	}

	log.Info().
		Str("ip", a.Session.IP).
		Str("username", a.Username).
		Msg("Notifications cleaned")
}
