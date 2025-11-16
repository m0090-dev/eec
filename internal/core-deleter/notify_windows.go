//go:build windows
// +build windows 
package core_deleter

import (
    "github.com/go-toast/toast"
)

func SendNotification(appID, title, message string) error {
    notification := toast.Notification{
        AppID:   appID,
        Title:   title,
        Message: message,
        Icon:    "",
    }
    return notification.Push()
}
