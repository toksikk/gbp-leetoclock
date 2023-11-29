package datastore

import (
	"errors"
	"log/slog"
	"os"
	"sync"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Player struct {
	gorm.Model
	UserID string `gorm:"not null;unique"`
}

type Season struct {
	gorm.Model
	StartDate time.Time `gorm:"not null"`
	EndDate   time.Time `gorm:"not null"`
}

type Game struct {
	gorm.Model
	ChannelID string    `gorm:"not null"`
	GuildID   string    `gorm:"not null"`
	GameDate  time.Time `gorm:"not null"`
	SeasonID  uint      `gorm:"not null"`
	Season    Season    `gorm:"foreignKey:SeasonID"`
}

type Score struct {
	gorm.Model
	GameID    uint   `gorm:"not null"`
	MessageID string `gorm:"not null;unique"`
	PlayerID  uint   `gorm:"not null"`
	Score     int    `gorm:"not null"`
	Game      Game   `gorm:"foreignKey:GameID"`
	Player    Player `gorm:"foreignKey:PlayerID"`
}

type Highscore struct {
	gorm.Model
	PlayerID uint   `gorm:"not null"`
	ScoreID  uint   `gorm:"not null;unique"`
	SeasonID uint   `gorm:"not null"`
	Player   Player `gorm:"foreignKey:PlayerID"`
	Score    Score  `gorm:"foreignKey:ScoreID"`
	Season   Season `gorm:"foreignKey:SeasonID"`
}

type Store struct {
	db *gorm.DB
	mu sync.Mutex
}

func InitDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("plugins/leetoclock.sqlite"), &gorm.Config{})

	if err != nil {
		slog.Error("failed to connect database", "Error", err)
		os.Exit(1)
	}

	err = db.AutoMigrate(&Player{}, &Season{}, &Game{}, &Score{}, &Highscore{})
	if err != nil {
		slog.Error("failed to migrate database", "Error", err)
		os.Exit(1)
	}

	return db
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// HELPER

func getSeasonStartDateForDate(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
}

func getSeasonEndDateForDate(date time.Time) time.Time {
	var lastDayOfMonth int
	switch date.Month() {
	case time.January, time.March, time.May, time.July, time.August, time.October, time.December:
		lastDayOfMonth = 31
	case time.April, time.June, time.September, time.November:
		lastDayOfMonth = 30
	case time.February:
		if date.Year()%4 == 0 {
			lastDayOfMonth = 29
		} else {
			lastDayOfMonth = 28
		}
	}

	return time.Date(date.Year(), date.Month(), lastDayOfMonth, 23, 59, 59, 999999999, date.Location())
}

// SEASON

func (s *Store) EnsureSeason(date time.Time) (*Season, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var season Season = Season{StartDate: getSeasonStartDateForDate(date), EndDate: getSeasonEndDateForDate(date)}
	result := s.db.Where("start_date <= ? AND end_date >= ?", date, date).First(&season)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			result := s.db.Create(&season)
			if result.Error != nil {
				return nil, result.Error
			}
		} else {
			return nil, result.Error
		}
	}
	return &season, nil
}

func (s *Store) GetSeasons() ([]Season, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var seasons []Season
	result := s.db.Find(&seasons)
	if result.Error != nil {
		return nil, result.Error
	}
	return seasons, nil
}

func (s *Store) GetSeasonByID(id uint) (*Season, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var season Season
	result := s.db.Where("id = ?", id).First(&season)
	if result.Error != nil {
		return nil, result.Error
	}
	return &season, nil
}

func (s *Store) GetSeasonByDate(date time.Time) (*Season, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var season Season
	result := s.db.Where("start_date <= ? AND end_date >= ?", date, date).First(&season)
	if result.Error != nil {
		return nil, result.Error
	}
	return &season, nil
}

// PLAYER

func (s *Store) CreatePlayer(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var player Player = Player{UserID: userID}
	result := s.db.Create(&player)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *Store) EnsurePlayer(userID string) (*Player, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var player Player = Player{UserID: userID}
	result := s.db.Where("user_id = ?", userID).First(&player)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			result := s.db.Create(&player)
			if result.Error != nil {
				return nil, result.Error
			}
		} else {
			return nil, result.Error
		}
	}
	return &player, nil
}

func (s *Store) GetPlayers() ([]Player, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var players []Player
	result := s.db.Find(&players)
	if result.Error != nil {
		return nil, result.Error
	}
	return players, nil
}

func (s *Store) GetPlayerByID(id uint) (*Player, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var player Player
	result := s.db.Where("id = ?", id).First(&player)
	if result.Error != nil {
		return nil, result.Error
	}
	return &player, nil
}

