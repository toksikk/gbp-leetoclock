package main

import (
	"fmt"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/sirupsen/logrus"
	"github.com/toksikk/gbp-leetoclock/pkg/datastore"
	"github.com/toksikk/gbp-leetoclock/pkg/helper"
)

var PluginName = "leetoclock"
var PluginVersion = ""
var PluginBuilddate = ""

var tt time.Time

var tHourInt, tMinuteInt = 13, 37

var session *discordgo.Session

var preparationAnnounceLock = false
var winnerAnnounceLock = false
var renewReactionsLock = false

var playersWithClockReactions []string = []string{}

const firstPlace string = "🥇"
const secondPlace string = "🥈"
const thirdPlace string = "🥉"
const otherPlace string = "🏅"
const zonk string = ":zonk:750630908372975636"
const lol string = ":louisdefunes_lol:357611625102180373"
const notamused string = ":louisdefunes_notamused:357611625521479680"
const wat string = ":gustaff:721122751145967679"

var announcementChannels = []string{}

var store *datastore.Store

func Start(discord *discordgo.Session) {
	session = discord
	store = datastore.NewStore(datastore.InitDB())
	session.AddHandler(onMessageCreate)

	if os.Getenv("LEETOCLOCK_DEBUG") != "" {
		t := time.Now()
		target := t.Add(time.Minute * 1)
		tHourInt, tMinuteInt = target.Hour(), target.Minute()
		logrus.Debugln("Updated target time to", tHourInt, tMinuteInt)
	}
	if os.Getenv("LEETOCLOCK_DEBUG_CHANNEL") != "" {
		announcementChannels = append(announcementChannels, os.Getenv("LEETOCLOCK_DEBUG_CHANNEL"))
	}
	go gameTick()
}

func calculateScore(messageTimestamp time.Time) int {
	return int(messageTimestamp.Sub(tt).Milliseconds())
}

func isOnTargetTimeRange(messageTimestamp time.Time, onlyOnTarget bool) bool {
	oneMinuteBefore := tt.Add(-time.Minute * 1)
	if messageTimestamp.Hour() == tt.Hour() && messageTimestamp.Minute() == tt.Minute() {
		return true
	}
	if !onlyOnTarget {
		if messageTimestamp.Hour() == oneMinuteBefore.Hour() && messageTimestamp.Minute() == oneMinuteBefore.Minute() {
			return true
		}
	}
	return false
}

func announcePreparation() {
	if isOnTargetTimeRange(time.Now(), false) {
		preparationAnnounceLock = true
		for _, v := range announcementChannels {
			_, err := session.ChannelMessageSend(v, fmt.Sprintf("## Leet o'Clock scheduled:\n<t:%d:R>", tt.Unix()))
			if err != nil {
				logrus.Errorln(err)
			}
		}
		time.Sleep(2 * time.Minute)
		preparationAnnounceLock = false
	}
}

func isScoreInScoreArray(s datastore.Score, a []datastore.Score) bool {
	for _, v := range a {
		if v.PlayerID == s.PlayerID {
			return true
		}
	}
	return false
}

