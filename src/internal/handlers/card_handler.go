package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"rfid/internal/models"
	"rfid/internal/utils"
	"rfid/internal/websocket"
)

type CardHandler struct {
	db            *gorm.DB
	accessControl *utils.AccessControlService
	wsHandler     *websocket.WebSocketHandler
	wsEnabled     bool
}

func NewCardHandler(db *gorm.DB) *CardHandler {
	accessControl := utils.NewAccessControlService(db)

	return &CardHandler{
		db:            db,
		accessControl: accessControl,
		wsEnabled:     false,
	}
}

func (h *CardHandler) SetWebSocketHandler(wsHandler *websocket.WebSocketHandler) {
	h.wsHandler = wsHandler
	h.wsEnabled = (wsHandler != nil)

	h.accessControl.SetWebSocketHandler(wsHandler)
}

func (h *CardHandler) GetCards(c *gin.Context) {
	var cards []models.Card

	query := h.db.Model(&models.Card{})

	query = query.Preload("User")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	if err := query.Find(&cards).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kártyák lekérése sikertelen"})
		return
	}

	c.JSON(http.StatusOK, cards)
}

func (h *CardHandler) GetCard(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen kártya azonosító"})
		return
	}

	var card models.Card
	if err := h.db.Preload("User").Preload("Permissions").Preload("Permissions.Room").First(&card, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kártya nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Kártya adatok lekérése sikertelen"})
		}
		return
	}

	c.JSON(http.StatusOK, card)
}

func (h *CardHandler) CreateCard(c *gin.Context) {
	var input struct {
		UserID     uint              `json:"user_id" binding:"required"`
		CardID     string            `json:"card_id" binding:"required"`
		Status     models.CardStatus `json:"status"`
		ExpiryDate *time.Time        `json:"expiry_date"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen adatok. Kérjük, ellenőrizze a bevitt információkat."})
		return
	}

	var user models.User
	if err := h.db.First(&user, input.UserID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen felhasználó azonosító"})
		return
	}

	var existingUserCard models.Card
	if result := h.db.Where("user_id = ?", input.UserID).First(&existingUserCard); result.Error == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "A felhasználónak már van kártyája. Egy felhasználóhoz csak egy kártya tartozhat."})
		return
	}

	var count int64
	if err := h.db.Model(&models.Card{}).Where("card_id = ?", input.CardID).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Adatbázis hiba történt a kártya ellenőrzése közben."})
		return
	}

	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Kártya azonosító már regisztrálva van."})
		return
	}

	card, err := h.accessControl.RegisterCard(input.UserID, input.CardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kártya regisztrálása sikertelen: " + err.Error()})
		return
	}

	if input.Status != "" {
		card.Status = input.Status
	}
	if input.ExpiryDate != nil {
		card.ExpiryDate = input.ExpiryDate
	}

	if card.Status != models.CardStatusActive {
		if err := h.db.Save(&card).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Kártya státusz frissítése sikertelen"})
			return
		}
	}

	h.db.Preload("User").First(&card, card.ID)

	if h.wsEnabled {
		event := map[string]interface{}{
			"action": "card_created",
			"card": map[string]interface{}{
				"id":      card.ID,
				"card_id": card.CardID,
				"status":  card.Status,
			},
			"user": map[string]interface{}{
				"id":   user.ID,
				"name": user.FirstName + " " + user.LastName,
			},
		}
		h.wsHandler.GetHub().BroadcastToAdmins("card_event", event)
	}

	c.JSON(http.StatusCreated, card)
}

func (h *CardHandler) UpdateCard(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen kártya azonosító"})
		return
	}

	var card models.Card
	if err := h.db.First(&card, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kártya nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Kártya adatok lekérése sikertelen"})
		}
		return
	}

	var input struct {
		UserID      *uint             `json:"user_id"`
		Status      models.CardStatus `json:"status"`
		ExpiryDate  *time.Time        `json:"expiry_date"`
		CardID      string            `json:"card_id"`
		Description string            `json:"description"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen adatok. Kérjük, ellenőrizze a bevitt információkat."})
		return
	}

	oldStatus := card.Status
	oldExpiryDate := card.ExpiryDate

	if input.UserID != nil {
		var user models.User
		if err := h.db.First(&user, *input.UserID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen felhasználó azonosító"})
			return
		}

		if *input.UserID != card.UserID {
			var existingUserCard models.Card
			if result := h.db.Where("user_id = ?", *input.UserID).First(&existingUserCard); result.Error == nil {
				c.JSON(http.StatusConflict, gin.H{"error": "A kiválasztott felhasználónak már van kártyája. Egy felhasználóhoz csak egy kártya tartozhat."})
				return
			}
		}

		card.UserID = *input.UserID
	}

	if input.Status != "" {
		card.Status = input.Status
	}

	if input.ExpiryDate != nil {
		card.ExpiryDate = input.ExpiryDate
	}

	if input.CardID != "" && input.CardID != card.CardID {
		var existingCard models.Card
		if result := h.db.Where("card_id = ? AND id != ?", input.CardID, card.ID).First(&existingCard); result.Error == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "Ez a kártya azonosító már használatban van."})
			return
		}

		card.CardID = input.CardID

		encryptedCardID, err := utils.EncryptCardID(input.CardID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Kártya azonosító titkosítása sikertelen"})
			return
		}
		card.EncryptedCardID = encryptedCardID
	}

	if err := h.db.Save(&card).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kártya adatok frissítése sikertelen"})
		return
	}

	h.db.Preload("User").Preload("Permissions").Preload("Permissions.Room").First(&card, card.ID)

	if h.wsEnabled && (oldStatus != card.Status ||
		(oldExpiryDate != nil && card.ExpiryDate != nil && !oldExpiryDate.Equal(*card.ExpiryDate)) ||
		(oldExpiryDate == nil && card.ExpiryDate != nil) ||
		(oldExpiryDate != nil && card.ExpiryDate == nil)) {

		event := map[string]interface{}{
			"action": "card_updated",
			"card": map[string]interface{}{
				"id":      card.ID,
				"card_id": card.CardID,
				"status":  card.Status,
				"expiry":  card.ExpiryDate,
			},
			"user": map[string]interface{}{
				"id":   card.User.ID,
				"name": card.User.FirstName + " " + card.User.LastName,
			},
			"changes": map[string]interface{}{
				"status_changed": oldStatus != card.Status,
				"expiry_changed": (oldExpiryDate != nil && card.ExpiryDate != nil && !oldExpiryDate.Equal(*card.ExpiryDate)) ||
					(oldExpiryDate == nil && card.ExpiryDate != nil) ||
					(oldExpiryDate != nil && card.ExpiryDate == nil),
			},
		}

		h.wsHandler.GetHub().BroadcastToAdmins("card_event", event)

		if card.UserID > 0 {
			h.wsHandler.GetHub().BroadcastToUser(card.UserID, "card_event", event)
		}
	}

	c.JSON(http.StatusOK, card)
}

