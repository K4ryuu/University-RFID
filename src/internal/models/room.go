package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type AccessLevel string

const (
	AccessLevelPublic     AccessLevel = "public"
	AccessLevelRestricted AccessLevel = "restricted"
	AccessLevelAdmin      AccessLevel = "admin"
)

type Room struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name        string      `gorm:"not null" json:"name"`
	Description string      `json:"description"`
	Building    string      `gorm:"not null" json:"building"`
	RoomNumber  string      `gorm:"not null" json:"room_number"`
	AccessLevel AccessLevel `gorm:"not null;default:'restricted'" json:"access_level"`
	Capacity    int         `json:"capacity"`

	OperatingHours    string `json:"operating_hours"`
	OperatingDays     string `json:"operating_days"`
	SpecialConditions string `json:"special_conditions"`

	Permissions []Permission `json:"permissions,omitempty"`
	Logs        []Log        `json:"logs,omitempty"`
}

func (r *Room) IsAccessibleWithoutCard() bool {
	return r.AccessLevel == AccessLevelPublic
}

func (r *Room) IsAccessibleAtTime(t time.Time) bool {
	day := int(t.Weekday())
	dayStr := string('0' + rune(day))
	if r.OperatingDays != "" && !contains(r.OperatingDays, dayStr) {
		return false
	}

	if r.OperatingHours != "" {
		var openHour, openMin, closeHour, closeMin int
		_, err := fmt.Sscanf(r.OperatingHours, "%d:%d-%d:%d", &openHour, &openMin, &closeHour, &closeMin)
		if err == nil {
			openTime := time.Date(t.Year(), t.Month(), t.Day(), openHour, openMin, 0, 0, t.Location())
			closeTime := time.Date(t.Year(), t.Month(), t.Day(), closeHour, closeMin, 0, 0, t.Location())

			if t.Before(openTime) || t.After(closeTime) {
				return false
			}
		}
	}

	return true
}

func contains(s, substr string) bool {
	for i := 0; i < len(s); i++ {
		if string(s[i]) == substr {
			return true
		}
	}
	return false
}
