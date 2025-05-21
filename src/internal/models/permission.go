package models

import (
	"time"

	"gorm.io/gorm"
)

type Permission struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	CardID *uint `json:"card_id,omitempty"`
	Card   Card  `json:"card,omitempty"`

	UserID *uint `json:"user_id,omitempty"`
	User   User  `json:"user,omitempty"`

	RoomID uint `gorm:"not null" json:"room_id"`
	Room   Room `json:"room,omitempty"`

	GrantedBy       uint       `json:"granted_by"`
	ValidFrom       time.Time  `gorm:"not null" json:"valid_from"`
	ValidUntil      *time.Time `json:"valid_until"`
	TimeRestriction string     `json:"time_restriction"`
	Active          bool       `gorm:"not null;default:true" json:"active"`
}

func (p *Permission) IsValid(currentTime time.Time) bool {
	if !p.Active {
		return false
	}

	if p.CardID == nil && p.UserID == nil {
		return false
	}

	if currentTime.Before(p.ValidFrom) {
		return false
	}

	if p.ValidUntil != nil && currentTime.After(*p.ValidUntil) {
		return false
	}

	if p.TimeRestriction != "" {
	}

	return true
}

func (p *Permission) Validate() error {
	if p.CardID == nil && p.UserID == nil {
		return gorm.ErrInvalidData
	}

	if p.CardID != nil && p.UserID != nil {
		return gorm.ErrInvalidData
	}

	return nil
}
