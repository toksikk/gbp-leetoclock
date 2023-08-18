package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
	"github.com/toksikk/gbp-leetoclock/pkg/datastore"
	"github.com/toksikk/gbp-leetoclock/pkg/helper"
)

var PluginName = "leetoclock"
var PluginVersion = ""
var PluginBuilddate = ""

var targetChannel string

var tt time.Time
var btt time.Time
var att time.Time

var tHour, tMinute = "13", "37"

var participatingMessages []*discordgo.Message
var session *discordgo.Session
var awards [3]string = [3]string{"🥇", "🥈", "🥉"}

var store *datastore.Store

func Start(discord *discordgo.Session) {
	session = discord
	store = datastore.NewStore(datastore.InitDB())
	participatingMessages = make([]*discordgo.Message, 0)
	session.AddHandler(onMessageCreate)
	setTargetTime()
	setTargetChannel()
	go leaderboardResetLoop()
	go winnerAnnounceLoop()
}

func getCurrentTimeForDebugging() (string, string) {
	t := time.Now()
	hour := strconv.Itoa(t.Hour())
	minute := strconv.Itoa(t.Minute() + 1)
	return hour, minute
}

func setTargetChannel() {
	if os.Getenv("LEETOCLOCK_DEBUG_CHANNEL") != "" {
		targetChannel = os.Getenv("LEETOCLOCK_DEBUG_CHANNEL")
		t := time.Now()
		tt = time.Date(t.Year(), t.Month(), t.Day(), tt.Hour(), tt.Minute(), 0, 0, t.Location())
		session.ChannelMessageSend(targetChannel, fmt.Sprintf("Target time: <t:%d:R>", tt.Unix()))
	} else {
		targetChannel = "225303764108705793"
	}
}

func setTargetTime() {
	if os.Getenv("LEETOCLOCK_DEBUG") != "" {
		tHour, tMinute = getCurrentTimeForDebugging()
	}
	ttString := fmt.Sprintf("2006-01-02T%s:%s:00Z", tHour, tMinute)
	tt, _ = time.Parse(time.RFC3339, ttString)
	btt = tt.Add(-time.Minute * 1)
	att = tt.Add(time.Minute * 1)
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
				timestamps = append(timestamps, helper.GetTimestamp(v.ID).UnixMilli())
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
					if helper.GetTimestamp(p.ID).UnixMilli() == v && containsMessageOfUser(&winningMessages, *p.Author) == -1 {
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
				s := fmt.Sprintf("%s <@%s> with %dms.", awards[k], v.Author.ID, helper.GetTimestamp(v.ID).Sub(t).Milliseconds())
				player, err := store.GetPlayerByUserID(v.Author.ID)
				if err != nil {
					logrus.Errorln(err)
				}
				game, err := store.GetGameByChannelID(targetChannel)
				if err != nil {
					logrus.Errorln(err)
				}
				err = store.CreateScore(v.ID, player.ID, int(helper.GetTimestamp(v.ID).Sub(t).Milliseconds()), game.ID)
				if err != nil {
					logrus.Errorln(err)
				}
				_, err = session.ChannelMessageSend(targetChannel, s)
				if err != nil {
					logrus.Errorln(err)
					break
				}
			}
			for _, v := range zonkMessages {
				s := fmt.Sprintf("🏅 <@%s> with %dms.", v.Author.ID, helper.GetTimestamp(v.ID).Sub(t).Milliseconds())
				player, err := store.GetPlayerByUserID(v.Author.ID)
				if err != nil {
					logrus.Errorln(err)
				}
				game, err := store.GetGameByChannelID(targetChannel)
				if err != nil {
					logrus.Errorln(err)
				}
				err = store.CreateScore(v.ID, player.ID, int(helper.GetTimestamp(v.ID).Sub(t).Milliseconds()), game.ID)
				if err != nil {
					logrus.Errorln(err)
				}
				_, err = session.ChannelMessageSend(targetChannel, s)
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

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	tm := helper.GetTimestamp(m.ID)
	if tm.Hour() == tt.Hour() && tm.Minute() == tt.Minute() && m.Author.ID != s.State.User.ID {
		season, err := store.EnsureSeason(time.Now())
		if err != nil {
			logrus.Errorln(err)
			season.ID = 0
		}
		err = store.CreateGame(m.ChannelID, time.Now(), season.ID)
		if err != nil {
			logrus.Errorln(err)
		}
		err = store.CreatePlayer(m.Author.ID)
		if err != nil {
			logrus.Errorln(err)
		}
		s.MessageReactionAdd(m.ChannelID, m.ID, "⏰")
		if m.ChannelID == targetChannel {
			newMessages := make([]*discordgo.Message, 0)

			i := containsMessageOfUser(&participatingMessages, *m.Message.Author)

			for k, v := range participatingMessages {
				if k != i {
					newMessages = append(newMessages, v)
				} else {
					if helper.GetTimestamp(m.Message.ID).UnixMilli() < helper.GetTimestamp(v.ID).UnixMilli() {
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
