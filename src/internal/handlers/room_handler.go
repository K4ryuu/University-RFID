package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"rfid/internal/models"
)

type RoomHandler struct {
	db *gorm.DB
}

func NewRoomHandler(db *gorm.DB) *RoomHandler {
	return &RoomHandler{db: db}
}

func (h *RoomHandler) GetRooms(c *gin.Context) {
	var rooms []models.Room

	query := h.db.Model(&models.Room{})

	if building := c.Query("building"); building != "" {
		query = query.Where("building = ?", building)
	}

	if accessLevel := c.Query("access_level"); accessLevel != "" {
		query = query.Where("access_level = ?", accessLevel)
	}

	if err := query.Find(&rooms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Helyiségek lekérése sikertelen"})
		return
	}

	c.JSON(http.StatusOK, rooms)
}

func (h *RoomHandler) GetRoom(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen helyiség azonosító"})
		return
	}

	var room models.Room
	if err := h.db.First(&room, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Helyiség nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Helyiség lekérése sikertelen"})
		}
		return
	}

	c.JSON(http.StatusOK, room)
}

func (h *RoomHandler) CreateRoom(c *gin.Context) {
	var input struct {
		Name              string             `json:"name" binding:"required"`
		Description       string             `json:"description"`
		Building          string             `json:"building" binding:"required"`
		RoomNumber        string             `json:"room_number" binding:"required"`
		AccessLevel       models.AccessLevel `json:"access_level"`
		Capacity          int                `json:"capacity"`
		OperatingHours    string             `json:"operating_hours"`
		OperatingDays     string             `json:"operating_days"`
		SpecialConditions string             `json:"special_conditions"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	room := models.Room{
		Name:              input.Name,
		Description:       input.Description,
		Building:          input.Building,
		RoomNumber:        input.RoomNumber,
		AccessLevel:       input.AccessLevel,
		Capacity:          input.Capacity,
		OperatingHours:    input.OperatingHours,
		OperatingDays:     input.OperatingDays,
		SpecialConditions: input.SpecialConditions,
	}

	if room.AccessLevel == "" {
		room.AccessLevel = models.AccessLevelRestricted
	}

	if err := h.db.Create(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Helyiség létrehozása sikertelen"})
		return
	}

	c.JSON(http.StatusCreated, room)
}

func (h *RoomHandler) UpdateRoom(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen helyiség azonosító"})
		return
	}

	var room models.Room
	if err := h.db.First(&room, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Helyiség nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Helyiség lekérése sikertelen"})
		}
		return
	}

	var input struct {
		Name              string             `json:"name"`
		Description       string             `json:"description"`
		Building          string             `json:"building"`
		RoomNumber        string             `json:"room_number"`
		AccessLevel       models.AccessLevel `json:"access_level"`
		Capacity          *int               `json:"capacity"`
		OperatingHours    string             `json:"operating_hours"`
		OperatingDays     string             `json:"operating_days"`
		SpecialConditions string             `json:"special_conditions"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.Name != "" {
		room.Name = input.Name
	}
	if input.Description != "" {
		room.Description = input.Description
	}
	if input.Building != "" {
		room.Building = input.Building
	}
	if input.RoomNumber != "" {
		room.RoomNumber = input.RoomNumber
	}
	if input.AccessLevel != "" {
		room.AccessLevel = input.AccessLevel
	}
	if input.Capacity != nil {
		room.Capacity = *input.Capacity
	}
	if input.OperatingHours != "" {
		room.OperatingHours = input.OperatingHours
	}
	if input.OperatingDays != "" {
		room.OperatingDays = input.OperatingDays
	}
	if input.SpecialConditions != "" {
		room.SpecialConditions = input.SpecialConditions
	}

	if err := h.db.Save(&room).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Helyiség frissítése sikertelen"})
		return
	}

	c.JSON(http.StatusOK, room)
}

func (h *RoomHandler) DeleteRoom(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen helyiség azonosító"})
		return
	}

	if err := h.db.Delete(&models.Room{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Helyiség törlése sikertelen"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Helyiség sikeresen törölve"})
}

func (h *RoomHandler) GetRoomPermissions(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen helyiség azonosító"})
		return
	}

	var permissions []models.Permission
	if err := h.db.Where("room_id = ?", id).Preload("Card").Preload("Card.User").Find(&permissions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Helyiség jogosultságainak lekérése sikertelen"})
		return
	}

	c.JSON(http.StatusOK, permissions)
}

func (h *RoomHandler) GetRoomLogs(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen helyiség azonosító"})
		return
	}

	var logs []models.Log
	query := h.db.Where("room_id = ?", id).Preload("Card").Preload("Card.User")

	if result := c.Query("result"); result != "" {
		query = query.Where("access_result = ?", result)
	}

	if startDate := c.Query("start_date"); startDate != "" {
		query = query.Where("timestamp >= ?", startDate)
	}
	if endDate := c.Query("end_date"); endDate != "" {
		query = query.Where("timestamp <= ?", endDate)
	}

	limit := 50
	page := 0
	if pageStr := c.Query("page"); pageStr != "" {
		if pageNum, err := strconv.Atoi(pageStr); err == nil && pageNum > 0 {
			page = pageNum - 1
		}
	}

	if err := query.Order("timestamp DESC").Limit(limit).Offset(page * limit).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Helyiség naplóbejegyzéseinek lekérése sikertelen"})
		return
	}

	c.JSON(http.StatusOK, logs)
}
