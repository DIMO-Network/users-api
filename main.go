package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/BurntSushi/toml"
	"github.com/gofiber/fiber/v2"
	jwtware "github.com/gofiber/jwt/v3"
	"github.com/golang-jwt/jwt/v4"
	_ "github.com/lib/pq"
)

type Config struct {
	Database DBConfig
}

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

const jwksURL = "http://127.0.0.1:5556/dex/keys"

type User struct {
	Id    string `json:"id"`
	Email string `json:"email"`
}

func getOrCreateUser(userID string) User {
	var email string
	if err := db.QueryRow(`SELECT email FROM users WHERE "id" = $1;`, userID).Scan(&email); err != nil {
		fmt.Println("Got here")
		// Race city
		if _, err := db.Exec(`INSERT INTO users ("id") VALUES ($1);`, userID); err != nil {
			panic(err)
		}
	}
	fmt.Println("Email is", email)
	return User{Id: userID, Email: email}
}

func getUserHandler(c *fiber.Ctx) error {
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := claims["sub"].(string)
	fmt.Println(userID)
	user := getOrCreateUser(userID)
	return c.JSON(user)
}

func updateUserHandler(c *fiber.Ctx) error {
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := claims["sub"].(string)

	var user User

	c.BodyParser(&user)
	fmt.Println("PUT userID", userID)

	if _, err := db.Exec("UPDATE users SET email = $1 WHERE id = $2", user.Email, userID); err != nil {
		panic(err)
	}

	return c.JSON(user)
}

var db *sql.DB

func main() {
	var err error
	const file = "config.toml"
	var config Config
	if _, err = toml.DecodeFile(file, &config); err != nil {
		panic(err)
	}
	dbConfig := config.Database
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.DBName)
	fmt.Println(connStr)
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	_, err = db.Exec("DROP TABLE IF EXISTS users; CREATE TABLE users (id TEXT PRIMARY KEY, email TEXT);")
	if err != nil {
		panic(err)
	}

	app := fiber.New()

	v1 := app.Group("/v1/user", jwtware.New(jwtware.Config{KeySetURL: jwksURL}))
	v1.Get("/", getUserHandler)
	v1.Put("/", updateUserHandler)

	log.Fatal(app.Listen(":3000"))
}
