package routes

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"rfid/internal/config"
	"rfid/internal/handlers"
	"rfid/internal/middleware"
	"rfid/internal/websocket"
)

func SetupRouter(db *gorm.DB, config *config.Config) *gin.Engine {
	router := gin.Default()

	authHandler := handlers.NewAuthHandler(db)
	userHandler := handlers.NewUserHandler(db)
	cardHandler := handlers.NewCardHandler(db)
	roomHandler := handlers.NewRoomHandler(db)
	permissionHandler := handlers.NewPermissionHandler(db)
	logHandler := handlers.NewLogHandler(db)
	simulationHandler := handlers.NewSimulationHandler(db)
	groupHandler := handlers.NewGroupHandler(db)

	var wsHandler *websocket.WebSocketHandler
	if config.EnableWebsocket {
		wsHandler = websocket.NewWebSocketHandler(db)

		cardHandler.SetWebSocketHandler(wsHandler)
	}

	authMiddleware := middleware.NewAuthMiddleware(db)
	apiKeyMiddleware := middleware.NewAPIKeyMiddleware(db, config)

	router.Use(func(c *gin.Context) {
		c.Set("config", config)
		c.Next()
	})

	router.Static("/static", "./web/static")
	router.LoadHTMLGlob("./web/templates/*")

	router.GET("/", func(c *gin.Context) {
		c.HTML(200, "index.html", gin.H{
			"title": "RFID Card Management System",
		})
	})

	if config.EnableWebsocket {
		router.GET("/ws", wsHandler.HandleWebSocket)
	}

	auth := router.Group("/api/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/register", authMiddleware.AuthRequired(), authMiddleware.AdminRequired(), authHandler.Register)
		auth.GET("/me", authMiddleware.AuthRequired(), authHandler.GetMe)
		auth.POST("/change-password", authMiddleware.AuthRequired(), authHandler.ChangePassword)
	}

	if config.EnableRESTAPI {
		api := router.Group("/api")

		if config.APIKeyRequired {
			api.Use(apiKeyMiddleware.APIKeyRequired())
		}

		api.Use(authMiddleware.AuthRequired())
		{
			users := api.Group("/users")
			users.Use(authMiddleware.AdminRequired())
			{
				users.GET("", userHandler.GetUsers)
				users.GET("/:id", userHandler.GetUser)
				users.POST("", userHandler.CreateUser)
				users.PUT("/:id", userHandler.UpdateUser)
				users.DELETE("/:id", userHandler.DeleteUser)
				users.GET("/:id/cards", userHandler.GetUserCards)
			}

			cards := api.Group("/cards")
			cards.Use(authMiddleware.AdminRequired())
			{
				cards.GET("", cardHandler.GetCards)
				cards.GET("/:id", cardHandler.GetCard)
				cards.POST("", cardHandler.CreateCard)
				cards.PUT("/:id", cardHandler.UpdateCard)
				cards.DELETE("/:id", cardHandler.DeleteCard)
				cards.POST("/:id/block", cardHandler.BlockCard)
				cards.POST("/:id/unblock", cardHandler.UnblockCard)
				cards.POST("/:id/revoke", cardHandler.RevokeCard)
				cards.GET("/expiring", cardHandler.GetExpiringCards)
			}

			rooms := api.Group("/rooms")
			rooms.Use(authMiddleware.AdminRequired())
			{
				rooms.GET("", roomHandler.GetRooms)
				rooms.GET("/:id", roomHandler.GetRoom)
				rooms.POST("", roomHandler.CreateRoom)
				rooms.PUT("/:id", roomHandler.UpdateRoom)
				rooms.DELETE("/:id", roomHandler.DeleteRoom)
				rooms.GET("/:id/permissions", roomHandler.GetRoomPermissions)
				rooms.GET("/:id/logs", roomHandler.GetRoomLogs)
			}

			permissions := api.Group("/permissions")
			permissions.Use(authMiddleware.AdminRequired())
			{
				permissions.GET("", permissionHandler.GetPermissions)
				permissions.GET("/:id", permissionHandler.GetPermission)
				permissions.POST("", permissionHandler.CreatePermission)
				permissions.PUT("/:id", permissionHandler.UpdatePermission)
				permissions.DELETE("/:id", permissionHandler.DeletePermission)
				permissions.POST("/:id/revoke", permissionHandler.RevokePermission)
			}

			logs := api.Group("/logs")
			logs.Use(authMiddleware.AdminRequired())
			{
				logs.GET("", logHandler.GetLogs)
				logs.GET("/:id", logHandler.GetLog)
				logs.POST("", logHandler.CreateLog)

				logs.GET("/stats/rooms", logHandler.GetRoomStats)
				logs.GET("/stats/cards", logHandler.GetCardStats)
				logs.GET("/stats/time-series", logHandler.GetAccessTimeSeries)
				logs.GET("/stats/most-accessed-rooms", logHandler.GetMostAccessedRooms)
				logs.GET("/stats/most-active-users", logHandler.GetMostActiveUsers)
			}

			groups := api.Group("/groups")
			groups.Use(authMiddleware.AdminRequired())
			{
				groups.GET("", groupHandler.GetGroups)
				groups.GET("/:id", groupHandler.GetGroup)
				groups.POST("", groupHandler.CreateGroup)
				groups.PUT("/:id", groupHandler.UpdateGroup)
				groups.DELETE("/:id", groupHandler.DeleteGroup)

				groups.GET("/:id/users", groupHandler.GetGroupUsers)
				groups.POST("/:id/users", groupHandler.AddUserToGroup)
				groups.DELETE("/:id/users/:user_id", groupHandler.RemoveUserFromGroup)

				groups.GET("/:id/rooms", groupHandler.GetGroupRooms)
				groups.POST("/:id/rooms", groupHandler.AddRoomToGroup)
				groups.DELETE("/:id/rooms/:room_id", groupHandler.RemoveRoomFromGroup)
			}

			api.POST("/check-access", cardHandler.CheckAccess)

			simulation := api.Group("/simulate")
			simulation.Use(authMiddleware.AdminRequired())
			{
				simulation.POST("/access", simulationHandler.SimulateAccess)
			}
		}
	}

	cardReader := router.Group("/reader")

	if config.APIKeyRequired {
		cardReader.Use(apiKeyMiddleware.APIKeyRequired())
	}

	{
		cardReader.POST("/check-access", cardHandler.CheckAccess)
	}

	return router
}