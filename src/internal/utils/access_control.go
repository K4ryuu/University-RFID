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

func NewAccessControlService(db *gorm.DB) *AccessControlService {
	return &AccessControlService{
		db:         db,
		wsEnabled:  false,
	}
}

func (acs *AccessControlService) SetWebSocketHandler(wsHandler *websocket.WebSocketHandler) {
	acs.wsHandler = wsHandler
	acs.wsEnabled = (wsHandler != nil)
}

func (acs *AccessControlService) CheckAccess(cardID string, roomID uint, deviceID string) (bool, models.DenialReason, error) {
	var card models.Card
	if err := acs.db.Preload("Permissions").Preload("User").First(&card, "card_id = ?", cardID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, models.DenialReasonNoPermission, nil
		}
		return false, "", err
	}

	cardWasActive := card.Status == models.CardStatusActive

	if !card.IsActive() {
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

		acs.LogAccess(card.ID, roomID, models.AccessDenied, denialReason, deviceID)
		return false, denialReason, nil
	}

	var room models.Room
	if err := acs.db.First(&room, roomID).Error; err != nil {
		return false, models.DenialReasonPermissionError, err
	}

	currentTime := time.Now()

	if !room.IsAccessibleAtTime(currentTime) {
		acs.LogAccess(card.ID, roomID, models.AccessDenied, models.DenialReasonOutsideHours, deviceID)
		return false, models.DenialReasonOutsideHours, nil
	}

	if room.IsAccessibleWithoutCard() {
		card.LastUsed = &currentTime
		acs.db.Save(&card)

		acs.LogAccess(card.ID, roomID, models.AccessGranted, "", deviceID)

		return true, "", nil
	}

	if card.CanAccessRoom(roomID, currentTime) {
		card.LastUsed = &currentTime
		acs.db.Save(&card)

		acs.LogAccess(card.ID, roomID, models.AccessGranted, "", deviceID)

		return true, "", nil
	}

	if card.UserID != 0 {
		var directPermissions []models.Permission
		if err := acs.db.Where("user_id = ? AND room_id = ? AND active = ?", card.UserID, roomID, true).Find(&directPermissions).Error; err != nil {
			return false, models.DenialReasonPermissionError, err
		}

		for _, perm := range directPermissions {
			if perm.IsValid(currentTime) {
				card.LastUsed = &currentTime
				acs.db.Save(&card)

				acs.LogAccess(card.ID, roomID, models.AccessGranted, "", deviceID)

				return true, "", nil
			}
		}
	}

	if card.UserID != 0 {
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
			card.LastUsed = &currentTime
			acs.db.Save(&card)

			acs.LogAccess(card.ID, roomID, models.AccessGranted, "", deviceID)

			return true, "", nil
		}
	}

	acs.LogAccess(card.ID, roomID, models.AccessDenied, models.DenialReasonNoPermission, deviceID)

	return false, models.DenialReasonNoPermission, nil
}

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

	if acs.wsEnabled {
		acs.wsHandler.NotifyAccessEvent(cardID, roomID, string(result), string(denialReason))
	}
}

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

func (acs *AccessControlService) RevokeAccess(cardID uint, roomID uint) error {
	return acs.db.Model(&models.Permission{}).
		Where("card_id = ? AND room_id = ?", cardID, roomID).
		Update("active", false).
		Error
}

func (acs *AccessControlService) RevokeDirectAccess(userID uint, roomID uint) error {
	return acs.db.Model(&models.Permission{}).
		Where("user_id = ? AND room_id = ?", userID, roomID).
		Update("active", false).
		Error
}

func (acs *AccessControlService) RegisterCard(userID uint, cardID string) (models.Card, error) {
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

func (acs *AccessControlService) BlockCard(cardID uint) error {
	return acs.db.Model(&models.Card{}).
		Where("id = ?", cardID).
		Update("status", models.CardStatusBlocked).
		Error
}

func (acs *AccessControlService) UnblockCard(cardID uint) error {
	return acs.db.Model(&models.Card{}).
		Where("id = ? AND status = ?", cardID, models.CardStatusBlocked).
		Update("status", models.CardStatusActive).
		Error
}

func (acs *AccessControlService) RevokeCard(cardID uint) error {
	return acs.db.Model(&models.Card{}).
		Where("id = ?", cardID).
		Update("status", models.CardStatusRevoked).
		Error
}