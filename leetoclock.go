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

type targetTime struct {
	hour   string
	minute string
}

func (tt targetTime) getHourAsInt() int {
	r, _ := strconv.Atoi(tt.hour)
	return r
}

func (tt targetTime) getMinuteAsInt() int {
	r, _ := strconv.Atoi(tt.minute)
	return r
}

var tt targetTime = targetTime{
	hour:   "13",
	minute: "37",
}

var participantsList []*discordgo.Message
var session *discordgo.Session
var awards [3]string = [3]string{"🥇", "🥈", "🥉"}

func Start(discord *discordgo.Session) {
	discord.AddHandler(onMessageCreate)
	participantsList = make([]*discordgo.Message, 0)
	session = discord
	go leaderboardResetLoop()
	go winnerAnnounceLoop()
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
		if time.Now().Hour() == tt.getHourAsInt() && time.Now().Minute() == tt.getMinuteAsInt()-1 {
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

func winnerAnnounceLoop() {
	sleepDelay := 60
	winningMessages := make([]*discordgo.Message, 0)
	awardCounter := 0
	for {
		if time.Now().Hour() == tt.getHourAsInt() && time.Now().Minute() == tt.getMinuteAsInt()-1 {
			sleepDelay = 1
		}
		if time.Now().Hour() == tt.getHourAsInt() && time.Now().Minute() == tt.getMinuteAsInt() {
			timestamps := make([]int64, 0)
			for _, v := range participantsList {
				timestamps = append(timestamps, getTimestamp(v.ID).UnixMilli())
			}
			sort.Slice(timestamps, func(i, j int) bool {
				return timestamps[i] < timestamps[j]
			})

			for _, v := range timestamps {
				for _, p := range participantsList {
					if getTimestamp(p.ID).UnixMilli() == v && !isAwarded(&winningMessages, *p.Author) {
						if awardCounter < 3 {
							session.MessageReactionAdd(p.ChannelID, p.ID, awards[awardCounter])
							awardCounter++
							winningMessages = append(winningMessages, p)
						} else {
							session.MessageReactionAdd(p.ChannelID, p.ID, ":zonk:750630908372975636")
						}
					}
				}
			}
		}
		if time.Now().Hour() == tt.getHourAsInt() && time.Now().Minute() == tt.getMinuteAsInt()+1 {
			awardCounter = 0
			sleepDelay = 60

			for k, v := range winningMessages {
				if k == 0 {
					session.ChannelMessageSend(targetChannel, "Today's 1337erboard:")
				}
				t := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), tt.getHourAsInt(), tt.getMinuteAsInt(), 0, 0, time.Now().Location())
				s := fmt.Sprintf("%s <@%s> with %dms.", awards[k], v.Author.ID, getTimestamp(v.ID).Sub(t).Milliseconds())
				_, err := session.ChannelMessageSend(targetChannel, s)
				if err != nil {
					logrus.Errorln(err)
					break
				}
			}

			winningMessages = make([]*discordgo.Message, 0)
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
	if tm.Hour() == tt.getHourAsInt() && tm.Minute() == tt.getMinuteAsInt() {
		s.MessageReactionAdd(m.ChannelID, m.ID, "⏰")
		if m.ChannelID == targetChannel {
			participantsList = append(participantsList, m.Message)
		}
	}
}
