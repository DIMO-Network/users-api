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
	pdb := database.NewDbConnectionFromSettings(ctx, settings, true)
	// check db ready, this is not ideal btw, the db connection handler would be nicer if it did this.
	totalTime := 0
	for !pdb.IsReady() {
		if totalTime > 30 {
			logger.Fatal().Msg("could not connect to postgres after 30 seconds")
		}
		time.Sleep(time.Second)
		totalTime++
	}

	arg := ""
	if len(os.Args) > 1 {
		arg = os.Args[1]
	}
	switch arg {
	case "migrate":
		migrateDatabase(logger, settings)
	case "generate-events":
		eventService := services.NewEventService(&logger, settings)
		generateEvents(&logger, settings, pdb, eventService)
	case "generate-referrals":
		eventService := services.NewEventService(&logger, settings)
		generateReferrals(&logger, settings, pdb, eventService)
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
		GroupID:         "users-api",
		MaxInFlight:     int64(5),
	}
	consumer, err := kafka.NewConsumer(cfg, &logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("could not start consumer")
	}
	eventReader := services.NewEventReader(pdb.DBS, &logger, eventService)
	consumer.Start(context.Background(), eventReader.ProcessDeviceStatusMessages)

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

	app.Get("/v1/swagger/*", swagger.HandlerDefault)

	keyRefreshInterval := time.Hour
	keyRefreshUnknownKID := true
	v1User := app.Group("/v1/user", jwtware.New(jwtware.Config{
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
	v1User.Get("/", userController.GetUser)
	v1User.Put("/", userController.UpdateUser)
	v1User.Delete("/", userController.DeleteUser)
	v1User.Post("/agree-tos", userController.AgreeTOS)
	v1User.Post("/send-confirmation-email", userController.SendConfirmationEmail)
	v1User.Post("/confirm-email", userController.ConfirmEmail)
	v1User.Post("/web3/challenge/generate", userController.GenerateEthereumChallenge)
	v1User.Post("/web3/challenge/submit", userController.SubmitEthereumChallenge)

	customerIOController := controllers.NewCustomerIOController(settings, pdb.DBS, &logger)
	v1User.Post("/vitamins/known", customerIOController.Track)

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
	message := "Internal error."

	if e, ok := err.(*fiber.Error); ok {
		// Override status code if fiber.Error type
		code = e.Code
		message = err.Error()
	}

	return c.Status(code).JSON(fiber.Map{
		"code":    code,
		"message": message,
	})
}
