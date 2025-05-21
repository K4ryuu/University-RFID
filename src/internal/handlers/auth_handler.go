package handlers

import (
	"errors"
	"net/http"
	"regexp"
	"sync"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"rfid/internal/middleware"
	"rfid/internal/models"
)

type loginAttempt struct {
	username  string
	ipAddress string
	timestamp time.Time
	success   bool
}

type AuthHandler struct {
	db               *gorm.DB
	authMiddleware   *middleware.AuthMiddleware
	loginAttempts    []loginAttempt
	rateLimitWindow  time.Duration
	maxLoginAttempts int
	blockDuration    time.Duration
	blockedIPs       map[string]time.Time
	blockedUsernames map[string]time.Time
	attemptsMutex    sync.Mutex
}

func NewAuthHandler(db *gorm.DB) *AuthHandler {
	return &AuthHandler{
		db:               db,
		authMiddleware:   middleware.NewAuthMiddleware(db),
		loginAttempts:    []loginAttempt{},
		rateLimitWindow:  10 * time.Minute,
		maxLoginAttempts: 3,
		blockDuration:    45 * time.Minute,
		blockedIPs:       make(map[string]time.Time),
		blockedUsernames: make(map[string]time.Time),
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ipAddress := c.ClientIP()

	if h.isIPBlocked(ipAddress) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Túl sok sikertelen bejelentkezési kísérlet. Kérjük, próbálja újra később."})
		return
	}

	if h.isUsernameBlocked(input.Username) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Túl sok sikertelen bejelentkezési kísérlet ezzel a felhasználónévvel. Kérjük, próbálja újra később."})
		return
	}

	var user models.User
	if err := h.db.Where("username = ?", input.Username).First(&user).Error; err != nil {
		h.recordLoginAttempt(input.Username, ipAddress, false)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Érvénytelen felhasználónév vagy jelszó"})
		return
	}

	if !user.Active {
		h.recordLoginAttempt(input.Username, ipAddress, false)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "A fiók inaktív"})
		return
	}

	if !user.CheckPassword(input.Password) {
		h.recordLoginAttempt(input.Username, ipAddress, false)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Érvénytelen felhasználónév vagy jelszó"})
		return
	}

	token, err := h.authMiddleware.GenerateToken(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nem sikerült a token generálása"})
		return
	}

	h.recordLoginAttempt(input.Username, ipAddress, true)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"firstName": user.FirstName,
			"lastName":  user.LastName,
			"email":     user.Email,
			"isAdmin":   user.IsAdmin,
		},
	})
}

func (h *AuthHandler) Register(c *gin.Context) {
	var input struct {
		Username  string `json:"username" binding:"required"`
		Password  string `json:"password" binding:"required"`
		FirstName string `json:"first_name" binding:"required"`
		LastName  string `json:"last_name" binding:"required"`
		Email     string `json:"email" binding:"required,email"`
		IsAdmin   bool   `json:"is_admin"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := validatePasswordStrength(input.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := models.User{
		Username:  input.Username,
		Password:  input.Password,
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Email:     input.Email,
		IsAdmin:   input.IsAdmin,
		Active:    true,
	}

	if err := h.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nem sikerült a felhasználó regisztrálása"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"message":  "User registered successfully",
	})
}

func (h *AuthHandler) GetMe(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "A felhasználó nincs bejelentkezve"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var input struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := validatePasswordStrength(input.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "A felhasználó nincs bejelentkezve"})
		return
	}

	user := userInterface.(models.User)

	if !user.CheckPassword(input.OldPassword) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "A jelenlegi jelszó helytelen"})
		return
	}

	user.Password = input.NewPassword

	if err := h.db.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Nem sikerült a jelszó frissítése"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "A jelszó sikeresen megváltoztatva"})
}

func (h *AuthHandler) recordLoginAttempt(username, ipAddress string, success bool) {
	h.attemptsMutex.Lock()
	defer h.attemptsMutex.Unlock()

	attempt := loginAttempt{
		username:  username,
		ipAddress: ipAddress,
		timestamp: time.Now(),
		success:   success,
	}
	h.loginAttempts = append(h.loginAttempts, attempt)

	if success {
		delete(h.blockedIPs, ipAddress)
		delete(h.blockedUsernames, username)
		return
	}

	cutoffTime := time.Now().Add(-h.rateLimitWindow)
	newAttempts := []loginAttempt{}
	for _, a := range h.loginAttempts {
		if a.timestamp.After(cutoffTime) {
			newAttempts = append(newAttempts, a)
		}
	}
	h.loginAttempts = newAttempts

	ipFailures := 0
	for _, a := range h.loginAttempts {
		if a.ipAddress == ipAddress && !a.success {
			ipFailures++
		}
	}

	usernameFailures := 0
	for _, a := range h.loginAttempts {
		if a.username == username && !a.success {
			usernameFailures++
		}
	}

	if ipFailures >= h.maxLoginAttempts {
		h.blockedIPs[ipAddress] = time.Now().Add(h.blockDuration)
	}

	if usernameFailures >= h.maxLoginAttempts {
		h.blockedUsernames[username] = time.Now().Add(h.blockDuration)
	}
}

func (h *AuthHandler) isIPBlocked(ipAddress string) bool {
	h.attemptsMutex.Lock()
	defer h.attemptsMutex.Unlock()

	blockUntil, exists := h.blockedIPs[ipAddress]
	if !exists {
		return false
	}

	if time.Now().After(blockUntil) {
		delete(h.blockedIPs, ipAddress)
		return false
	}

	return true
}

func (h *AuthHandler) isUsernameBlocked(username string) bool {
	h.attemptsMutex.Lock()
	defer h.attemptsMutex.Unlock()

	blockUntil, exists := h.blockedUsernames[username]
	if !exists {
		return false
	}

	if time.Now().After(blockUntil) {
		delete(h.blockedUsernames, username)
		return false
	}

	return true
}

func validatePasswordStrength(password string) error {
	if len(password) < 8 {
		return errors.New("a jelszónak legalább 8 karakter hosszúnak kell lennie")
	}

	hasUpper := false
	for _, c := range password {
		if unicode.IsUpper(c) {
			hasUpper = true
			break
		}
	}
	if !hasUpper {
		return errors.New("a jelszónak legalább egy nagybetűt kell tartalmaznia")
	}

	hasLower := false
	for _, c := range password {
		if unicode.IsLower(c) {
			hasLower = true
			break
		}
	}
	if !hasLower {
		return errors.New("a jelszónak legalább egy kisbetűt kell tartalmaznia")
	}

	hasDigit := false
	for _, c := range password {
		if unicode.IsDigit(c) {
			hasDigit = true
			break
		}
	}
	if !hasDigit {
		return errors.New("a jelszónak legalább egy számot kell tartalmaznia")
	}

	specialChar := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`)
	if !specialChar.MatchString(password) {
		return errors.New("a jelszónak legalább egy speciális karaktert kell tartalmaznia (pl. !@#$%^&*)")
	}

	return nil
}
