package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"rfid/internal/models"
)

type GroupHandler struct {
	db *gorm.DB
}

func NewGroupHandler(db *gorm.DB) *GroupHandler {
	return &GroupHandler{db: db}
}

func (h *GroupHandler) GetGroups(c *gin.Context) {
	var groups []models.Group

	query := h.db

	if c.Query("include_users") == "true" {
		query = query.Preload("Users")
	}
	if c.Query("include_rooms") == "true" {
		query = query.Preload("Rooms")
	}
	if c.Query("include_parent") == "true" {
		query = query.Preload("Parent")
	}

	if err := query.Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Csoportok lekérdezése sikertelen"})
		return
	}

	c.JSON(http.StatusOK, groups)
}

func (h *GroupHandler) GetGroup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen csoport azonosító"})
		return
	}

	var group models.Group
	query := h.db

	if c.Query("include_users") == "true" {
		query = query.Preload("Users")
	}
	if c.Query("include_rooms") == "true" {
		query = query.Preload("Rooms")
	}
	if c.Query("include_parent") == "true" {
		query = query.Preload("Parent")
	}

	if err := query.First(&group, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Csoport nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Csoport lekérdezése sikertelen"})
		}
		return
	}

	c.JSON(http.StatusOK, group)
}

func (h *GroupHandler) CreateGroup(c *gin.Context) {
	var input struct {
		Name        string             `json:"name" binding:"required"`
		Description string             `json:"description"`
		ParentID    *uint              `json:"parent_id,omitempty"`
		AccessLevel models.AccessLevel `json:"access_level" binding:"required"`
		UserIDs     []uint             `json:"user_ids,omitempty"`
		RoomIDs     []uint             `json:"room_ids,omitempty"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen adatok. Kérjük, ellenőrizze a megadott információkat."})
		return
	}

	if input.ParentID != nil {
		var parentGroup models.Group
		if err := h.db.First(&parentGroup, *input.ParentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusBadRequest, gin.H{"error": "A megadott szülő csoport nem létezik"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Hiba történt a szülő csoport ellenőrzésekor"})
			}
			return
		}
	}

	group := models.Group{
		Name:        input.Name,
		Description: input.Description,
		ParentID:    input.ParentID,
		AccessLevel: models.AccessLevelRestricted,
	}

	tx := h.db.Begin()

	if err := tx.Create(&group).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "A csoport létrehozása sikertelen"})
		return
	}

	if len(input.UserIDs) > 0 {
		for _, userID := range input.UserIDs {
			var user models.User
			if err := tx.First(&user, userID).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": "Egy vagy több felhasználó nem található"})
				return
			}
			if err := tx.Model(&group).Association("Users").Append(&user); err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Hiba történt a felhasználók hozzáadásakor"})
				return
			}
		}
	}

	if len(input.RoomIDs) > 0 {
		for _, roomID := range input.RoomIDs {
			var room models.Room
			if err := tx.First(&room, roomID).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": "Egy vagy több szoba nem található"})
				return
			}
			if err := tx.Model(&group).Association("Rooms").Append(&room); err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Hiba történt a szobák hozzáadásakor"})
				return
			}
		}
	}

	tx.Commit()

	var createdGroup models.Group
	h.db.Preload("Users").Preload("Rooms").Preload("Parent").First(&createdGroup, group.ID)

	c.JSON(http.StatusCreated, createdGroup)
}

func (h *GroupHandler) UpdateGroup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen csoport azonosító"})
		return
	}

	var group models.Group
	if err := h.db.First(&group, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Csoport nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Csoport lekérdezése sikertelen"})
		}
		return
	}

	var input struct {
		Name        string             `json:"name"`
		Description string             `json:"description"`
		ParentID    *uint              `json:"parent_id,omitempty"`
		AccessLevel models.AccessLevel `json:"access_level"`
		UserIDs     []uint             `json:"user_ids,omitempty"`
		RoomIDs     []uint             `json:"room_ids,omitempty"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen adatok. Kérjük, ellenőrizze a megadott információkat."})
		return
	}

	if input.ParentID != nil && *input.ParentID != group.ID {
		var parentGroup models.Group
		if err := h.db.First(&parentGroup, *input.ParentID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusBadRequest, gin.H{"error": "A megadott szülő csoport nem létezik"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Hiba történt a szülő csoport ellenőrzésekor"})
			}
			return
		}
	} else if input.ParentID != nil && *input.ParentID == group.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Egy csoport nem lehet önmaga szülője"})
		return
	}

	if input.Name != "" {
		group.Name = input.Name
	}
	if input.Description != "" {
		group.Description = input.Description
	}
	if input.ParentID != nil {
		group.ParentID = input.ParentID
	}
	if input.AccessLevel != "" {
		group.AccessLevel = input.AccessLevel
	}

	tx := h.db.Begin()

	if err := tx.Save(&group).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "A csoport frissítése sikertelen"})
		return
	}

	if input.UserIDs != nil {
		if err := tx.Model(&group).Association("Users").Clear(); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Hiba történt a felhasználók eltávolításakor"})
			return
		}

		for _, userID := range input.UserIDs {
			var user models.User
			if err := tx.First(&user, userID).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": "Egy vagy több felhasználó nem található"})
				return
			}
			if err := tx.Model(&group).Association("Users").Append(&user); err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Hiba történt a felhasználók hozzáadásakor"})
				return
			}
		}
	}

	if input.RoomIDs != nil {
		if err := tx.Model(&group).Association("Rooms").Clear(); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Hiba történt a szobák eltávolításakor"})
			return
		}

		for _, roomID := range input.RoomIDs {
			var room models.Room
			if err := tx.First(&room, roomID).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": "Egy vagy több szoba nem található"})
				return
			}
			if err := tx.Model(&group).Association("Rooms").Append(&room); err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Hiba történt a szobák hozzáadásakor"})
				return
			}
		}
	}

	tx.Commit()

	var updatedGroup models.Group
	h.db.Preload("Users").Preload("Rooms").Preload("Parent").First(&updatedGroup, group.ID)

	c.JSON(http.StatusOK, updatedGroup)
}

