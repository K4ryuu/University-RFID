package models

import (
	"time"

	"gorm.io/gorm"
)

type Group struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	Name        string      `gorm:"not null" json:"name"`
	Description string      `json:"description"`
	ParentID    *uint       `json:"parent_id,omitempty"`
	Parent      *Group      `json:"parent,omitempty"`
	AccessLevel AccessLevel `gorm:"not null;default:'restricted'" json:"access_level"`

	Users []User `gorm:"many2many:user_groups;" json:"users,omitempty"`
	Rooms []Room `gorm:"many2many:group_rooms;" json:"rooms,omitempty"`
}
