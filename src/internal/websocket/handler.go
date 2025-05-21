package websocket

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"rfid/internal/models"
)

type WebSocketHandler struct {
	db  *gorm.DB
	hub *Hub
}

func NewWebSocketHandler(db *gorm.DB) *WebSocketHandler {
	hub := NewHub()
	go hub.Run()

	return &WebSocketHandler{
		db:  db,
		hub: hub,
	}
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	var userID uint
	var isAdmin bool

	tokenString := c.Query("token")
	if tokenString != "" {
		jwtSecret := os.Getenv("JWT_SECRET")
		
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("váratlan aláírási módszer: %v", token.Header["alg"])
			}
			
			return []byte(jwtSecret), nil
		})

		if err == nil && token.Valid {
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if id, ok := claims["user_id"].(float64); ok {
					userID = uint(id)
				}
				if admin, ok := claims["is_admin"].(bool); ok {
					isAdmin = admin
				}
			}
		}
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket kapcsolat létrehozása sikertelen: %v", err)
		return
	}

	client := &Client{
		hub:     h.hub,
		conn:    conn,
		send:    make(chan []byte, 256),
		userID:  userID,
		isAdmin: isAdmin,
	}

	go client.HandleClientConnection()
}

func (h *WebSocketHandler) NotifyAccessEvent(cardID uint, roomID uint, result string, reason string) {
	var card models.Card
	if err := h.db.Preload("User").First(&card, cardID).Error; err != nil {
		log.Printf("Kártya adatok lekérése sikertelen: %v", err)
		return
	}

	var room models.Room
	if err := h.db.First(&room, roomID).Error; err != nil {
		log.Printf("Helyiség adatok lekérése sikertelen: %v", err)
		return
	}

	event := map[string]interface{}{
		"timestamp": map[string]interface{}{
			"unix": card.LastUsed.Unix(),
			"iso":  card.LastUsed.Format("2006-01-02T15:04:05Z07:00"),
		},
		"card": map[string]interface{}{
			"id":     card.ID,
			"card_id": card.CardID,
			"status": card.Status,
		},
		"room": map[string]interface{}{
			"id":          room.ID,
			"name":        room.Name,
			"building":    room.Building,
			"room_number": room.RoomNumber,
		},
		"result": result,
		"reason": reason,
	}

	if card.UserID > 0 {
		event["user"] = map[string]interface{}{
			"id":   card.User.ID,
			"name": card.User.FirstName + " " + card.User.LastName,
		}
		event["user_id"] = card.UserID
	}

	h.hub.SendAccessEvent(event)
}

func (h *WebSocketHandler) NotifyCardExpiration(card models.Card) {
	if card.UserID == 0 {
		return
	}

	var user models.User
	if err := h.db.First(&user, card.UserID).Error; err != nil {
		log.Printf("Felhasználó adatok lekérése sikertelen: %v", err)
		return
	}

	event := map[string]interface{}{
		"card": map[string]interface{}{
			"id":      card.ID,
			"card_id": card.CardID,
			"status":  card.Status,
			"expiry":  card.ExpiryDate.Format("2006-01-02"),
		},
		"user": map[string]interface{}{
			"id":   user.ID,
			"name": user.FirstName + " " + user.LastName,
		},
	}

	h.hub.BroadcastToAdmins("card_expiration", event)

	h.hub.BroadcastToUser(card.UserID, "card_expiration", event)
}

func (h *WebSocketHandler) GetHub() *Hub {
	return h.hub
}