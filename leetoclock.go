package main

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
)

var PluginName = "leetoclock"
var PluginVersion = ""
var PluginBuilddate = ""

var targetChannel string = "225303764108705793"

var tt time.Time
var btt time.Time
var att time.Time

const tHour string = "13"
const tMinute string = "37"

var participantsList []*discordgo.Message
var session *discordgo.Session
var awards [3]string = [3]string{"🥇", "🥈", "🥉"}

func Start(discord *discordgo.Session) {
	setTargetTime()
	discord.AddHandler(onMessageCreate)
	participantsList = make([]*discordgo.Message, 0)
	session = discord
	go leaderboardResetLoop()
	go winnerAnnounceLoop()
}

func setTargetTime() {
	ttString := fmt.Sprintf("2006-01-02T%s:%s:00Z", tHour, tMinute)
	tt, _ = time.Parse(time.RFC3339, ttString)
	btt = tt.Add(-time.Minute * 1)
	att = tt.Add(time.Minute * 1)
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
		if time.Now().Hour() == btt.Hour() && time.Now().Minute() == btt.Minute() {
			participantsList = make([]*discordgo.Message, 0)
		}
		time.Sleep(60 * time.Second)
	}
}

func isAwarded(awardedMessages *[]*discordgo.Message, user discordgo.User) bool {
	for _, v := range *awardedMessages {
		if v.Author.ID == user.ID {
			return true
		}
	}
	return false
}

func participatingAuthorsAmount(messages []*discordgo.Message) int {
	authors := make([]string, 0)
	for _, v := range messages {
		exists := false
		for _, a := range authors {
			if a == v.Author.ID {
				exists = true
			}
		}
		if !exists {
			authors = append(authors, v.Author.ID)
		}
	}
	return len(authors)
}

func winnerAnnounceLoop() {
	sleepDelay := 60
	winningMessages := make([]*discordgo.Message, 0)
	zonkMessages := make([]*discordgo.Message, 0)
	awardCounter := 0
	previousTimestampsAmount := 0
	for {
		currentTime := time.Now()
		if currentTime.Hour() == btt.Hour() && currentTime.Minute() == btt.Minute() {
			sleepDelay = 1
		}
		if currentTime.Hour() == tt.Hour() && currentTime.Minute() == tt.Minute() {

			timestamps := make([]int64, 0)
			for _, v := range participantsList {
				timestamps = append(timestamps, getTimestamp(v.ID).UnixMilli())
			}
			sort.Slice(timestamps, func(i, j int) bool {
				return timestamps[i] < timestamps[j]
			})

			if len(timestamps) > previousTimestampsAmount {
				for _, v := range winningMessages {
					for _, a := range awards {
						session.MessageReactionRemove(v.ChannelID, v.ID, a, session.State.User.ID)
					}
				}
				for _, v := range zonkMessages {
					session.MessageReactionRemove(v.ChannelID, v.ID, ":zonk:750630908372975636", session.State.User.ID)
				}

				winningMessages = make([]*discordgo.Message, 0)
				zonkMessages = make([]*discordgo.Message, 0)
				awardCounter = 0
			}

			previousTimestampsAmount = len(timestamps)

			for _, v := range timestamps {
				for _, p := range participantsList {
					if getTimestamp(p.ID).UnixMilli() == v && !isAwarded(&winningMessages, *p.Author) {
						if awardCounter < 3 {
							session.MessageReactionAdd(p.ChannelID, p.ID, awards[awardCounter])
							awardCounter++
							winningMessages = append(winningMessages, p)
						} else {
							if !isAwarded(&zonkMessages, *p.Author) {
								session.MessageReactionAdd(p.ChannelID, p.ID, ":zonk:750630908372975636")
								zonkMessages = append(zonkMessages, p)
							}
						}
					}
				}
			}
		}
		if currentTime.Hour() == att.Hour() && currentTime.Minute() == att.Minute() {
			awardCounter = 0
			sleepDelay = 60

			t := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), tt.Hour(), tt.Minute(), 0, 0, currentTime.Location())
			for k, v := range winningMessages {
				if k == 0 {
					session.ChannelMessageSend(targetChannel, "Today's 1337erboard:")
				}
				s := fmt.Sprintf("%s <@%s> with %dms.", awards[k], v.Author.ID, getTimestamp(v.ID).Sub(t).Milliseconds())
				_, err := session.ChannelMessageSend(targetChannel, s)
				if err != nil {
					logrus.Errorln(err)
					break
				}
			}
			for _, v := range zonkMessages {
				s := fmt.Sprintf("🏅 <@%s> with %dms.", v.Author.ID, getTimestamp(v.ID).Sub(t).Milliseconds())
				_, err := session.ChannelMessageSend(targetChannel, s)
				if err != nil {
					logrus.Errorln(err)
					break
				}
			}

			winningMessages = make([]*discordgo.Message, 0)
			zonkMessages = make([]*discordgo.Message, 0)
		}
		time.Sleep(time.Duration(sleepDelay) * time.Second)
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
	if tm.Hour() == tt.Hour() && tm.Minute() == tt.Minute() &&  m.Author.ID != s.State.User.ID {
		s.MessageReactionAdd(m.ChannelID, m.ID, "⏰")
		if m.ChannelID == targetChannel {
			participantsList = append(participantsList, m.Message)
		}
	}
}
