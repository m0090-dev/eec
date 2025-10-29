//go:build linux || darwin
// +build linux darwin
package main

import (
    "github.com/martinlindhe/notify"
)

func SendNotification(appID, title, message string) error {
    notify.Notify(appID, title, message, "")
    return nil
}
