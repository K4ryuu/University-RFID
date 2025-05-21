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

type LogHandler struct {
	db           *gorm.DB
	statsService *utils.StatisticsService
}

func NewLogHandler(db *gorm.DB) *LogHandler {
	return &LogHandler{
		db:           db,
		statsService: utils.NewStatisticsService(db),
	}
}

func (h *LogHandler) GetLogs(c *gin.Context) {
	var logs []models.Log

	query := h.db.Model(&models.Log{}).Preload("Card").Preload("Card.User").Preload("Room")

	if cardID := c.Query("card_id"); cardID != "" {
		query = query.Where("card_id = ?", cardID)
	}

	if roomID := c.Query("room_id"); roomID != "" {
		query = query.Where("room_id = ?", roomID)
	}

	if result := c.Query("result"); result != "" {
		query = query.Where("access_result = ?", result)
	}

	if startDate := c.Query("start_date"); startDate != "" {
		query = query.Where("timestamp >= ?", startDate+" 00:00:00")
	}
	if endDate := c.Query("end_date"); endDate != "" {
		query = query.Where("timestamp <= ?", endDate+" 23:59:59")
	}

	limit := 50
	if limitStr := c.Query("limit"); limitStr != "" {
		if limitNum, err := strconv.Atoi(limitStr); err == nil && limitNum > 0 {
			limit = limitNum
		}
	}

	page := 0
	if pageStr := c.Query("page"); pageStr != "" {
		if pageNum, err := strconv.Atoi(pageStr); err == nil && pageNum > 0 {
			page = pageNum - 1
		}
	}

	if err := query.Order("timestamp DESC").Limit(limit).Offset(page * limit).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Naplóbejegyzések lekérése sikertelen"})
		return
	}

	c.JSON(http.StatusOK, logs)
}

func (h *LogHandler) GetLog(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen napló azonosító"})
		return
	}

	var log models.Log
	if err := h.db.Preload("Card").Preload("Card.User").Preload("Room").First(&log, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Naplóbejegyzés nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Naplóbejegyzés lekérése sikertelen"})
		}
		return
	}

	c.JSON(http.StatusOK, log)
}

func (h *LogHandler) CreateLog(c *gin.Context) {
	var input struct {
		CardID       uint                `json:"card_id" binding:"required"`
		RoomID       uint                `json:"room_id" binding:"required"`
		Timestamp    *time.Time          `json:"timestamp"`
		AccessResult models.AccessResult `json:"access_result" binding:"required"`
		DenialReason models.DenialReason `json:"denial_reason"`
		Description  string              `json:"description"`
		IPAddress    string              `json:"ip_address"`
		DeviceID     string              `json:"device_id"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var card models.Card
	if err := h.db.First(&card, input.CardID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen kártya azonosító"})
		return
	}

	var room models.Room
	if err := h.db.First(&room, input.RoomID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen helyiség azonosító"})
		return
	}

	timestamp := time.Now()
	if input.Timestamp != nil {
		timestamp = *input.Timestamp
	}

	log := models.Log{
		CardID:       input.CardID,
		RoomID:       input.RoomID,
		Timestamp:    timestamp,
		AccessResult: input.AccessResult,
		DenialReason: input.DenialReason,
		Description:  input.Description,
		IPAddress:    input.IPAddress,
		DeviceID:     input.DeviceID,
	}

	if err := h.db.Create(&log).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Naplóbejegyzés létrehozása sikertelen"})
		return
	}

	c.JSON(http.StatusCreated, log)
}

func (h *LogHandler) GetRoomStats(c *gin.Context) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, -1, 0)

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = t
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = t.Add(24*time.Hour - time.Second)
		}
	}

	var roomID uint
	if roomIDStr := c.Query("room_id"); roomIDStr != "" {
		if id, err := strconv.Atoi(roomIDStr); err == nil {
			roomID = uint(id)
		}
	}

	stats, err := h.statsService.GetRoomUsageStats(roomID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Helyiség statisztikák lekérése sikertelen: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *LogHandler) GetCardStats(c *gin.Context) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, -1, 0)

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = t
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = t.Add(24*time.Hour - time.Second)
		}
	}

	var cardID uint
	if cardIDStr := c.Query("card_id"); cardIDStr != "" {
		if id, err := strconv.Atoi(cardIDStr); err == nil {
			cardID = uint(id)
		}
	}

	stats, err := h.statsService.GetCardUsageStats(cardID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Kártya statisztikák lekérése sikertelen: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

func (h *LogHandler) GetAccessTimeSeries(c *gin.Context) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7)

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = t
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = t.Add(24*time.Hour - time.Second)
		}
	}

	var roomID uint
	if roomIDStr := c.Query("room_id"); roomIDStr != "" {
		if id, err := strconv.Atoi(roomIDStr); err == nil {
			roomID = uint(id)
		}
	}

	interval := c.DefaultQuery("interval", "day")

	data, err := h.statsService.GetAccessTimeSeriesData(roomID, interval, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Idősorozat adatok lekérése sikertelen: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

func (h *LogHandler) GetMostAccessedRooms(c *gin.Context) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, -1, 0)

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = t
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = t.Add(24*time.Hour - time.Second)
		}
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	rooms, err := h.statsService.GetMostAccessedRooms(limit, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Leggyakrabban használt helyiségek lekérése sikertelen: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, rooms)
}

func (h *LogHandler) GetMostActiveUsers(c *gin.Context) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, -1, 0)

	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = t
		}
	}

	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = t.Add(24*time.Hour - time.Second)
		}
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	users, err := h.statsService.GetMostActiveUsers(limit, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Legaktívabb felhasználók lekérése sikertelen: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}
