package main

import (
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

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	logrus.Infoln("leetoclock onMessageCreate handler called")
}
