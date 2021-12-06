package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/smtp"
	"os"

	"github.com/DIMO-INC/users-api/models"
	"github.com/gofiber/fiber/v2"
	jwtware "github.com/gofiber/jwt/v3"
	"github.com/golang-jwt/jwt/v4"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		DBName   string `yaml:"dbname"`
	} `yaml:"database"`
	Email struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		From     string `yaml:"from"`
	} `yaml:"email"`
}

const jwksURL = "http://127.0.0.1:5556/dex/keys"

type User struct {
	Id    string `json:"id"`
	Email string `json:"email"`
}

func getOrCreateUser(userID string, ctx context.Context) (user *models.User, err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	user, err = models.FindUser(ctx, tx, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			var newUser models.User
			newUser.ID = userID
			err = newUser.Insert(ctx, tx, boil.Infer())
			if err != nil {
				tx.Rollback()
				return
			}
			tx.Commit()
			return &newUser, nil
		}
		tx.Rollback()
		return
	}
	tx.Commit()
	return
}

func getUserHandler(c *fiber.Ctx) error {
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := claims["sub"].(string)

	user, err := getOrCreateUser(userID, c.Context())
	if err != nil {
		panic(err)
	}
	return c.JSON(user)
}

func updateUserHandler(c *fiber.Ctx) error {
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := claims["sub"].(string)

	user, err := getOrCreateUser(userID, c.Context())
	if err != nil {
		return err
	}

	var body User
	c.BodyParser(&body)
	user.Email = null.StringFrom(body.Email)
	user.Update(c.Context(), db, boil.Infer())

	return c.JSON(user)
}

func sendEmailHandler(c *fiber.Ctx) error {
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := claims["sub"].(string)

	user, err := getOrCreateUser(userID, c.Context())
	auth := smtp.PlainAuth("", config.Email.Username, config.Email.Password, config.Email.From)
	addr := fmt.Sprintf("%s:%d", config.Email.Host, config.Email.Port)
	err = smtp.SendMail(addr, auth, config.Email.From, []string{user.Email.String}, []byte{})
	if err != nil {
		return err
	}
	return nil
}

var db *sql.DB
var config Config

func main() {
	var err error
	file, err := os.Open("config.yml")
	if err != nil {
		panic(err)
	}
	if err := yaml.NewDecoder(file).Decode(&config); err != nil {
		panic(err)
	}
	defer file.Close()
	dbConfig := config.Database
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.DBName)
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	command := ""
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	switch command {
	case "migrate":
		goose.Run("up", db, "migrations")
	default:
		app := fiber.New()

		v1 := app.Group("/v1/user", jwtware.New(jwtware.Config{KeySetURL: jwksURL}))
		v1.Get("/", getUserHandler)
		v1.Put("/", updateUserHandler)
		v1.Post("/send-email", sendEmailHandler)

		log.Fatal(app.Listen(":3000"))
	}

}
