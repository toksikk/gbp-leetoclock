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

var participatingMessages []*discordgo.Message
var session *discordgo.Session
var awards [3]string = [3]string{"🥇", "🥈", "🥉"}

func Start(discord *discordgo.Session) {
	setTargetTime()
	discord.AddHandler(onMessageCreate)
	participatingMessages = make([]*discordgo.Message, 0)
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
			participatingMessages = make([]*discordgo.Message, 0)
		}
		time.Sleep(60 * time.Second)
	}
}

func containsMessageOfUser(messages *[]*discordgo.Message, user discordgo.User) int {
	for k, v := range *messages {
		if v.Author.ID == user.ID {
			return k
		}
	}
	return -1
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
			for _, v := range participatingMessages {
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
				for _, p := range participatingMessages {
					if getTimestamp(p.ID).UnixMilli() == v && containsMessageOfUser(&winningMessages, *p.Author) == -1 {
						if awardCounter < 3 {
							session.MessageReactionAdd(p.ChannelID, p.ID, awards[awardCounter])
							awardCounter++
							winningMessages = append(winningMessages, p)
						} else {
							if containsMessageOfUser(&zonkMessages, *p.Author) == -1 {
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
	if tm.Hour() == tt.Hour() && tm.Minute() == tt.Minute() && m.Author.ID != s.State.User.ID {
		s.MessageReactionAdd(m.ChannelID, m.ID, "⏰")
		if m.ChannelID == targetChannel {
			newMessages := make([]*discordgo.Message, 0)

			i := containsMessageOfUser(&participatingMessages, *m.Message.Author)

			for k, v := range participatingMessages {
				if k != i {
					newMessages = append(newMessages, v)
				} else {
					if getTimestamp(m.Message.ID).UnixMilli() < getTimestamp(v.ID).UnixMilli() {
						newMessages = append(newMessages, m.Message)
					} else {
						newMessages = append(newMessages, v)
					}
				}
			}

			if len(participatingMessages) == 0 || i == -1 {
				newMessages = append(newMessages, m.Message)
			}

			participatingMessages = make([]*discordgo.Message, 0)
			for _, v := range newMessages {
				participatingMessages = append(participatingMessages, v)
			}
		}
	}
}
