package main

import (
	"context"
	"os"
	"strings"
	"time"

	_ "go.uber.org/automaxprocs"

	_ "github.com/DIMO-INC/users-api/docs"
	"github.com/DIMO-INC/users-api/internal/config"
	"github.com/DIMO-INC/users-api/internal/controllers"
	"github.com/DIMO-INC/users-api/internal/database"
	"github.com/DIMO-INC/users-api/internal/services"
	"github.com/DIMO-INC/users-api/internal/services/kafka"
	"github.com/Shopify/sarama"
	"github.com/ansrivas/fiberprometheus/v2"
	swagger "github.com/arsmn/fiber-swagger/v2"
	"github.com/gofiber/fiber/v2"
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
		eventService := services.NewEventService(&logger, settings)
		startEventConsumer(logger, settings, pdb, eventService)
		startWebAPI(logger, settings, pdb, eventService)
	}
}

func startEventConsumer(logger zerolog.Logger, settings *config.Settings, pdb database.DbStore, eventService *services.EventService) {
	clusterConfig := sarama.NewConfig()
	clusterConfig.Version = sarama.V2_6_0_0
	clusterConfig.Consumer.Offsets.Initial = sarama.OffsetNewest

	cfg := &kafka.Config{
		ClusterConfig:   clusterConfig,
		BrokerAddresses: strings.Split(settings.KafkaBrokers, ","),
		Topic:           settings.EventsTopic,
		GroupID:         "user-devices",
		MaxInFlight:     int64(5),
	}
	_, err := kafka.NewConsumer(cfg, &logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not start consumer")
	}
	services.NewEventReader(pdb.DBS, &logger, eventService)

	logger.Info().Msg("kafka consumer started")
}

func startWebAPI(logger zerolog.Logger, settings *config.Settings, pdb database.DbStore, eventService *services.EventService) {
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

	prometheus := fiberprometheus.New("users-api")
	prometheus.RegisterAt(app, "/metrics")
	app.Use(prometheus.Middleware)

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

	userController := controllers.NewUserController(settings, pdb.DBS, eventService, &logger)
	v1.Get("/", userController.GetUser)
	v1.Put("/", userController.UpdateUser)
	v1.Delete("/", userController.DeleteUser)
	v1.Post("/agree-tos", userController.AgreeTOS)
	v1.Post("/send-confirmation-email", userController.SendConfirmationEmail)
	v1.Post("/confirm-email", userController.ConfirmEmail)

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
