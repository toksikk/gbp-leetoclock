package datastore

import (
	"time"

	"github.com/sirupsen/logrus"
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
}

func InitDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open("plugins/leetoclock.sqlite"), &gorm.Config{})

	if err != nil {
		logrus.Fatalf("failed to connect database: %v", err)
	}

	err = db.AutoMigrate(&Player{}, &Season{}, &Game{}, &Score{}, &Highscore{})
	if err != nil {
		logrus.Fatalf("failed to migrate database: %v", err)
	}

	return db
}

func NewStore(db *gorm.DB) *Store {
	return &Store{db: db}
}

// HELPER

func zeroTime(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
}

func getSeasonStartDateForDate(date time.Time) time.Time {
	var startMonth time.Month
	switch date.Month() {
	case time.January, time.February, time.March:
		startMonth = time.January
	case time.April, time.May, time.June:
		startMonth = time.April
	case time.July, time.August, time.September:
		startMonth = time.July
	case time.October, time.November, time.December:
		startMonth = time.October
	}
	return time.Date(date.Year(), startMonth, 1, 0, 0, 0, 0, date.Location())
}

func getSeasonEndDateForDate(date time.Time) time.Time {
	var endMonth time.Month
	switch date.Month() {
	case time.January, time.February, time.March:
		endMonth = time.March
	case time.April, time.May, time.June:
		endMonth = time.June
	case time.July, time.August, time.September:
		endMonth = time.September
	case time.October, time.November, time.December:
		endMonth = time.December
	}

	var lastDayOfMonth int
	switch endMonth {
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

	return time.Date(date.Year(), endMonth, lastDayOfMonth, 0, 0, 0, 0, date.Location())
}

// SEASON

func (s *Store) EnsureSeason(date time.Time) (*Season, error) {
	zeroedDate := zeroTime(date)
	var season Season = Season{StartDate: getSeasonStartDateForDate(zeroedDate), EndDate: getSeasonEndDateForDate(zeroedDate)}
	result := s.db.Where("start_date <= ? AND end_date >= ?", zeroedDate, zeroedDate).First(&season)
	if result.Error != nil {
		if result.Error.Error() == "record not found" {
			// Create season
			result := s.db.FirstOrCreate(&season, Season{StartDate: getSeasonStartDateForDate(zeroedDate), EndDate: getSeasonEndDateForDate(zeroedDate)})
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
	var seasons []Season
	result := s.db.Find(&seasons)
	if result.Error != nil {
		return nil, result.Error
	}
	return seasons, nil
}

func (s *Store) GetSeasonByID(id uint) (*Season, error) {
	var season Season
	result := s.db.Where("id = ?", id).First(&season)
	if result.Error != nil {
		return nil, result.Error
	}
	return &season, nil
}

func (s *Store) GetSeasonByDate(date time.Time) (*Season, error) {
	zeroedDate := zeroTime(date)
	var season Season
	result := s.db.Where("start_date <= ? AND end_date >= ?", zeroedDate, zeroedDate).First(&season)
	if result.Error != nil {
		return nil, result.Error
	}
	return &season, nil
}

// PLAYER

func (s *Store) CreatePlayer(userID string) error {
	var player Player = Player{UserID: userID}
	result := s.db.FirstOrCreate(&player, Player{UserID: userID})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *Store) GetPlayers() ([]Player, error) {
	var players []Player
	result := s.db.Find(&players)
	if result.Error != nil {
		return nil, result.Error
	}
	return players, nil
}

func (s *Store) GetPlayerByID(id uint) (*Player, error) {
	var player Player
	result := s.db.Where("id = ?", id).First(&player)
	if result.Error != nil {
		return nil, result.Error
	}
	return &player, nil
}

func (s *Store) GetPlayerByUserID(userID string) (*Player, error) {
	var player Player
	result := s.db.Where("user_id = ?", userID).First(&player)
	if result.Error != nil {
		return nil, result.Error
	}
	return &player, nil
}

// GAME

func (s *Store) CreateGame(channelID string, gameDate time.Time, seasonID uint) error {
	zeroedGameDate := zeroTime(gameDate)
	var game Game = Game{ChannelID: channelID, GameDate: zeroedGameDate, SeasonID: seasonID}
	result := s.db.FirstOrCreate(&game, Game{ChannelID: channelID, GameDate: zeroedGameDate, SeasonID: seasonID})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *Store) GetGames() ([]Game, error) {
	var games []Game
	result := s.db.Find(&games)
	if result.Error != nil {
		return nil, result.Error
	}
	return games, nil
}

func (s *Store) GetGameByID(id uint) (*Game, error) {
	var game Game
	result := s.db.Where("id = ?", id).First(&game)
	if result.Error != nil {
		return nil, result.Error
	}
	return &game, nil
}

func (s *Store) GetGameByChannelID(channelID string) (*Game, error) {
	var game Game
	result := s.db.Where("channel_id = ?", channelID).First(&game)
	if result.Error != nil {
		return nil, result.Error
	}
	return &game, nil
}

func (s *Store) GetGameByDate(date time.Time) (*Game, error) {
	zeroedDate := zeroTime(date)
	var game Game
	result := s.db.Where("game_date = ?", zeroedDate).First(&game)
	if result.Error != nil {
		return nil, result.Error
	}
	return &game, nil
}

// SCORE

func (s *Store) CreateScore(messageID string, playerID uint, score int, gameID uint) error {
	var scoreObj Score = Score{MessageID: messageID, PlayerID: playerID, Score: score, GameID: gameID}
	result := s.db.FirstOrCreate(&scoreObj, Score{PlayerID: playerID, GameID: gameID})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *Store) GetScores() ([]Score, error) {
	var scores []Score
	result := s.db.Find(&scores)
	if result.Error != nil {
		return nil, result.Error
	}
	return scores, nil
}

func (s *Store) GetScoreByID(id uint) (*Score, error) {
	var score Score
	result := s.db.Where("id = ?", id).First(&score)
	if result.Error != nil {
		return nil, result.Error
	}
	return &score, nil
}

// HIGHSCORE

func (s *Store) CreateHighscore(playerID uint, scoreID uint, seasonID uint) error {
	var highscore Highscore = Highscore{PlayerID: playerID, ScoreID: scoreID, SeasonID: seasonID}
	result := s.db.FirstOrCreate(&highscore, Highscore{PlayerID: playerID, ScoreID: scoreID, SeasonID: seasonID})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (s *Store) GetHighscores() ([]Highscore, error) {
	var highscores []Highscore
	result := s.db.Find(&highscores)
	if result.Error != nil {
		return nil, result.Error
	}
	return highscores, nil
}

func (s *Store) GetHighscoreByID(id uint) (*Highscore, error) {
	var highscore Highscore
	result := s.db.Where("id = ?", id).First(&highscore)
	if result.Error != nil {
		return nil, result.Error
	}
	return &highscore, nil
}

func (s *Store) GetHighscoreByPlayerIDAndSeasonID(playerID uint, seasonID uint) (*Highscore, error) {
	var highscore Highscore
	result := s.db.Where("player_id = ? AND season_id = ?", playerID, seasonID).First(&highscore)
	if result.Error != nil {
		return nil, result.Error
	}
	return &highscore, nil
}