func buildScoreboardForGame(game datastore.Game) (string, []datastore.Score, []datastore.Score, []datastore.Score, error) {
	scores, err := store.GetScoresForGameID(game.ID)
	if err != nil {
		return "", []datastore.Score{}, []datastore.Score{}, []datastore.Score{}, err
	}
	channel, _ := session.Channel(game.ChannelID)

	scoreboard := fmt.Sprintf("## 1337erboard for <t:%d>\n", tt.Unix())

	earlyBirds := make([]datastore.Score, 0)
	winners := make([]datastore.Score, 0)
	zonks := make([]datastore.Score, 0)

	printHeader := true
	for _, score := range scores {
		player, err := store.GetPlayerByID(score.PlayerID)
		if err != nil {
			return "", []datastore.Score{}, []datastore.Score{}, []datastore.Score{}, err
		}

		if isScoreInScoreArray(score, winners) || len(winners) >= 3 {
			continue
		} else {
			if score.Score >= 0 {
				if printHeader {
					scoreboard += "### Top scorers\n"
					printHeader = false
				}
				winners = append(winners, score)
				var award string
				if len(winners) == 1 {
					award = firstPlace
				} else if len(winners) == 2 {
					award = secondPlace
				} else if len(winners) == 3 {
					award = thirdPlace
				} else {
					award = otherPlace
				}

				scoreboard += fmt.Sprintf("%s <@%s> with %d ms (https://discord.com/channels/%s/%s/%s)\n", award, player.UserID, score.Score, channel.GuildID, game.ChannelID, score.MessageID)
			}
		}
	}

	printHeader = true
	for _, score := range scores {
		player, err := store.GetPlayerByID(score.PlayerID)
		if err != nil {
			return "", []datastore.Score{}, []datastore.Score{}, []datastore.Score{}, err
		}

		if isScoreInScoreArray(score, zonks) || isScoreInScoreArray(score, winners) {
			continue
		} else {
			if score.Score > 0 {
				if printHeader {
					scoreboard += "### Zonks\n"
					printHeader = false
				}
				zonks = append(zonks, score)

				scoreboard += fmt.Sprintf("%s <@%s> with %d ms\n (https://discord.com/channels/%s/%s/%s)", "😭", player.UserID, score.Score, channel.GuildID, game.ChannelID, score.MessageID)
			}
		}
	}

	printHeader = true
	for _, score := range scores {
		player, err := store.GetPlayerByID(score.PlayerID)
		if err != nil {
			return "", []datastore.Score{}, []datastore.Score{}, []datastore.Score{}, err
		}

		if isScoreInScoreArray(score, earlyBirds) {
			continue
		} else {
			if score.Score >= -5000 && score.Score < 0 {
				if printHeader {
					scoreboard += "### Honorlolable mentions\n"
					printHeader = false
				}
				earlyBirds = append(earlyBirds, score)
				var award string
				if isScoreInScoreArray(score, zonks) {
					award = "🫠"
				} else if isScoreInScoreArray(score, winners) {
					award = "😐"
				} else {
					award = "🤨"
				}

				scoreboard += fmt.Sprintf("%s <@%s> with %d ms\n (https://discord.com/channels/%s/%s/%s)", award, player.UserID, score.Score, channel.GuildID, game.ChannelID, score.MessageID)
			}
		}
	}

	// TODO: this "find highest score for current season" should be a function in datastore
	var memScore datastore.Score = datastore.Score{Score: 999999999999999999}
	season, err := store.GetSeasonByDate(time.Now())
	if err != nil {
		logrus.Errorln(err)
	}
	games, err := store.GetGames()
	if err != nil {
		logrus.Errorln(err)
	}
	for _, g := range games {
		if g.SeasonID == season.ID {
			scores, err := store.GetScoresForGameID(g.ID)
			if err != nil {
				logrus.Errorln(err)
			}
			for _, s := range scores {
				if s.Score >= 0 && s.Score < memScore.Score {
					memScore = s
				}
			}
		}
	}
	player, err := store.GetPlayerByID(memScore.PlayerID)
	if err != nil {
		logrus.Errorln(err)
	}

	scoreboard += fmt.Sprintf("### Current season highscore\n<@%s> with %d ms on <t:%d> (https://discord.com/channels/%s/%s/%s)\n", player.UserID, memScore.Score, memScore.CreatedAt.Unix(), channel.GuildID, game.ChannelID, memScore.MessageID)
	scoreboard += fmt.Sprintf("\nCurrent season ends on <t:%d> (<t:%d:R>)\n", season.EndDate.Unix(), season.EndDate.Unix())

	return scoreboard, earlyBirds, winners, zonks, nil
}