func (h *CardHandler) DeleteCard(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen kártya azonosító"})
		return
	}

	var card models.Card
	if err := h.db.Preload("User").First(&card, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kártya nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Kártya adatok lekérése sikertelen"})
		}
		return
	}

	userID := card.UserID
	userName := ""
	if card.User.ID > 0 {
		userName = card.User.FirstName + " " + card.User.LastName
	}

	if err := h.db.Unscoped().Where("card_id = ?", id).Delete(&models.Permission{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Jogosultságok törlése sikertelen"})
		return
	}

	if err := h.db.Unscoped().Delete(&models.Card{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kártya törlése sikertelen"})
		return
	}

	if h.wsEnabled {
		event := map[string]interface{}{
			"action": "card_deleted",
			"card": map[string]interface{}{
				"id":      card.ID,
				"card_id": card.CardID,
			},
		}

		if userID > 0 {
			event["user"] = map[string]interface{}{
				"id":   userID,
				"name": userName,
			}

			h.wsHandler.GetHub().BroadcastToUser(userID, "card_event", event)
		}

		h.wsHandler.GetHub().BroadcastToAdmins("card_event", event)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Kártya sikeresen törölve"})
}

func (h *CardHandler) BlockCard(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen kártya azonosító"})
		return
	}

	if err := h.accessControl.BlockCard(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kártya zárolása sikertelen"})
		return
	}

	var card models.Card
	h.db.Preload("User").First(&card, id)

	if h.wsEnabled {
		event := map[string]interface{}{
			"action": "card_blocked",
			"card": map[string]interface{}{
				"id":      card.ID,
				"card_id": card.CardID,
				"status":  card.Status,
			},
		}

		if card.UserID > 0 {
			event["user"] = map[string]interface{}{
				"id":   card.User.ID,
				"name": card.User.FirstName + " " + card.User.LastName,
			}

			h.wsHandler.GetHub().BroadcastToUser(card.UserID, "card_event", event)
		}

		h.wsHandler.GetHub().BroadcastToAdmins("card_event", event)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Kártya sikeresen zárolva",
		"card":    card,
	})
}

