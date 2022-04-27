package main

import (
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

var PluginName = "leetoclock"
var PluginVersion = ""
var PluginBuilddate = ""

var leaderboardCounter = 0
var awards [3]string = [3]string{"🥇", "🥈", "🥉"}

func Start(discord *discordgo.Session) {
	discord.AddHandler(onMessageCreate)
	go leaderboardResetLoop()
}

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

func leaderboardResetLoop() {
	for {
		if time.Now().Hour() == 13 && time.Now().Minute() == 36 {
			leaderboardCounter = 0
		}
		time.Sleep(60 * time.Second)
	}
}

func getTimestamp(messageID string) time.Time {
	timestamp, err := idToTimestamp(messageID)
	if err != nil {
		logrus.Errorln(err)
		return time.Time{}
	}
	return time.UnixMilli(timestamp)
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	tm := getTimestamp(m.ID)
	if tm.Hour() == 13 && tm.Minute() == 37 {
		s.MessageReactionAdd(m.ChannelID, m.ID, "⏰")
		if leaderboardCounter <= 2 {
			s.MessageReactionAdd(m.ChannelID, m.ID, awards[leaderboardCounter])
			leaderboardCounter++
		}
	}
}
