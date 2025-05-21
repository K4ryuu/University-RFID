package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"rfid/internal/config"
	"rfid/internal/models"
	"rfid/internal/routes"
)

func getTimePtr(t time.Time) *time.Time {
	return &t
}

func getUintPtr(u uint) *uint {
	return &u
}

func main() {
	appConfig := config.Load()

	db, err := setupDatabase(appConfig)
	if err != nil {
		log.Fatalf("Adatbázis kapcsolódás sikertelen: %v", err)
	}

	router := setupRouter(db, appConfig)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", appConfig.Port),
		Handler: router,
	}

	go func() {
		log.Printf("Szerver elindult a %s porton\n", appConfig.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Szerver indítása sikertelen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Szerver leállítása folyamatban...")
}

func setupDatabase(config *config.Config) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(config.DBPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&models.User{}, &models.Card{}, &models.Room{}, &models.Permission{}, &models.Log{}, &models.Group{}); err != nil {
		return nil, fmt.Errorf("adatbázis migráció sikertelen: %w", err)
	}

	if err := createInitialData(db); err != nil {
		return nil, fmt.Errorf("kezdeti adatok létrehozása sikertelen: %w", err)
	}

	return db, nil
}

func createInitialData(db *gorm.DB) error {
	var adminCount int64
	if err := db.Model(&models.User{}).Where("is_admin = ?", true).Count(&adminCount).Error; err != nil {
		return err
	}

	adminUsername := getEnv("ADMIN_USERNAME", "admin")

	if adminCount == 0 {
		var existingUser models.User
		result := db.Where("username = ?", adminUsername).First(&existingUser)

		if result.Error == nil {
			existingUser.IsAdmin = true
			if err := db.Save(&existingUser).Error; err != nil {
				return err
			}
			log.Println("Meglévő felhasználó admin jogosultsággal frissítve:", adminUsername)
		} else if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			adminPassword := getEnv("ADMIN_PASSWORD", "admin123")
			adminFirstName := getEnv("ADMIN_FIRST_NAME", "Rendszer")
			adminLastName := getEnv("ADMIN_LAST_NAME", "Adminisztrátor")
			adminEmail := getEnv("ADMIN_EMAIL", "admin@egyetem.hu")

			adminUser := models.User{
				Username:  adminUsername,
				Password:  adminPassword,
				FirstName: adminFirstName,
				LastName:  adminLastName,
				Email:     adminEmail,
				IsAdmin:   true,
				Active:    true,
			}

			if err := db.Create(&adminUser).Error; err != nil {
				return err
			}
			log.Println("Alapértelmezett admin felhasználó létrehozva (felhasználónév: admin, jelszó: admin123)")
		} else {
			return result.Error
		}
	}

	var userCount int64
	if err := db.Model(&models.User{}).Where("username != ?", adminUsername).Count(&userCount).Error; err != nil {
		return err
	}

	if userCount == 0 {
		users := []models.User{
			{
				Username:  "tanar",
				Password:  "Tanar123!",
				FirstName: "Kovács",
				LastName:  "János",
				Email:     "kovacs.janos@egyetem.hu",
				IsAdmin:   false,
				Active:    true,
			},
			{
				Username:  "hallgato1",
				Password:  "Hallgato123!",
				FirstName: "Nagy",
				LastName:  "Emma",
				Email:     "nagy.emma@hallgato.egyetem.hu",
				IsAdmin:   false,
				Active:    true,
			},
			{
				Username:  "hallgato2",
				Password:  "Hallgato123!",
				FirstName: "Szabó",
				LastName:  "Péter",
				Email:     "szabo.peter@hallgato.egyetem.hu",
				IsAdmin:   false,
				Active:    true,
			},
			{
				Username:  "vendeg",
				Password:  "Vendeg123!",
				FirstName: "Vendég",
				LastName:  "Felhasználó",
				Email:     "vendeg@egyetem.hu",
				IsAdmin:   false,
				Active:    true,
			},
			{
				Username:  "inaktiv",
				Password:  "Inaktiv123!",
				FirstName: "Inaktív",
				LastName:  "Felhasználó",
				Email:     "inaktiv@egyetem.hu",
				IsAdmin:   false,
				Active:    false,
			},
		}

		for _, user := range users {
			if err := db.Create(&user).Error; err != nil {
				return err
			}
		}
		log.Println("Demo felhasználók létrehozva")
	}

	var roomCount int64
	if err := db.Model(&models.Room{}).Count(&roomCount).Error; err != nil {
		return err
	}

	if roomCount == 0 {
		rooms := []models.Room{
			{
				Name:           "Főbejárat",
				Description:    "Egyetem főbejárata",
				Building:       "Főépület",
				RoomNumber:     "101",
				AccessLevel:    models.AccessLevelPublic,
				OperatingHours: "07:00-20:00",
				OperatingDays:  "1,2,3,4,5",
			},
			{
				Name:           "Informatikai Labor",
				Description:    "Számítógépes labor hallgatók számára",
				Building:       "Informatikai Épület",
				RoomNumber:     "I-203",
				AccessLevel:    models.AccessLevelRestricted,
				OperatingHours: "08:00-18:00",
				OperatingDays:  "1,2,3,4,5",
			},
			{
				Name:           "Könyvtár",
				Description:    "Egyetemi könyvtár",
				Building:       "Központi Épület",
				RoomNumber:     "K-002",
				AccessLevel:    models.AccessLevelPublic,
				OperatingHours: "08:00-20:00",
				OperatingDays:  "1,2,3,4,5,6",
			},
			{
				Name:           "Tanulmányi Osztály",
				Description:    "Tanulmányi ügyintézés",
				Building:       "Főépület",
				RoomNumber:     "F-112",
				AccessLevel:    models.AccessLevelPublic,
				OperatingHours: "09:00-16:00",
				OperatingDays:  "1,2,3,4,5",
			},
			{
				Name:           "Szerver szoba",
				Description:    "Központi szerverek, hálózati eszközök",
				Building:       "Informatikai Épület",
				RoomNumber:     "I-001",
				AccessLevel:    models.AccessLevelAdmin,
				OperatingHours: "00:00-23:59",
				OperatingDays:  "0,1,2,3,4,5,6",
			},
			{
				Name:           "Kutatólabor",
				Description:    "Kutatási projektek laborja",
				Building:       "Kutatási Épület",
				RoomNumber:     "K-101",
				AccessLevel:    models.AccessLevelRestricted,
				OperatingHours: "00:00-23:59",
				OperatingDays:  "0,1,2,3,4,5,6",
			},
			{
				Name:           "Előadóterem",
				Description:    "Központi előadóterem",
				Building:       "Főépület",
				RoomNumber:     "F-201",
				AccessLevel:    models.AccessLevelPublic,
				OperatingHours: "08:00-20:00",
				OperatingDays:  "1,2,3,4,5",
			},
		}

		for _, room := range rooms {
			if err := db.Create(&room).Error; err != nil {
				return err
			}
		}
		log.Println("Alapértelmezett helyiségek létrehozva")
	}

	var groupCount int64
	if err := db.Model(&models.Group{}).Count(&groupCount).Error; err != nil {
		return err
	}

	if groupCount == 0 {
		groups := []models.Group{
			{
				Name:        "Oktatók",
				Description: "Egyetemi oktatók csoportja",
			},
			{
				Name:        "Hallgatók",
				Description: "Az egyetem hallgatói",
			},
			{
				Name:        "Vendégek",
				Description: "Vendég felhasználók korlátozott hozzáféréssel",
			},
			{
				Name:        "Informatikus BSc",
				Description: "Informatikus BSc hallgatók csoportja",
			},
			{
				Name:        "IT Adminisztrátorok",
				Description: "IT rendszergazdák csoportja",
			},
		}

		for _, group := range groups {
			if err := db.Create(&group).Error; err != nil {
				return err
			}
		}
		log.Println("Csoportok létrehozva")

		var teacherGroup, studentGroup, guestGroup, adminGroup models.Group

		db.Where("name = ?", "Oktatók").First(&teacherGroup)
		db.Where("name = ?", "Hallgatók").First(&studentGroup)
		db.Where("name = ?", "Vendégek").First(&guestGroup)
		db.Where("name = ?", "IT Adminisztrátorok").First(&adminGroup)

		var teacher, student1, student2, guest, admin models.User

		db.Where("username = ?", "tanar").First(&teacher)
		db.Where("username = ?", "hallgato1").First(&student1)
		db.Where("username = ?", "hallgato2").First(&student2)
		db.Where("username = ?", "vendeg").First(&guest)
		db.Where("username = ?", "admin").First(&admin)

		if teacher.ID != 0 && teacherGroup.ID != 0 {
			db.Model(&teacherGroup).Association("Users").Append(&teacher)
		}

		if student1.ID != 0 && studentGroup.ID != 0 {
			db.Model(&studentGroup).Association("Users").Append(&student1)
		}

		if student2.ID != 0 && studentGroup.ID != 0 {
			db.Model(&studentGroup).Association("Users").Append(&student2)
		}

		if guest.ID != 0 && guestGroup.ID != 0 {
			db.Model(&guestGroup).Association("Users").Append(&guest)
		}

		if admin.ID != 0 && adminGroup.ID != 0 {
			db.Model(&adminGroup).Association("Users").Append(&admin)
		}
	}

	var cardCount int64
	if err := db.Model(&models.Card{}).Count(&cardCount).Error; err != nil {
		return err
	}

	if cardCount == 0 {
		var teacher, student1, student2, guest, admin models.User

		db.Where("username = ?", "tanar").First(&teacher)
		db.Where("username = ?", "hallgato1").First(&student1)
		db.Where("username = ?", "hallgato2").First(&student2)
		db.Where("username = ?", "vendeg").First(&guest)
		db.Where("username = ?", "admin").First(&admin)

		now := time.Now()

		cards := []models.Card{
			{
				CardID:     "CARD001",
				UserID:     admin.ID,
				IssueDate:  now.AddDate(-1, 0, 0),
				ExpiryDate: getTimePtr(now.AddDate(2, 0, 0)),
				Status:     models.CardStatusActive,
			},
			{
				CardID:     "CARD002",
				UserID:     teacher.ID,
				IssueDate:  now.AddDate(0, -6, 0),
				ExpiryDate: getTimePtr(now.AddDate(0, 6, 0)),
				Status:     models.CardStatusActive,
			},
			{
				CardID:     "CARD003",
				UserID:     student1.ID,
				IssueDate:  now.AddDate(0, -3, 0),
				ExpiryDate: getTimePtr(now.AddDate(0, 0, 10)),
				Status:     models.CardStatusActive,
			},
			{
				CardID:     "CARD004",
				UserID:     student2.ID,
				IssueDate:  now.AddDate(0, -4, 0),
				ExpiryDate: getTimePtr(now.AddDate(0, 8, 0)),
				Status:     models.CardStatusActive,
			},
			{
				CardID:     "CARD005",
				UserID:     guest.ID,
				IssueDate:  now.AddDate(0, 0, -10),
				ExpiryDate: getTimePtr(now.AddDate(0, 0, 5)),
				Status:     models.CardStatusActive,
			},
			{
				CardID:     "CARD006",
				UserID:     student1.ID,
				IssueDate:  now.AddDate(-1, 0, 0),
				ExpiryDate: getTimePtr(now.AddDate(-1, 1, 0)),
				Status:     models.CardStatusExpired,
			},
			{
				CardID:     "CARD007",
				UserID:     admin.ID,
				IssueDate:  now.AddDate(0, -1, 0),
				ExpiryDate: getTimePtr(now.AddDate(1, 0, 0)),
				Status:     models.CardStatusActive,
			},
		}

		for _, card := range cards {
			if err := db.Create(&card).Error; err != nil {
				return err
			}
		}
		log.Println("Demo kártyák létrehozva")

		var mainEntrance, itLab, library, studyOffice, serverRoom, researchLab, lectureHall models.Room

		db.Where("name = ?", "Főbejárat").First(&mainEntrance)
		db.Where("name = ?", "Informatikai Labor").First(&itLab)
		db.Where("name = ?", "Könyvtár").First(&library)
		db.Where("name = ?", "Tanulmányi Osztály").First(&studyOffice)
		db.Where("name = ?", "Szerver szoba").First(&serverRoom)
		db.Where("name = ?", "Kutatólabor").First(&researchLab)
		db.Where("name = ?", "Előadóterem").First(&lectureHall)

		var adminCard, teacherCard, student1Card, student2Card, guestCard models.Card

		db.Where("card_id = ?", "CARD001").First(&adminCard)
		db.Where("card_id = ?", "CARD002").First(&teacherCard)
		db.Where("card_id = ?", "CARD003").First(&student1Card)
		db.Where("card_id = ?", "CARD004").First(&student2Card)
		db.Where("card_id = ?", "CARD005").First(&guestCard)

		permissions := []models.Permission{
			{
				CardID:     getUintPtr(adminCard.ID),
				RoomID:     mainEntrance.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(-1, 0, 0),
				ValidUntil: getTimePtr(now.AddDate(2, 0, 0)),
				Active:     true,
			},
			{
				CardID:     getUintPtr(adminCard.ID),
				RoomID:     itLab.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(-1, 0, 0),
				ValidUntil: getTimePtr(now.AddDate(2, 0, 0)),
				Active:     true,
			},
			{
				CardID:     getUintPtr(adminCard.ID),
				RoomID:     library.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(-1, 0, 0),
				ValidUntil: getTimePtr(now.AddDate(2, 0, 0)),
				Active:     true,
			},
			{
				CardID:     getUintPtr(adminCard.ID),
				RoomID:     studyOffice.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(-1, 0, 0),
				ValidUntil: getTimePtr(now.AddDate(2, 0, 0)),
				Active:     true,
			},
			{
				CardID:     getUintPtr(adminCard.ID),
				RoomID:     serverRoom.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(-1, 0, 0),
				ValidUntil: getTimePtr(now.AddDate(2, 0, 0)),
				Active:     true,
			},
			{
				CardID:     getUintPtr(adminCard.ID),
				RoomID:     researchLab.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(-1, 0, 0),
				ValidUntil: getTimePtr(now.AddDate(2, 0, 0)),
				Active:     true,
			},
			{
				CardID:     getUintPtr(adminCard.ID),
				RoomID:     lectureHall.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(-1, 0, 0),
				ValidUntil: getTimePtr(now.AddDate(2, 0, 0)),
				Active:     true,
			},

			{
				CardID:     getUintPtr(teacherCard.ID),
				RoomID:     mainEntrance.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(0, -6, 0),
				ValidUntil: getTimePtr(now.AddDate(0, 6, 0)),
				Active:     true,
			},
			{
				CardID:     getUintPtr(teacherCard.ID),
				RoomID:     itLab.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(0, -6, 0),
				ValidUntil: getTimePtr(now.AddDate(0, 6, 0)),
				Active:     true,
			},
			{
				CardID:     getUintPtr(teacherCard.ID),
				RoomID:     library.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(0, -6, 0),
				ValidUntil: getTimePtr(now.AddDate(0, 6, 0)),
				Active:     true,
			},
			{
				CardID:     getUintPtr(teacherCard.ID),
				RoomID:     researchLab.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(0, -6, 0),
				ValidUntil: getTimePtr(now.AddDate(0, 6, 0)),
				Active:     true,
			},
			{
				CardID:     getUintPtr(teacherCard.ID),
				RoomID:     lectureHall.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(0, -6, 0),
				ValidUntil: getTimePtr(now.AddDate(0, 6, 0)),
				Active:     true,
			},

			{
				CardID:     getUintPtr(student1Card.ID),
				RoomID:     mainEntrance.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(0, -3, 0),
				ValidUntil: getTimePtr(now.AddDate(0, 0, 10)),
				Active:     true,
			},
			{
				CardID:     getUintPtr(student1Card.ID),
				RoomID:     itLab.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(0, -3, 0),
				ValidUntil: getTimePtr(now.AddDate(0, 0, 10)),
				Active:     true,
			},
			{
				CardID:     getUintPtr(student1Card.ID),
				RoomID:     library.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(0, -3, 0),
				ValidUntil: getTimePtr(now.AddDate(0, 0, 10)),
				Active:     true,
			},
			{
				CardID:     getUintPtr(student1Card.ID),
				RoomID:     lectureHall.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(0, -3, 0),
				ValidUntil: getTimePtr(now.AddDate(0, 0, 10)),
				Active:     true,
			},

			{
				CardID:     getUintPtr(student2Card.ID),
				RoomID:     mainEntrance.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(0, -4, 0),
				ValidUntil: getTimePtr(now.AddDate(0, 8, 0)),
				Active:     true,
			},
			{
				CardID:     getUintPtr(student2Card.ID),
				RoomID:     library.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(0, -4, 0),
				ValidUntil: getTimePtr(now.AddDate(0, 8, 0)),
				Active:     true,
			},
			{
				CardID:     getUintPtr(student2Card.ID),
				RoomID:     lectureHall.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(0, -4, 0),
				ValidUntil: getTimePtr(now.AddDate(0, 8, 0)),
				Active:     true,
			},

			{
				CardID:     getUintPtr(guestCard.ID),
				RoomID:     mainEntrance.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(0, 0, -10),
				ValidUntil: getTimePtr(now.AddDate(0, 0, 5)),
				Active:     true,
			},
			{
				CardID:     getUintPtr(guestCard.ID),
				RoomID:     library.ID,
				GrantedBy:  admin.ID,
				ValidFrom:  now.AddDate(0, 0, -10),
				ValidUntil: getTimePtr(now.AddDate(0, 0, 5)),
				Active:     true,
			},
		}

		for _, permission := range permissions {
			if err := db.Create(&permission).Error; err != nil {
				return err
			}
		}
		log.Println("Hozzáférési jogosultságok létrehozva")

		logs := []models.Log{
			{
				CardID:       adminCard.ID,
				RoomID:       mainEntrance.ID,
				Timestamp:    now.Add(-time.Hour * 48),
				AccessResult: "granted",
			},
			{
				CardID:       adminCard.ID,
				RoomID:       serverRoom.ID,
				Timestamp:    now.Add(-time.Hour * 47),
				AccessResult: "granted",
			},
			{
				CardID:       teacherCard.ID,
				RoomID:       mainEntrance.ID,
				Timestamp:    now.Add(-time.Hour * 30),
				AccessResult: "granted",
			},
			{
				CardID:       teacherCard.ID,
				RoomID:       lectureHall.ID,
				Timestamp:    now.Add(-time.Hour * 29),
				AccessResult: "granted",
			},
			{
				CardID:       student1Card.ID,
				RoomID:       mainEntrance.ID,
				Timestamp:    now.Add(-time.Hour * 26),
				AccessResult: "granted",
			},
			{
				CardID:       student1Card.ID,
				RoomID:       itLab.ID,
				Timestamp:    now.Add(-time.Hour * 25),
				AccessResult: "granted",
			},
			{
				CardID:       student2Card.ID,
				RoomID:       mainEntrance.ID,
				Timestamp:    now.Add(-time.Hour * 23),
				AccessResult: "granted",
			},
			{
				CardID:       student2Card.ID,
				RoomID:       library.ID,
				Timestamp:    now.Add(-time.Hour * 22),
				AccessResult: "granted",
			},

			{
				CardID:       adminCard.ID,
				RoomID:       mainEntrance.ID,
				Timestamp:    now.Add(-time.Hour * 5),
				AccessResult: "granted",
			},
			{
				CardID:       teacherCard.ID,
				RoomID:       mainEntrance.ID,
				Timestamp:    now.Add(-time.Hour * 4),
				AccessResult: "granted",
			},
			{
				CardID:       student1Card.ID,
				RoomID:       mainEntrance.ID,
				Timestamp:    now.Add(-time.Hour * 3),
				AccessResult: "granted",
			},
			{
				CardID:       guestCard.ID,
				RoomID:       mainEntrance.ID,
				Timestamp:    now.Add(-time.Hour * 2),
				AccessResult: "granted",
			},
			{
				CardID:       guestCard.ID,
				RoomID:       library.ID,
				Timestamp:    now.Add(-time.Hour * 1),
				AccessResult: "granted",
			},

			{
				CardID:       student1Card.ID,
				RoomID:       serverRoom.ID,
				Timestamp:    now.Add(-time.Hour * 28),
				AccessResult: "denied",
				DenialReason: "no_permission",
			},
			{
				CardID:       student2Card.ID,
				RoomID:       itLab.ID,
				Timestamp:    now.Add(-time.Hour * 24),
				AccessResult: "denied",
				DenialReason: "no_permission",
			},
			{
				CardID:       guestCard.ID,
				RoomID:       itLab.ID,
				Timestamp:    now.Add(-time.Hour * 6),
				AccessResult: "denied",
				DenialReason: "no_permission",
			},
			{
				CardID:       guestCard.ID,
				RoomID:       researchLab.ID,
				Timestamp:    now.Add(-time.Minute * 30),
				AccessResult: "denied",
				DenialReason: "no_permission",
			},
		}

		for _, log := range logs {
			if err := db.Create(&log).Error; err != nil {
				return err
			}
		}
		log.Println("Demo naplóbejegyzések létrehozva")
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func setupRouter(db *gorm.DB, config *config.Config) *gin.Engine {
	router := routes.SetupRouter(db, config)
	return router
}