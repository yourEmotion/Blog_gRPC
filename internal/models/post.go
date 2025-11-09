package models

import (
	"time"
)

type Post struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	AuthorID  uint64
	Body      string
	CreatedAt time.Time
}