func (h *GroupHandler) DeleteGroup(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen csoport azonosító"})
		return
	}

	var group models.Group
	if err := h.db.First(&group, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Csoport nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Csoport lekérdezése sikertelen"})
		}
		return
	}

	var childCount int64
	if err := h.db.Model(&models.Group{}).Where("parent_id = ?", id).Count(&childCount).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Hiba történt a gyermek csoportok ellenőrzésekor"})
		return
	}

	if childCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A csoport nem törölhető, mert vannak hozzá tartozó alcsoportok"})
		return
	}

	tx := h.db.Begin()

	if err := tx.Model(&group).Association("Users").Clear(); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Hiba történt a felhasználók eltávolításakor"})
		return
	}

	if err := tx.Model(&group).Association("Rooms").Clear(); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Hiba történt a szobák eltávolításakor"})
		return
	}

	if err := tx.Delete(&group).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "A csoport törlése sikertelen"})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Csoport sikeresen törölve"})
}

func (h *GroupHandler) AddUserToGroup(c *gin.Context) {
	groupID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen csoport azonosító"})
		return
	}

	var input struct {
		UserID uint `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen felhasználó azonosító"})
		return
	}

	var group models.Group
	if err := h.db.First(&group, groupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Csoport nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Csoport lekérdezése sikertelen"})
		}
		return
	}

	var user models.User
	if err := h.db.First(&user, input.UserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Felhasználó nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Felhasználó lekérdezése sikertelen"})
		}
		return
	}

	isInGroup := false
	var users []models.User
	h.db.Model(&group).Association("Users").Find(&users)
	for _, u := range users {
		if u.ID == user.ID {
			isInGroup = true
			break
		}
	}

	if isInGroup {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A felhasználó már tagja a csoportnak"})
		return
	}

	if err := h.db.Model(&group).Association("Users").Append(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "A felhasználó hozzáadása sikertelen"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Felhasználó sikeresen hozzáadva a csoporthoz"})
}

func (h *GroupHandler) RemoveUserFromGroup(c *gin.Context) {
	groupID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen csoport azonosító"})
		return
	}

	userID, err := strconv.Atoi(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen felhasználó azonosító"})
		return
	}

	var group models.Group
	if err := h.db.First(&group, groupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Csoport nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Csoport lekérdezése sikertelen"})
		}
		return
	}

	var user models.User
	if err := h.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Felhasználó nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Felhasználó lekérdezése sikertelen"})
		}
		return
	}

	if err := h.db.Model(&group).Association("Users").Delete(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "A felhasználó eltávolítása sikertelen"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Felhasználó sikeresen eltávolítva a csoportból"})
}

func (h *GroupHandler) AddRoomToGroup(c *gin.Context) {
	groupID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen csoport azonosító"})
		return
	}

	var input struct {
		RoomID uint `json:"room_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen szoba azonosító"})
		return
	}

	var group models.Group
	if err := h.db.First(&group, groupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Csoport nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Csoport lekérdezése sikertelen"})
		}
		return
	}

	var room models.Room
	if err := h.db.First(&room, input.RoomID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Szoba nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Szoba lekérdezése sikertelen"})
		}
		return
	}

	isInGroup := false
	var rooms []models.Room
	h.db.Model(&group).Association("Rooms").Find(&rooms)
	for _, r := range rooms {
		if r.ID == room.ID {
			isInGroup = true
			break
		}
	}

	if isInGroup {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A szoba már hozzá van rendelve a csoporthoz"})
		return
	}

	if err := h.db.Model(&group).Association("Rooms").Append(&room); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "A szoba hozzáadása sikertelen"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Szoba sikeresen hozzáadva a csoporthoz"})
}

func (h *GroupHandler) RemoveRoomFromGroup(c *gin.Context) {
	groupID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen csoport azonosító"})
		return
	}

	roomID, err := strconv.Atoi(c.Param("room_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen szoba azonosító"})
		return
	}

	var group models.Group
	if err := h.db.First(&group, groupID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Csoport nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Csoport lekérdezése sikertelen"})
		}
		return
	}

	var room models.Room
	if err := h.db.First(&room, roomID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Szoba nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Szoba lekérdezése sikertelen"})
		}
		return
	}

	if err := h.db.Model(&group).Association("Rooms").Delete(&room); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "A szoba eltávolítása sikertelen"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Szoba sikeresen eltávolítva a csoportból"})
}

func (h *GroupHandler) GetGroupUsers(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen csoport azonosító"})
		return
	}

	var group models.Group
	if err := h.db.First(&group, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Csoport nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Csoport lekérdezése sikertelen"})
		}
		return
	}

	var users []models.User
	if err := h.db.Model(&group).Association("Users").Find(&users); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Felhasználók lekérdezése sikertelen"})
		return
	}

	c.JSON(http.StatusOK, users)
}

func (h *GroupHandler) GetGroupRooms(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen csoport azonosító"})
		return
	}

	var group models.Group
	if err := h.db.First(&group, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Csoport nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Csoport lekérdezése sikertelen"})
		}
		return
	}

	var rooms []models.Room
	if err := h.db.Model(&group).Association("Rooms").Find(&rooms); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Szobák lekérdezése sikertelen"})
		return
	}

	c.JSON(http.StatusOK, rooms)
}
