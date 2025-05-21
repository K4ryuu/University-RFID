package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"rfid/internal/models"
	"rfid/internal/utils"
)

type SimulationHandler struct {
	DB                   *gorm.DB
	accessControlService *utils.AccessControlService
}

type SimulateAccessRequest struct {
	CardID uint `json:"card_id" binding:"required"`
	RoomID uint `json:"room_id" binding:"required"`
}

type SimulateAccessResponse struct {
	AccessGranted bool   `json:"access_granted"`
	Reason        string `json:"reason,omitempty"`
	Timestamp     string `json:"timestamp"`
}

func NewSimulationHandler(db *gorm.DB) *SimulationHandler {
	return &SimulationHandler{
		DB:                   db,
		accessControlService: utils.NewAccessControlService(db),
	}
}

func (h *SimulationHandler) SimulateAccess(c *gin.Context) {
	var req SimulateAccessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen kérés"})
		return
	}

	var card models.Card
	if result := h.DB.Preload("User").First(&card, req.CardID); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kártya nem található"})
		return
	}

	var room models.Room
	if result := h.DB.First(&room, req.RoomID); result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Helyiség nem található"})
		return
	}

	access, denialReason, err := h.accessControlService.CheckAccess(card.CardID, req.RoomID, "simulation")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Hiba történt a hozzáférés ellenőrzésekor"})
		return
	}

	now := time.Now()

	if access {
		c.JSON(http.StatusOK, SimulateAccessResponse{
			AccessGranted: true,
			Timestamp:     now.Format(time.RFC3339),
		})
		return
	}

	var reasonText string
	switch denialReason {
	case models.DenialReasonCardExpired:
		reasonText = "A kártya lejárt"
	case models.DenialReasonCardInactive:
		reasonText = "A kártya inaktív"
	case models.DenialReasonCardBlocked:
		reasonText = "A kártya le van tiltva"
	case models.DenialReasonCardRevoked:
		reasonText = "A kártya vissza lett vonva"
	case models.DenialReasonOutsideHours:
		reasonText = "A belépés nyitvatartási időn kívül történt"
	case models.DenialReasonRoomClosed:
		reasonText = "A helyiség zárva van"
	case models.DenialReasonPermissionError:
		reasonText = "Hiba a jogosultság ellenőrzésekor"
	case models.DenialReasonNoPermission:
		reasonText = "Nincs jogosultság a helyiséghez"
	default:
		reasonText = "Ismeretlen ok"
	}

	c.JSON(http.StatusOK, SimulateAccessResponse{
		AccessGranted: false,
		Reason:        reasonText,
		Timestamp:     now.Format(time.RFC3339),
	})
}
