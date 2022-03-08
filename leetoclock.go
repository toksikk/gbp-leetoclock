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

func Start(discord *discordgo.Session) {
	logrus.Infoln("loaded leetoclock plugin")
	discord.AddHandler(onMessageCreate)
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

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	timestamp, _ := idToTimestamp(m.ID)
	tm := time.Unix(timestamp/1000, 0)
	logrus.Infof("leetoclock: message has unix timestamp %s which equals %s\n", timestamp, tm)
}
