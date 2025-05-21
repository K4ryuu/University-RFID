package models

import (
	"time"

	"gorm.io/gorm"
)

type CardStatus string

const (
	CardStatusActive  CardStatus = "active"
	CardStatusBlocked CardStatus = "blocked"
	CardStatusRevoked CardStatus = "revoked"
	CardStatusExpired CardStatus = "expired"
	CardStatusPending CardStatus = "pending"
)

type Card struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	UserID uint `gorm:"not null" json:"user_id"`
	User   User `json:"user,omitempty"`

	CardID          string     `gorm:"uniqueIndex;not null" json:"card_id"`
	EncryptedCardID string     `gorm:"not null" json:"-"`
	Status          CardStatus `gorm:"not null;default:'active'" json:"status"`
	ExpiryDate      *time.Time `json:"expiry_date"`
	IssueDate       time.Time  `gorm:"not null" json:"issue_date"`
	LastUsed        *time.Time `json:"last_used"`

	Permissions []Permission `json:"permissions,omitempty"`
	Logs        []Log        `json:"logs,omitempty"`
}

func (c *Card) IsActive() bool {
	if c.Status != CardStatusActive {
		return false
	}

	if c.ExpiryDate != nil && time.Now().After(*c.ExpiryDate) {
		c.Status = CardStatusExpired
		return false
	}

	return true
}

func (c *Card) CanAccessRoom(roomID uint, currentTime time.Time) bool {
	if !c.IsActive() {
		return false
	}

	for _, permission := range c.Permissions {
		if permission.RoomID == roomID && permission.IsValid(currentTime) {
			return true
		}
	}

	return false
}
