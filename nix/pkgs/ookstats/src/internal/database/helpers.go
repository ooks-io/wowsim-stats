package database

import (
	"strings"
	"time"
)

const busyRetryAttempts = 8

func isBusyError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "database is locked") || strings.Contains(msg, "busy")
}

func retryOnBusy(op func() error) error {
	var err error
	for attempt := 0; attempt < busyRetryAttempts; attempt++ {
		err = op()
		if err == nil || !isBusyError(err) {
			return err
		}
		backoff := time.Duration(attempt+1) * 100 * time.Millisecond
		if backoff > time.Second {
			backoff = time.Second
		}
		time.Sleep(backoff)
	}
	return err
}
