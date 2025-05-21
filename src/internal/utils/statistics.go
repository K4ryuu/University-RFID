package utils

import (
	"time"

	"gorm.io/gorm"
)

type StatisticsService struct {
	db *gorm.DB
}

// NewStatisticsService creates a new instance of StatisticsService
func NewStatisticsService(db *gorm.DB) *StatisticsService {
	return &StatisticsService{
		db: db,
	}
}

type RoomUsageStats struct {
	RoomID       uint    `json:"room_id"`
	RoomName     string  `json:"room_name"`
	TotalEntries int     `json:"total_entries"`
	TotalDenials int     `json:"total_denials"`
	AccessRate   float64 `json:"access_rate"` // Percentage of successful entries
}

type CardUsageStats struct {
	CardID             uint       `json:"card_id"`
	UserFullName       string     `json:"user_full_name"`
	TotalUsage         int        `json:"total_usage"`
	UniqueRoomsVisited int        `json:"unique_rooms_visited"`
	LastUsed           *time.Time `json:"last_used"`
}

type TimeSeriesData struct {
	Timestamp time.Time `json:"timestamp"`
	Count     int       `json:"count"`
}

// GetRoomUsageStats gets usage statistics for all rooms or a specific room
func (ss *StatisticsService) GetRoomUsageStats(roomID uint, start, end time.Time) ([]RoomUsageStats, error) {
	var stats []RoomUsageStats

	// Build the query
	query := ss.db.Table("logs").
		Select("logs.room_id, rooms.name as room_name, "+
			"COUNT(CASE WHEN logs.access_result = 'granted' THEN 1 END) as total_entries, "+
			"COUNT(CASE WHEN logs.access_result = 'denied' THEN 1 END) as total_denials, "+
			"CAST(COUNT(CASE WHEN logs.access_result = 'granted' THEN 1 END) AS FLOAT) / COUNT(*) * 100 as access_rate").
		Joins("LEFT JOIN rooms ON logs.room_id = rooms.id").
		Where("logs.timestamp BETWEEN ? AND ?", start, end).
		Group("logs.room_id, rooms.name")

	// Filter by room if specified
	if roomID > 0 {
		query = query.Where("logs.room_id = ?", roomID)
	}

	if err := query.Scan(&stats).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// GetCardUsageStats gets usage statistics for all cards or a specific card
func (ss *StatisticsService) GetCardUsageStats(cardID uint, start, end time.Time) ([]CardUsageStats, error) {
	var stats []CardUsageStats

	// Build the query
	query := ss.db.Table("logs").
		Select("logs.card_id, "+
			"CONCAT(users.first_name, ' ', users.last_name) as user_full_name, "+
			"COUNT(*) as total_usage, "+
			"COUNT(DISTINCT logs.room_id) as unique_rooms_visited, "+
			"MAX(logs.timestamp) as last_used").
		Joins("LEFT JOIN cards ON logs.card_id = cards.id").
		Joins("LEFT JOIN users ON cards.user_id = users.id").
		Where("logs.timestamp BETWEEN ? AND ? AND logs.access_result = 'granted'", start, end).
		Group("logs.card_id, users.first_name, users.last_name")

	// Filter by card if specified
	if cardID > 0 {
		query = query.Where("logs.card_id = ?", cardID)
	}

	if err := query.Scan(&stats).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// GetAccessTimeSeriesData gets time series data for accesses
func (ss *StatisticsService) GetAccessTimeSeriesData(roomID uint, interval string, start, end time.Time) ([]TimeSeriesData, error) {
	var data []TimeSeriesData

	// Define time format based on interval
	var timeFormat string
	switch interval {
	case "hour":
		timeFormat = "2006-01-02 15:00:00"
	case "day":
		timeFormat = "2006-01-02 00:00:00"
	case "week":
		timeFormat = "2006-01-02" // This will need custom handling
	case "month":
		timeFormat = "2006-01-01 00:00:00"
	default:
		timeFormat = "2006-01-02 15:00:00" // Default to hourly
	}

	// Build query
	query := ss.db.Table("logs").
		Select("strftime(?, logs.timestamp) as timestamp_str, COUNT(*) as count", timeFormat).
		Where("logs.timestamp BETWEEN ? AND ? AND logs.access_result = 'granted'", start, end).
		Group("timestamp_str").
		Order("timestamp_str")

	// Filter by room if specified
	if roomID > 0 {
		query = query.Where("logs.room_id = ?", roomID)
	}

	// Get the data
	type rawData struct {
		TimestampStr string `gorm:"column:timestamp_str"`
		Count        int    `gorm:"column:count"`
	}

	var rawResults []rawData
	if err := query.Scan(&rawResults).Error; err != nil {
		return nil, err
	}

	// Convert string timestamps to time.Time
	for _, r := range rawResults {
		t, err := time.Parse(timeFormat, r.TimestampStr)
		if err != nil {
			continue
		}

		data = append(data, TimeSeriesData{
			Timestamp: t,
			Count:     r.Count,
		})
	}

	return data, nil
}

// GetMostAccessedRooms gets the most frequently accessed rooms
func (ss *StatisticsService) GetMostAccessedRooms(limit int, start, end time.Time) ([]RoomUsageStats, error) {
	var stats []RoomUsageStats

	if err := ss.db.Table("logs").
		Select("logs.room_id, rooms.name as room_name, "+
			"COUNT(*) as total_entries, "+
			"0 as total_denials, "+
			"100.0 as access_rate").
		Joins("LEFT JOIN rooms ON logs.room_id = rooms.id").
		Where("logs.timestamp BETWEEN ? AND ? AND logs.access_result = 'granted'", start, end).
		Group("logs.room_id, rooms.name").
		Order("total_entries DESC").
		Limit(limit).
		Scan(&stats).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// GetMostActiveUsers gets the most active users
func (ss *StatisticsService) GetMostActiveUsers(limit int, start, end time.Time) ([]struct {
	UserID      uint   `json:"user_id"`
	FullName    string `json:"full_name"`
	TotalAccess int    `json:"total_access"`
}, error) {
	var results []struct {
		UserID      uint   `json:"user_id"`
		FullName    string `json:"full_name"`
		TotalAccess int    `json:"total_access"`
	}

	if err := ss.db.Table("logs").
		Select("users.id as user_id, "+
			"CONCAT(users.first_name, ' ', users.last_name) as full_name, "+
			"COUNT(*) as total_access").
		Joins("LEFT JOIN cards ON logs.card_id = cards.id").
		Joins("LEFT JOIN users ON cards.user_id = users.id").
		Where("logs.timestamp BETWEEN ? AND ? AND logs.access_result = 'granted'", start, end).
		Group("users.id, users.first_name, users.last_name").
		Order("total_access DESC").
		Limit(limit).
		Scan(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}
