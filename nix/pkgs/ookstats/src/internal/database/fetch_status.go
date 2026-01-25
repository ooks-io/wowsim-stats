package database

import (
	"errors"
	"fmt"
	"net/http"
	"ookstats/internal/blizzard"
	"strconv"
	"time"
)

const (
	fetchStatusOK      = "ok"
	fetchStatusMissing = "missing"
	fetchStatusError   = "error"
)

// RecordFetchStatus records the status of an API fetch attempt
func (ds *DatabaseService) RecordFetchStatus(region, realmSlug string, dungeonID, periodID int, status string, httpStatus int, message string) error {
	if periodID == 0 {
		return nil
	}

	if len(message) > 512 {
		message = message[:512]
	}

	err := retryOnBusy(func() error {
		_, execErr := ds.db.Exec(`
			INSERT INTO fetch_status (region, realm_slug, dungeon_id, period_id, status, http_status, checked_at, message)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(region, realm_slug, dungeon_id, period_id)
			DO UPDATE SET
				status = excluded.status,
				http_status = excluded.http_status,
				checked_at = excluded.checked_at,
				message = excluded.message
		`, region, realmSlug, dungeonID, periodID, status, httpStatus, time.Now().Unix(), message)
		return execErr
	})
	return err
}

func (ds *DatabaseService) recordFetchStatusResult(res blizzard.FetchResult) {
	periodID := parsePeriodID(res)
	if periodID == 0 {
		return
	}

	status := fetchStatusOK
	httpStatus := http.StatusOK
	message := ""

	if res.Error != nil {
		status = fetchStatusError
		message = truncateStatusMessage(res.Error.Error())

		var apiErr *blizzard.APIError
		if errors.As(res.Error, &apiErr) {
			if apiErr.Status != 0 {
				httpStatus = apiErr.Status
			}
			if apiErr.Status == http.StatusNotFound {
				status = fetchStatusMissing
			}
		} else {
			httpStatus = 0
		}
	} else if res.Leaderboard == nil || len(res.Leaderboard.LeadingGroups) == 0 {
		message = "no runs returned"
	}

	if err := ds.RecordFetchStatus(res.RealmInfo.Region, res.RealmInfo.Slug, res.Dungeon.ID, periodID, status, httpStatus, message); err != nil {
		fmt.Printf("[WARN] failed to record fetch status for %s/%s period %d: %v\n",
			res.RealmInfo.Slug, res.Dungeon.Slug, periodID, err)
	}
}

func parsePeriodID(res blizzard.FetchResult) int {
	if res.PeriodID != "" {
		if pid, err := strconv.Atoi(res.PeriodID); err == nil {
			return pid
		}
	}
	if res.Leaderboard != nil {
		return res.Leaderboard.Period
	}
	return 0
}

func truncateStatusMessage(msg string) string {
	if len(msg) <= 512 {
		return msg
	}
	return msg[:512]
}
