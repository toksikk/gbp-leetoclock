package helper

import (
	"log/slog"
	"strconv"
	"time"
)

func idToTimestamp(id string) (int64, error) {
	convertedID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return -1, err
	}
	convertedIDString := strconv.FormatInt(convertedID, 2)
	m := 64 - len(convertedIDString)
	unixbin := convertedIDString[0 : 42-m]
	unix, err := strconv.ParseInt(unixbin, 2, 64)
	if err != nil {
		return -1, err
	}
	return unix + 1420070400000, nil
}

func GetTimestampOfMessage(messageID string) time.Time {
	timestamp, err := idToTimestamp(messageID)
	if err != nil {
		slog.Error("Error while converting messageID to timestamp", "Error", err)
		return time.Time{}
	}
	return time.UnixMilli(timestamp)
}
