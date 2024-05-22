package persistence

import (
	"gorm.io/driver/sqlite" // Sqlite driver based on CGO
	// "github.com/glebarez/sqlite" // Pure go SQLite driver, checkout https://github.com/glebarez/sqlite for details
	"gorm.io/gorm"
)

type Sentence struct {
	Id      int64  `gorm:"primaryKey;autoIncrement"`
	UserId  string `gorm:"index"`
	Role    string `gorm:"index"`
	Content string
	gorm.Model
}

var db *gorm.DB

func InitDB() error {
	var err error
	// github.com/mattn/go-sqlite3
	db, err = gorm.Open(sqlite.Open("chat.db"), &gorm.Config{})
	if err != nil {
		return err
	}

	return db.AutoMigrate(&Sentence{})
}

func InsertSentence(userId, content, role string) error {
	return db.Create(&Sentence{
		UserId:  userId,
		Role:    role,
		Content: content,
	}).Error
}

func GetSentences(userId string, count int) ([]*Sentence, error) {
	sentences := make([]*Sentence, 0, count)

	err := db.Where("user_id = ?", userId).Order("id desc").Limit(count).Find(&sentences).Error
	return sentences, err
}
