package main

import (
	"context"
	"os"
	"time"

	_ "go.uber.org/automaxprocs"

	_ "github.com/DIMO-INC/users-api/docs"
	"github.com/DIMO-INC/users-api/internal/config"
	"github.com/DIMO-INC/users-api/internal/controllers"
	"github.com/DIMO-INC/users-api/internal/database"
	swagger "github.com/arsmn/fiber-swagger/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	jwtware "github.com/gofiber/jwt/v3"
	"github.com/rs/zerolog"
)

// @title DIMO User API
// @version 1.0
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	gitSha1 := os.Getenv("GIT_SHA1")
	ctx := context.Background()
	logger := zerolog.New(os.Stdout).With().
		Timestamp().
		Str("app", "users-api").
		Str("git-sha1", gitSha1).
		Logger()

	settings, err := config.LoadConfig("settings.yaml")
	if err != nil {
		logger.Fatal().Err(err).Msg("could not load settings")
	}
	pdb := database.NewDbConnectionFromSettings(ctx, settings)

	arg := ""
	if len(os.Args) > 1 {
		arg = os.Args[1]
	}
	switch arg {
	case "migrate":
		migrateDatabase(logger, settings)
	default:
		startWebAPI(logger, settings, pdb)
	}
}

func startWebAPI(logger zerolog.Logger, settings *config.Settings, pdb database.DbStore) {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return ErrorHandler(c, err, logger)
		},
		DisableStartupMessage: true,
		ReadBufferSize:        16000,
	})

	app.Use(recover.New(recover.Config{
		Next:              nil,
		EnableStackTrace:  true,
		StackTraceHandler: nil,
	}))
	app.Use(cors.New())
	app.Get("/", HealthCheck)

	app.Get("/swagger/*", swagger.Handler)

	keyRefreshInterval := time.Hour
	keyRefreshUnknownKID := true
	v1 := app.Group("/v1/user", jwtware.New(jwtware.Config{
		KeySetURL: settings.JWTKeySetURL,
		KeyRefreshErrorHandler: func(j *jwtware.KeySet, err error) {
			logger.Error().Err(err).Msg("Key refresh error")
		},
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusUnauthorized).JSON(struct {
				Message string `json:"message"`
			}{"Invalid or expired JWT"})
		},
		KeyRefreshInterval:   &keyRefreshInterval,
		KeyRefreshUnknownKID: &keyRefreshUnknownKID,
	}))

	userController := controllers.NewUserController(settings, pdb.DBS, &logger)
	v1.Get("/", userController.GetUser)
	v1.Put("/", userController.UpdateUser)
	v1.Delete("/", userController.DeleteUser)
	v1.Post("/agree-tos", userController.AgreeTOS)
	v1.Post("/send-confirmation-email", userController.SendConfirmationEmail)
	v1.Post("/confirm-email", userController.ConfirmEmail)

	// Temporary endpoints for the migration from Django
	if len(settings.AdminPassword) >= 8 {
		admin := app.Group("/admin", basicauth.New(basicauth.Config{
			Users: map[string]string{
				"admin": settings.AdminPassword,
			},
		}))
		admin.Post("/create-user", userController.AdminCreateUser)
		admin.Get("/view-users", userController.AdminViewUsers)
		admin.Post("/delete-user/:userID", userController.AdminDeleteUser)
	}

	customerIOController := controllers.NewCustomerIOController(settings, pdb.DBS, &logger)
	v1.Post("/vitamins/known", customerIOController.Track)

	logger.Info().Msg("Server started on port " + settings.Port)

	// Start Server
	if err := app.Listen(":" + settings.Port); err != nil {
		logger.Fatal().Err(err)
	}
}

// HealthCheck godoc
// @Summary Show the status of server.
// @Description get the status of server.
// @Tags root
// @Accept */*
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router / [get]
func HealthCheck(c *fiber.Ctx) error {
	res := map[string]interface{}{
		"data": "Server is up and running",
	}

	if err := c.JSON(res); err != nil {
		return err
	}

	return nil
}

// ErrorHandler custom handler to log recovered errors using our logger and return json instead of string
func ErrorHandler(c *fiber.Ctx, err error, logger zerolog.Logger) error {
	code := fiber.StatusInternalServerError // Default 500 statuscode

	if e, ok := err.(*fiber.Error); ok {
		// Override status code if fiber.Error type
		code = e.Code
	}
	logger.Err(err).Msg("caught a panic")

	return c.Status(code).JSON(fiber.Map{
		"error": true,
		"msg":   err.Error(),
	})
}