func (h *CardHandler) UnblockCard(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen kártya azonosító"})
		return
	}

	if err := h.accessControl.UnblockCard(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kártya zárolásának feloldása sikertelen"})
		return
	}

	var card models.Card
	h.db.Preload("User").First(&card, id)

	if h.wsEnabled {
		event := map[string]interface{}{
			"action": "card_unblocked",
			"card": map[string]interface{}{
				"id":      card.ID,
				"card_id": card.CardID,
				"status":  card.Status,
			},
		}

		if card.UserID > 0 {
			event["user"] = map[string]interface{}{
				"id":   card.User.ID,
				"name": card.User.FirstName + " " + card.User.LastName,
			}

			h.wsHandler.GetHub().BroadcastToUser(card.UserID, "card_event", event)
		}

		h.wsHandler.GetHub().BroadcastToAdmins("card_event", event)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Kártya zárolása sikeresen feloldva",
		"card":    card,
	})
}

func (h *CardHandler) RevokeCard(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen kártya azonosító"})
		return
	}

	if err := h.accessControl.RevokeCard(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kártya visszavonása sikertelen"})
		return
	}

	var card models.Card
	h.db.Preload("User").First(&card, id)

	if h.wsEnabled {
		event := map[string]interface{}{
			"action": "card_revoked",
			"card": map[string]interface{}{
				"id":      card.ID,
				"card_id": card.CardID,
				"status":  card.Status,
			},
		}

		if card.UserID > 0 {
			event["user"] = map[string]interface{}{
				"id":   card.User.ID,
				"name": card.User.FirstName + " " + card.User.LastName,
			}

			h.wsHandler.GetHub().BroadcastToUser(card.UserID, "card_event", event)
		}

		h.wsHandler.GetHub().BroadcastToAdmins("card_event", event)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Kártya sikeresen visszavonva",
		"card":    card,
	})
}

func (h *CardHandler) GetExpiringCards(c *gin.Context) {
	days := 30
	if daysStr := c.Query("days"); daysStr != "" {
		if daysNum, err := strconv.Atoi(daysStr); err == nil && daysNum > 0 {
			days = daysNum
		}
	}

	now := time.Now()
	expiryLimit := now.AddDate(0, 0, days)

	var expiringCards []models.Card
	if err := h.db.
		Preload("User").
		Where("status = ? AND expiry_date IS NOT NULL AND expiry_date <= ? AND expiry_date >= ?",
			models.CardStatusActive, expiryLimit, now).
		Order("expiry_date ASC").
		Find(&expiringCards).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lejáró kártyák lekérdezése sikertelen"})
		return
	}

	if h.wsEnabled && len(expiringCards) > 0 {
		for _, card := range expiringCards {
			h.wsHandler.NotifyCardExpiration(card)
		}
	}

	c.JSON(http.StatusOK, expiringCards)
}

func (h *CardHandler) CheckAccess(c *gin.Context) {
	var input struct {
		CardID   string `json:"card_id" binding:"required"`
		RoomID   uint   `json:"room_id" binding:"required"`
		DeviceID string `json:"device_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen adatok. Kérjük, adja meg a kártya azonosítót és a helyiség azonosítót."})
		return
	}

	var room models.Room
	if err := h.db.First(&room, input.RoomID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Helyiség nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Helyiség adatok lekérése sikertelen"})
		}
		return
	}

	hasAccess, reason, err := h.accessControl.CheckAccess(input.CardID, input.RoomID, input.DeviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Hozzáférés ellenőrzése sikertelen: " + err.Error()})
		return
	}

	reasonText := ""
	if reason != "" {
		switch reason {
		case models.DenialReasonNoPermission:
			reasonText = "Nincs jogosultság a helyiséghez"
		case models.DenialReasonCardInactive:
			reasonText = "Inaktív kártya"
		case models.DenialReasonCardExpired:
			reasonText = "Lejárt kártya"
		case models.DenialReasonOutsideHours:
			reasonText = "Nyitvatartási időn kívül"
		case models.DenialReasonRoomClosed:
			reasonText = "Helyiség zárva"
		case models.DenialReasonCardBlocked:
			reasonText = "Zárolt kártya"
		case models.DenialReasonCardRevoked:
			reasonText = "Visszavont kártya"
		case models.DenialReasonPermissionError:
			reasonText = "Jogosultság hiba"
		default:
			reasonText = string(reason)
		}
	}

	var card models.Card
	var user models.User
	cardData := gin.H{}

	if result := h.db.Where("card_id = ?", input.CardID).First(&card); result.Error == nil {
		if h.db.First(&user, card.UserID).Error == nil {
			cardData = gin.H{
				"id":      card.ID,
				"card_id": card.CardID,
				"status":  card.Status,
				"user": gin.H{
					"id":   user.ID,
					"name": user.FirstName + " " + user.LastName,
				},
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"has_access": hasAccess,
		"room": gin.H{
			"id":          room.ID,
			"name":        room.Name,
			"building":    room.Building,
			"room_number": room.RoomNumber,
		},
		"reason_code": string(reason),
		"reason_text": reasonText,
		"card":        cardData,
		"timestamp":   time.Now().Format(time.RFC3339),
	})
}