func (s *Store) GetPlayerByUserID(userID string) (*Player, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var player Player
	result := s.db.Where("user_id = ?", userID).First(&player)
	if result.Error != nil {
		return nil, result.Error
	}
	return &player, nil
}

// GAME

func (s *Store) CreateGame(channelID string, guildID string, gameDate time.Time, seasonID uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var game Game = Game{ChannelID: channelID, GuildID: guildID, GameDate: gameDate, SeasonID: seasonID}
	result := s.db.FirstOrCreate(&game)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *Store) EnsureGame(channelID string, guildID string, gameDate time.Time, seasonID uint) (*Game, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var game Game = Game{ChannelID: channelID, GuildID: guildID, GameDate: gameDate, SeasonID: seasonID}
	result := s.db.Where("channel_id = ? AND guild_id = ? AND game_date = ? AND season_id = ?", channelID, guildID, gameDate, seasonID).First(&game)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			result := s.db.Create(&game)
			if result.Error != nil {
				return nil, result.Error
			}
		} else {
			return nil, result.Error
		}
	}
	return &game, nil
}

func (s *Store) GetGames() ([]Game, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var games []Game
	result := s.db.Find(&games)
	if result.Error != nil {
		return nil, result.Error
	}
	return games, nil
}

func (s *Store) GetGameByID(id uint) (*Game, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var game Game
	result := s.db.Where("id = ?", id).First(&game)
	if result.Error != nil {
		return nil, result.Error
	}
	return &game, nil
}

func (s *Store) GetGamesByChannelID(channelID string) ([]Game, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var games []Game
	result := s.db.Where("channel_id = ?", channelID).Find(&games)
	if result.Error != nil {
		return nil, result.Error
	}
	return games, nil
}

func (s *Store) GetGamesByGuildID(guildID string) ([]Game, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var games []Game
	result := s.db.Where("guild_id = ?", guildID).Find(&games)
	if result.Error != nil {
		return nil, result.Error
	}
	return games, nil
}

func (s *Store) GetGamesByDate(date time.Time) ([]Game, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var games []Game
	startDate := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endDate := time.Date(date.Year(), date.Month(), date.Day(), 23, 59, 59, 0, date.Location())
	result := s.db.Where("game_date >= ? AND game_date <= ?", startDate, endDate).Find(&games)
	if result.Error != nil {
		return nil, result.Error
	}
	return games, nil
}

func (s *Store) GetGameBySpecificDateTimeAndChannelID(gameDate time.Time, channelID string) (*Game, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var game Game
	result := s.db.Where("game_date = ? AND channel_id = ?", gameDate, channelID).First(&game)
	if result.Error != nil {
		slog.Error("AN ERROR OCCURED", "Error", result.Error)
		games, _ := s.GetGames()
		for _, g := range games {
			slog.Info("GAME", "Game", g)
		}
		return nil, result.Error
	}
	return &game, nil
}

// SCORE

func (s *Store) CreateScore(messageID string, playerID uint, score int, gameID uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var scoreObj Score = Score{MessageID: messageID, PlayerID: playerID, Score: score, GameID: gameID}
	result := s.db.Create(&scoreObj)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *Store) GetScores() ([]Score, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var scores []Score
	result := s.db.Find(&scores)
	if result.Error != nil {
		return nil, result.Error
	}
	return scores, nil
}

func (s *Store) GetScoreByID(id uint) (*Score, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var score Score
	result := s.db.Where("id = ?", id).First(&score)
	if result.Error != nil {
		return nil, result.Error
	}
	return &score, nil
}

func (s *Store) GetScoresForGameID(gameID uint) ([]Score, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var scores []Score
	result := s.db.Where("game_id = ?", gameID).Find(&scores)
	if result.Error != nil {
		return nil, result.Error
	}
	return scores, nil
}

// HIGHSCORE

func (s *Store) CreateHighscore(playerID uint, scoreID uint, seasonID uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var highscore Highscore = Highscore{PlayerID: playerID, ScoreID: scoreID, SeasonID: seasonID}
	result := s.db.Create(&highscore)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *Store) GetHighscores() ([]Highscore, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var highscores []Highscore
	result := s.db.Find(&highscores)
	if result.Error != nil {
		return nil, result.Error
	}
	return highscores, nil
}

func (s *Store) GetHighscoreByID(id uint) (*Highscore, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var highscore Highscore
	result := s.db.Where("id = ?", id).First(&highscore)
	if result.Error != nil {
		return nil, result.Error
	}
	return &highscore, nil
}

func (s *Store) GetHighscoreByPlayerIDAndSeasonID(playerID uint, seasonID uint) (*Highscore, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var highscore Highscore
	result := s.db.Where("player_id = ? AND season_id = ?", playerID, seasonID).First(&highscore)
	if result.Error != nil {
		return nil, result.Error
	}
	return &highscore, nil
}
