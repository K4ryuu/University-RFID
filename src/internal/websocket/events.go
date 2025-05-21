package websocket

import (
	"log"
	"time"
)

type AccessEvent struct {
	CardID       string `json:"cardId"`
	RoomID       uint   `json:"roomId"`
	RoomName     string `json:"roomName"`
	AccessResult string `json:"accessResult"`
	UserID       uint   `json:"userId,omitempty"`
	UserName     string `json:"userName,omitempty"`
	Timestamp    string `json:"timestamp"`
}

type SystemEvent struct {
	Message   string `json:"message"`
	Severity  string `json:"severity"`
	Source    string `json:"source"`
	Timestamp string `json:"timestamp"`
}

func (h *Hub) BroadcastAccessEvent(accessEvent AccessEvent) {
	h.BroadcastToAdmins("access_event", accessEvent)

	if accessEvent.UserID > 0 {
		h.BroadcastToUser(accessEvent.UserID, "access_event", accessEvent)
	}
}

func (h *Hub) BroadcastSystemEvent(systemEvent SystemEvent, adminOnly bool) {
	if adminOnly {
		h.BroadcastToAdmins("system_event", systemEvent)
	} else {
		h.BroadcastToAuthenticated("system_event", systemEvent)
	}
}

func (h *Hub) BroadcastCardExpiration(cardID string, userID uint, expiryDate time.Time) {
	data := map[string]interface{}{
		"cardId":     cardID,
		"userId":     userID,
		"expiryDate": expiryDate.Format(time.RFC3339),
		"daysLeft":   int(time.Until(expiryDate).Hours() / 24),
	}

	h.BroadcastToAdmins("card_expiry", data)

	if userID > 0 {
		h.BroadcastToUser(userID, "card_expiry", data)
	}

	log.Printf("Card expiration notification sent for card %s (User ID: %d)", cardID, userID)
}