package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"rfid/internal/models"
	"rfid/internal/utils"
)

type PermissionHandler struct {
	db            *gorm.DB
	accessControl *utils.AccessControlService
}

func NewPermissionHandler(db *gorm.DB) *PermissionHandler {
	return &PermissionHandler{
		db:            db,
		accessControl: utils.NewAccessControlService(db),
	}
}

func (h *PermissionHandler) GetPermissions(c *gin.Context) {
	var permissions []models.Permission

	query := h.db.Model(&models.Permission{})

	if cardID := c.Query("card_id"); cardID != "" {
		query = query.Where("card_id = ?", cardID)
	}

	if userID := c.Query("user_id"); userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	if roomID := c.Query("room_id"); roomID != "" {
		query = query.Where("room_id = ?", roomID)
	}

	if activeStr := c.Query("active"); activeStr != "" {
		active := activeStr == "true"
		query = query.Where("active = ?", active)
	}

	if permType := c.Query("type"); permType != "" {
		if permType == "card" {
			query = query.Where("card_id IS NOT NULL")
		} else if permType == "user" {
			query = query.Where("user_id IS NOT NULL")
		}
	}

	if err := query.Preload("Card").Preload("Card.User").Preload("User").Preload("Room").Find(&permissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nem sikerült lekérni a jogosultságokat"})
		return
	}

	c.JSON(http.StatusOK, permissions)
}

func (h *PermissionHandler) GetPermission(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen jogosultság azonosító"})
		return
	}

	var permission models.Permission
	if err := h.db.Preload("Card").Preload("Card.User").Preload("User").Preload("Room").First(&permission, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Jogosultság nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Nem sikerült lekérni a jogosultságot"})
		}
		return
	}

	c.JSON(http.StatusOK, permission)
}

func (h *PermissionHandler) CreatePermission(c *gin.Context) {
	var input struct {
		CardID          *uint      `json:"card_id"`
		UserID          *uint      `json:"user_id"`
		RoomID          uint       `json:"room_id" binding:"required"`
		GrantedBy       uint       `json:"granted_by" binding:"required"`
		ValidFrom       *time.Time `json:"valid_from"`
		ValidUntil      *time.Time `json:"valid_until"`
		TimeRestriction string     `json:"time_restriction"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if (input.CardID == nil && input.UserID == nil) || (input.CardID != nil && input.UserID != nil) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Pontosan egyet kell megadni a card_id vagy user_id paraméterek közül"})
		return
	}

	if input.CardID != nil {
		var card models.Card
		if err := h.db.First(&card, *input.CardID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen kártya azonosító"})
			return
		}
	}

	if input.UserID != nil {
		var user models.User
		if err := h.db.First(&user, *input.UserID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen felhasználó azonosító"})
			return
		}
	}

	var room models.Room
	if err := h.db.First(&room, input.RoomID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen helyiség azonosító"})
		return
	}

	var user models.User
	if err := h.db.First(&user, input.GrantedBy).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen felhasználó azonosító a jogosultság megadójához"})
		return
	}

	validFrom := time.Now()
	if input.ValidFrom != nil {
		validFrom = *input.ValidFrom
	}

	permission := models.Permission{
		CardID:          input.CardID,
		UserID:          input.UserID,
		RoomID:          input.RoomID,
		GrantedBy:       input.GrantedBy,
		ValidFrom:       validFrom,
		ValidUntil:      input.ValidUntil,
		TimeRestriction: input.TimeRestriction,
		Active:          true,
	}

	if err := permission.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen jogosultság adatok"})
		return
	}

	if err := h.db.Create(&permission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nem sikerült létrehozni a jogosultságot: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, permission)
}

func (h *PermissionHandler) UpdatePermission(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen jogosultság azonosító"})
		return
	}

	var permission models.Permission
	if err := h.db.First(&permission, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Jogosultság nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Jogosultság lekérése sikertelen"})
		}
		return
	}

	var input struct {
		ValidFrom       *time.Time `json:"valid_from"`
		ValidUntil      *time.Time `json:"valid_until"`
		TimeRestriction string     `json:"time_restriction"`
		Active          *bool      `json:"active"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.ValidFrom != nil {
		permission.ValidFrom = *input.ValidFrom
	}
	if input.ValidUntil != nil {
		permission.ValidUntil = input.ValidUntil
	}
	if input.TimeRestriction != "" {
		permission.TimeRestriction = input.TimeRestriction
	}
	if input.Active != nil {
		permission.Active = *input.Active
	}

	if err := h.db.Save(&permission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Jogosultság frissítése sikertelen"})
		return
	}

	c.JSON(http.StatusOK, permission)
}

func (h *PermissionHandler) DeletePermission(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen jogosultság azonosító"})
		return
	}

	if err := h.db.Delete(&models.Permission{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Jogosultság törlése sikertelen"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Jogosultság sikeresen törölve"})
}

func (h *PermissionHandler) RevokePermission(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen jogosultság azonosító"})
		return
	}

	var permission models.Permission
	if err := h.db.First(&permission, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Jogosultság nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Jogosultság lekérése sikertelen"})
		}
		return
	}

	permission.Active = false

	if err := h.db.Save(&permission).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Jogosultság visszavonása sikertelen"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Jogosultság sikeresen visszavonva"})
}
