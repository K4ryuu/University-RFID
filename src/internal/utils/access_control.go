package utils

import (
	"time"

	"gorm.io/gorm"

	"rfid/internal/models"
	"rfid/internal/websocket"
)

type AccessControlService struct {
	db         *gorm.DB
	wsHandler  *websocket.WebSocketHandler
	wsEnabled  bool
}

// NewAccessControlService creates a new instance of AccessControlService
func NewAccessControlService(db *gorm.DB) *AccessControlService {
	return &AccessControlService{
		db:         db,
		wsEnabled:  false,
	}
}

// SetWebSocketHandler sets the WebSocket handler for real-time notifications
func (acs *AccessControlService) SetWebSocketHandler(wsHandler *websocket.WebSocketHandler) {
	acs.wsHandler = wsHandler
	acs.wsEnabled = (wsHandler != nil)
}

// CheckAccess determines if a card has access to a room
func (acs *AccessControlService) CheckAccess(cardID string, roomID uint, deviceID string) (bool, models.DenialReason, error) {
	var card models.Card
	if err := acs.db.Preload("Permissions").Preload("User").First(&card, "card_id = ?", cardID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, models.DenialReasonNoPermission, nil
		}
		return false, "", err
	}

	// Check if card is active
	cardWasActive := card.Status == models.CardStatusActive

	if !card.IsActive() {
		// Ha a kártya lejárt, mentsük el a változtatást az adatbázisban
		if cardWasActive && card.Status == models.CardStatusExpired {
			acs.db.Save(&card)
		}

		var denialReason models.DenialReason
		if card.Status == models.CardStatusBlocked {
			denialReason = models.DenialReasonCardBlocked
		} else if card.Status == models.CardStatusRevoked {
			denialReason = models.DenialReasonCardRevoked
		} else if card.Status == models.CardStatusExpired {
			denialReason = models.DenialReasonCardExpired
		} else {
			denialReason = models.DenialReasonCardInactive
		}

		// Log failed access with WebSocket notification
		acs.LogAccess(card.ID, roomID, models.AccessDenied, denialReason, deviceID)
		return false, denialReason, nil
	}

	var room models.Room
	if err := acs.db.First(&room, roomID).Error; err != nil {
		return false, models.DenialReasonPermissionError, err
	}

	currentTime := time.Now()

	// Check if room is open
	if !room.IsAccessibleAtTime(currentTime) {
		// Log failed access with WebSocket notification
		acs.LogAccess(card.ID, roomID, models.AccessDenied, models.DenialReasonOutsideHours, deviceID)
		return false, models.DenialReasonOutsideHours, nil
	}

	// Check if room is publicly accessible (no card needed)
	if room.IsAccessibleWithoutCard() {
		// Update last used time
		card.LastUsed = &currentTime
		acs.db.Save(&card)

		// Log successful access with WebSocket notification
		acs.LogAccess(card.ID, roomID, models.AccessGranted, "", deviceID)

		return true, "", nil
	}

	// Check permissions through the card
	if card.CanAccessRoom(roomID, currentTime) {
		// Update last used time
		card.LastUsed = &currentTime
		acs.db.Save(&card)

		// Log successful access with WebSocket notification
		acs.LogAccess(card.ID, roomID, models.AccessGranted, "", deviceID)

		return true, "", nil
	}

	// If the card doesn't have direct permission, check if the user has direct permission
	// This is only applicable if the card is associated with a user
	if card.UserID != 0 {
		// Check for direct user-to-room permissions
		var directPermissions []models.Permission
		if err := acs.db.Where("user_id = ? AND room_id = ? AND active = ?", card.UserID, roomID, true).Find(&directPermissions).Error; err != nil {
			return false, models.DenialReasonPermissionError, err
		}

		// Check if there's at least one valid permission
		for _, perm := range directPermissions {
			if perm.IsValid(currentTime) {
				// Update last used time
				card.LastUsed = &currentTime
				acs.db.Save(&card)

				// Log successful access with WebSocket notification
				acs.LogAccess(card.ID, roomID, models.AccessGranted, "", deviceID)

				return true, "", nil
			}
		}
	}

	// Check if the user belongs to any groups that have access to the room
	if card.UserID != 0 {
		// Query for group membership and group-room relationships
		var groupsCount int64
		err := acs.db.Raw(`
			SELECT COUNT(g.id)
			FROM groups g
			JOIN user_groups ug ON g.id = ug.group_id
			JOIN group_rooms gr ON g.id = gr.group_id
			WHERE ug.user_id = ? AND gr.room_id = ? AND g.deleted_at IS NULL
		`, card.UserID, roomID).Count(&groupsCount).Error

		if err != nil {
			return false, models.DenialReasonPermissionError, err
		}

		if groupsCount > 0 {
			// User belongs to at least one group that has access to the room
			// Update last used time
			card.LastUsed = &currentTime
			acs.db.Save(&card)

			// Log successful access with WebSocket notification
			acs.LogAccess(card.ID, roomID, models.AccessGranted, "", deviceID)

			return true, "", nil
		}
	}

	// Log failed access with WebSocket notification
	acs.LogAccess(card.ID, roomID, models.AccessDenied, models.DenialReasonNoPermission, deviceID)

	return false, models.DenialReasonNoPermission, nil
}

