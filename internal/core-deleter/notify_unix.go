//go:build linux || darwin
// +build linux darwin
package core_deleter

import (
    "github.com/martinlindhe/notify"
)

func SendNotification(appID, title, message string) error {
    notify.Notify(appID, title, message, "")
    return nil
}
