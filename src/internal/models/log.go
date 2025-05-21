package models

import (
	"time"

	"gorm.io/gorm"
)

type AccessResult string

const (
	AccessGranted AccessResult = "granted"
	AccessDenied  AccessResult = "denied"
)

type DenialReason string

const (
	DenialReasonNoPermission    DenialReason = "no_permission"
	DenialReasonCardInactive    DenialReason = "card_inactive"
	DenialReasonCardExpired     DenialReason = "card_expired"
	DenialReasonOutsideHours    DenialReason = "outside_hours"
	DenialReasonRoomClosed      DenialReason = "room_closed"
	DenialReasonCardBlocked     DenialReason = "card_blocked"
	DenialReasonCardRevoked     DenialReason = "card_revoked"
	DenialReasonPermissionError DenialReason = "permission_error"
)

type Log struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	CardID uint `gorm:"not null" json:"card_id"`
	Card   Card `json:"card,omitempty"`

	RoomID uint `gorm:"not null" json:"room_id"`
	Room   Room `json:"room,omitempty"`

	Timestamp    time.Time    `gorm:"not null" json:"timestamp"`
	AccessResult AccessResult `gorm:"not null" json:"access_result"`
	DenialReason DenialReason `json:"denial_reason,omitempty"`
	Description  string       `json:"description,omitempty"`
	IPAddress    string       `json:"ip_address,omitempty"`
	DeviceID     string       `json:"device_id,omitempty"`
}