// LogAccess records an access attempt in the database
func (acs *AccessControlService) LogAccess(cardID uint, roomID uint, result models.AccessResult, denialReason models.DenialReason, deviceID string) {
	log := models.Log{
		CardID:       cardID,
		RoomID:       roomID,
		Timestamp:    time.Now(),
		AccessResult: result,
		DenialReason: denialReason,
		DeviceID:     deviceID,
	}

	acs.db.Create(&log)

	// Send WebSocket notification if WebSocket is enabled
	if acs.wsEnabled {
		acs.wsHandler.NotifyAccessEvent(cardID, roomID, string(result), string(denialReason))
	}
}

// GrantAccess gives a card permission to access a room
func (acs *AccessControlService) GrantAccess(cardID uint, roomID uint, grantedBy uint, validUntil *time.Time, timeRestriction string) error {
	cardIDPtr := &cardID
	permission := models.Permission{
		CardID:          cardIDPtr,
		RoomID:          roomID,
		GrantedBy:       grantedBy,
		ValidFrom:       time.Now(),
		ValidUntil:      validUntil,
		TimeRestriction: timeRestriction,
		Active:          true,
	}

	return acs.db.Create(&permission).Error
}

// GrantDirectAccess gives a user direct permission to access a room (without card)
func (acs *AccessControlService) GrantDirectAccess(userID uint, roomID uint, grantedBy uint, validUntil *time.Time, timeRestriction string) error {
	userIDPtr := &userID
	permission := models.Permission{
		UserID:          userIDPtr,
		RoomID:          roomID,
		GrantedBy:       grantedBy,
		ValidFrom:       time.Now(),
		ValidUntil:      validUntil,
		TimeRestriction: timeRestriction,
		Active:          true,
	}

	return acs.db.Create(&permission).Error
}

// RevokeAccess removes a card's permission to access a room
func (acs *AccessControlService) RevokeAccess(cardID uint, roomID uint) error {
	return acs.db.Model(&models.Permission{}).
		Where("card_id = ? AND room_id = ?", cardID, roomID).
		Update("active", false).
		Error
}

// RevokeDirectAccess removes a user's direct permission to access a room
func (acs *AccessControlService) RevokeDirectAccess(userID uint, roomID uint) error {
	return acs.db.Model(&models.Permission{}).
		Where("user_id = ? AND room_id = ?", userID, roomID).
		Update("active", false).
		Error
}

// RegisterCard creates a new card in the system
func (acs *AccessControlService) RegisterCard(userID uint, cardID string) (models.Card, error) {
	// Encrypt the card ID
	encryptedCardID, err := EncryptCardID(cardID)
	if err != nil {
		return models.Card{}, err
	}

	card := models.Card{
		UserID:          userID,
		CardID:          cardID,
		EncryptedCardID: encryptedCardID,
		Status:          models.CardStatusActive,
		IssueDate:       time.Now(),
	}

	if err := acs.db.Create(&card).Error; err != nil {
		return models.Card{}, err
	}

	return card, nil
}

// BlockCard temporarily blocks a card
func (acs *AccessControlService) BlockCard(cardID uint) error {
	return acs.db.Model(&models.Card{}).
		Where("id = ?", cardID).
		Update("status", models.CardStatusBlocked).
		Error
}

// UnblockCard reactivates a blocked card
func (acs *AccessControlService) UnblockCard(cardID uint) error {
	return acs.db.Model(&models.Card{}).
		Where("id = ? AND status = ?", cardID, models.CardStatusBlocked).
		Update("status", models.CardStatusActive).
		Error
}

// RevokeCard permanently revokes a card
func (acs *AccessControlService) RevokeCard(cardID uint) error {
	return acs.db.Model(&models.Card{}).
		Where("id = ?", cardID).
		Update("status", models.CardStatusRevoked).
		Error
}