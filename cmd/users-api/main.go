package main

import (
	"context"
	"net"
	"os"
	"time"
	"strings"

	_ "github.com/lib/pq"
	_ "go.uber.org/automaxprocs"
	"google.golang.org/grpc"

	"github.com/DIMO-Network/shared"
	pb "github.com/DIMO-Network/shared/api/users"
	"github.com/DIMO-Network/shared/db"
	_ "github.com/DIMO-Network/users-api/docs"
	"github.com/DIMO-Network/users-api/internal/api"
	"github.com/DIMO-Network/users-api/internal/config"
	"github.com/DIMO-Network/users-api/internal/controllers"
	"github.com/DIMO-Network/users-api/internal/database"
	"github.com/DIMO-Network/users-api/internal/services"
	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	jwtware "github.com/gofiber/jwt/v3"
	"github.com/gofiber/swagger"
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

	settings, err := shared.LoadConfig[config.Settings]("settings.yaml")
	if err != nil {
		logger.Fatal().Err(err).Msg("could not load settings")
	}

	dbs := db.NewDbConnectionFromSettings(ctx, &settings.DB, true)
	dbs.WaitForDB(logger)

	arg := ""
	if len(os.Args) > 1 {
		arg = os.Args[1]
	}
	switch arg {
	case "migrate":
		command := "up"
		if len(os.Args) > 2 {
			command = os.Args[2]
			if command == "down-to" || command == "up-to" {
				command = command + " " + os.Args[3]
			}
		}
		if err := database.MigrateDatabase(logger, &settings.DB, command, "migrations"); err != nil {
			logger.Fatal().Err(err).Msg("Failed to migrate datbase.")
		}
	case "generate-events":
		eventService := services.NewEventService(&logger, &settings)
		generateEvents(&logger, &settings, dbs, eventService)
	case "generate-referral-codes":
		grc := &generateReferralCodeCmd{
			dbs:      dbs,
			log:      &logger,
			Settings: &settings,
		}

		if err := grc.Execute(ctx); err != nil {
			logger.Fatal().Err(err).Msg("Error during referral code generation.")
		}
	default:
		eventService := services.NewEventService(&logger, &settings)
		startWebAPI(logger, &settings, dbs, eventService)
	}
}

func startWebAPI(logger zerolog.Logger, settings *config.Settings, dbs db.Store, eventService services.EventService) {
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

	go func() {
		monApp := fiber.New(fiber.Config{DisableStartupMessage: true})

		prometheus.RegisterAt(monApp, "/metrics")

		if err := monApp.Listen(":" + settings.MonitoringPort); err != nil {
			logger.Fatal().Err(err).Str("port", settings.MonitoringPort).Msg("Failed to start monitoring web server.")
		}
	}()

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

	userController := controllers.NewUserController(settings, dbs, eventService, &logger)
	v1User.Get("/", userController.GetUser)
	v1User.Put("/", userController.UpdateUser)
	v1User.Delete("/", userController.DeleteUser)
	v1User.Get("/check-accounts", userController.CheckAccount)
	v1User.Post("/agree-tos", userController.AgreeTOS)
	v1User.Post("/send-confirmation-email", userController.SendConfirmationEmail)
	v1User.Post("/confirm-email", userController.ConfirmEmail)
	if settings.Environment != "prod" {
		v1User.Post("/submit-referral-code", userController.SubmitReferralCode)
	}
	v1User.Post("/web3/challenge/generate", userController.GenerateEthereumChallenge)
	v1User.Post("/web3/challenge/submit", userController.SubmitEthereumChallenge)

	logger.Info().Msg("Server started on port " + settings.Port)

	go startGRPCServer(settings, dbs, &logger)

	// Start Server
	if err := app.Listen(":" + settings.Port); err != nil {
		logger.Fatal().Err(err)
	}
}

func startGRPCServer(settings *config.Settings, dbs db.Store, logger *zerolog.Logger) {
	lis, err := net.Listen("tcp", ":"+settings.GRPCPort)
	if err != nil {
		logger.Fatal().Err(err).Msgf("Couldn't listen on gRPC port %s", settings.GRPCPort)
	}

	logger.Info().Msgf("Starting gRPC server on port %s", settings.GRPCPort)
	server := grpc.NewServer()
	pb.RegisterUserServiceServer(server, api.NewUserService(dbs, logger))

	if err := server.Serve(lis); err != nil {
		logger.Fatal().Err(err).Msg("gRPC server terminated unexpectedly")
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
	err := c.JSON(res)
	if err != nil {
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
	
	logger.Err(err).Int("code", code).Str("path", strings.TrimPrefix(c.Path(), "/")).Msg("Failed request.")

	return c.Status(code).JSON(fiber.Map{
		"code":    code,
		"message": message,
	})
}
