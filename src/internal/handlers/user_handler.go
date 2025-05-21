package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"rfid/internal/models"
)

type UserHandler struct {
	db *gorm.DB
}

func NewUserHandler(db *gorm.DB) *UserHandler {
	return &UserHandler{db: db}
}

func (h *UserHandler) GetUsers(c *gin.Context) {
	var users []models.User

	if err := h.db.Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "nem sikerült lekérni a felhasználókat"})
		return
	}

	c.JSON(http.StatusOK, users)
}

func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "érvénytelen felhasználó azonosító"})
		return
	}

	query := h.db.Model(&models.User{})

	if c.Query("include_groups") == "true" {
		query = query.Preload("Groups")
	}
	if c.Query("include_permissions") == "true" {
		query = query.Preload("Permissions").Preload("Permissions.Room")
	}
	if c.Query("include_cards") == "true" {
		query = query.Preload("Cards")
	}

	var user models.User
	if err := query.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "felhasználó nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "nem sikerült lekérni a felhasználót"})
		}
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var input struct {
		Username  string `json:"username" binding:"required"`
		Password  string `json:"password" binding:"required"`
		FirstName string `json:"first_name" binding:"required"`
		LastName  string `json:"last_name" binding:"required"`
		Email     string `json:"email" binding:"required,email"`
		IsAdmin   bool   `json:"is_admin"`
		Active    bool   `json:"active"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen adatok. Kérjük, ellenőrizze a megadott információkat."})
		return
	}

	user := models.User{
		Username:  input.Username,
		Password:  input.Password,
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Email:     input.Email,
		IsAdmin:   input.IsAdmin,
		Active:    input.Active,
	}

	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "A felhasználó létrehozása sikertelen. A felhasználónév vagy email cím már használatban lehet."})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "érvénytelen felhasználó azonosító"})
		return
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Felhasználó nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Felhasználó lekérése sikertelen"})
		}
		return
	}

	var input struct {
		Username  string `json:"username"`
		Password  string `json:"password"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
		IsAdmin   *bool  `json:"is_admin"`
		Active    *bool  `json:"active"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Érvénytelen adatok. Kérjük, ellenőrizze a megadott információkat."})
		return
	}

	if input.Username != "" {
		user.Username = input.Username
	}
	if input.Password != "" {
		user.Password = input.Password
	}
	if input.FirstName != "" {
		user.FirstName = input.FirstName
	}
	if input.LastName != "" {
		user.LastName = input.LastName
	}
	if input.Email != "" {
		user.Email = input.Email
	}
	if input.IsAdmin != nil {
		user.IsAdmin = *input.IsAdmin
	}
	if input.Active != nil {
		user.Active = *input.Active
	}

	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "nem sikerült frissíteni a felhasználót"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "érvénytelen felhasználó azonosító"})
		return
	}

	var user models.User
	if err := h.db.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Felhasználó nem található"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Felhasználó lekérése sikertelen"})
		}
		return
	}

	if err := h.db.Unscoped().Delete(&models.User{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Felhasználó törlése sikertelen"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Felhasználó sikeresen törölve"})
}

func (h *UserHandler) GetUserCards(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "érvénytelen felhasználó azonosító"})
		return
	}

	var cards []models.Card
	if err := h.db.Where("user_id = ?", id).Find(&cards).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "nem sikerült lekérni a felhasználó kártyáit"})
		return
	}

	c.JSON(http.StatusOK, cards)
}