func renewReactions(game datastore.Game) {
	for renewReactionsLock {
		time.Sleep(1 * time.Second)
	}

	renewReactionsLock = true

	_, earlybirds, winners, zonks, err := buildScoreboardForGame(game)
	if err != nil {
		logrus.Errorln(err)
		return
	}

	for _, v := range earlybirds {
		_ = session.MessageReactionRemove(game.ChannelID, v.MessageID, lol, session.State.User.ID)
		_ = session.MessageReactionRemove(game.ChannelID, v.MessageID, notamused, session.State.User.ID)
		_ = session.MessageReactionRemove(game.ChannelID, v.MessageID, wat, session.State.User.ID)
		if isScoreInScoreArray(v, zonks) {
			_ = session.MessageReactionAdd(game.ChannelID, v.MessageID, lol)
		} else if isScoreInScoreArray(v, winners) {
			_ = session.MessageReactionAdd(game.ChannelID, v.MessageID, notamused)
		} else {
			_ = session.MessageReactionAdd(game.ChannelID, v.MessageID, wat)
		}
	}

	for i, v := range winners {
		_ = session.MessageReactionRemove(game.ChannelID, v.MessageID, firstPlace, session.State.User.ID)
		_ = session.MessageReactionRemove(game.ChannelID, v.MessageID, secondPlace, session.State.User.ID)
		_ = session.MessageReactionRemove(game.ChannelID, v.MessageID, thirdPlace, session.State.User.ID)
		if i == 0 {
			_ = session.MessageReactionAdd(game.ChannelID, v.MessageID, firstPlace)
		} else if i == 1 {
			_ = session.MessageReactionAdd(game.ChannelID, v.MessageID, secondPlace)
		} else if i == 2 {
			_ = session.MessageReactionAdd(game.ChannelID, v.MessageID, thirdPlace)
		}
	}

	for _, v := range zonks {
		_ = session.MessageReactionRemove(game.ChannelID, v.MessageID, zonk, session.State.User.ID)
		_ = session.MessageReactionAdd(game.ChannelID, v.MessageID, zonk)
	}

	renewReactionsLock = false
}

func announceTodaysWinners() {
	if isOnTargetTimeRange(time.Now(), true) {
		winnerAnnounceLock = true
		time.Sleep(62 * time.Second)
		games, err := store.GetGamesByDate(time.Now())
		if err != nil {
			logrus.Errorln(err)
			return
		}
		for _, game := range games {
			scoreboard, _, _, _, err := buildScoreboardForGame(game)
			if err != nil {
				logrus.Errorln(err)
				return
			}
			_, err = session.ChannelMessageSend(game.ChannelID, scoreboard)
			if err != nil {
				logrus.Errorln(err)
			}
		}
	}
	winnerAnnounceLock = false
	resetGameVars()
}

func resetGameVars() {
	playersWithClockReactions = []string{}
}

func gameTick() {
	for {
		if isOnTargetTimeRange(time.Now(), false) {
			time.Sleep(1 * time.Second)
		} else {
			if os.Getenv("LEETOCLOCK_DEBUG") != "" {
				time.Sleep(1 * time.Second)
			} else {
				time.Sleep(1 * time.Minute)
			}
			updateTTHelper()
		}
		if !preparationAnnounceLock {
			go announcePreparation()
		}
		if !winnerAnnounceLock {
			go announceTodaysWinners()
		}
	}
}

func updateTTHelper() {
	t := time.Now()
	tt = time.Date(t.Year(), t.Month(), t.Day(), tHourInt, tMinuteInt, 0, 0, t.Location())
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	message := m.Message
	messageTimestamp := helper.GetTimestampOfMessage(message.ID)

	if isOnTargetTimeRange(messageTimestamp, false) {
		season, err := store.EnsureSeason(time.Now())
		if err != nil {
			logrus.Errorln(err)
		}
		game, err := store.EnsureGame(message.ChannelID, tt, season.ID)
		if err != nil {
			logrus.Errorln(err)
		}
		player, err := store.EnsurePlayer(message.Author.ID)
		if err != nil {
			logrus.Errorln(err)
		}
		err = store.CreateScore(message.ID, player.ID, calculateScore(messageTimestamp), game.ID)
		if err != nil {
			logrus.Errorln(err)
		}

		if isOnTargetTimeRange(messageTimestamp, true) {
			hasPlayerClockReaction := func() bool {
				for _, v := range playersWithClockReactions {
					if v == message.Author.ID {
						return true
					}
				}
				return false
			}
			if !hasPlayerClockReaction() {
				_ = s.MessageReactionAdd(m.ChannelID, m.ID, "⏰")
				playersWithClockReactions = append(playersWithClockReactions, message.Author.ID)
			}
		}
		go renewReactions(*game)
	}
}
