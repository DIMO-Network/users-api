package main

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"database/sql"

	jwtware "github.com/gofiber/jwt/v3"
	"github.com/golang-jwt/jwt/v4"
	_ "github.com/mattn/go-sqlite3"
)

const jwksURL = "http://127.0.0.1:5556/dex/keys"

type User struct {
	Id    string `json:"id"`
	Email string `json:"email"`
}

func checkUser(c *fiber.Ctx) error {
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	authUserID := claims["sub"].(string)
	if authUserID != c.Params("userID") {
		return c.SendString("Shoot")
	}
	return c.Next()
}

func getOrCreateUser(userID string) User {
	var email string
	if err := db.QueryRow("SELECT email FROM users WHERE id = ?;", userID).Scan(&email); err != nil {
		db.Exec("INSERT INTO users (sub) VALUES (?);", userID)
	}
	return User{Id: userID, Email: email}
}

func getUserHandler(c *fiber.Ctx) error {
	userID := c.Params("userID")
	user := getOrCreateUser(userID)
	return c.JSON(user)
}

func updateUserHandler(c *fiber.Ctx) error {
	userID := c.Params("userID")
	user := getOrCreateUser(userID)
	// Going to pick up dog
}

var db *sql.DB

func main() {
	db, _ = sql.Open("sqlite3", ":memory:")

	defer db.Close()

	db.Exec("CREATE TABLE users (id TEXT PRIMARY KEY, email TEXT);")

	app := fiber.New()

	jwtMiddle := jwtware.New(jwtware.Config{KeySetURL: jwksURL})

	v1 := app.Group("/v1/users/:userID", jwtMiddle, checkUser)
	v1.Get("/", getUserHandler)
	v1.Put("/", updateUserHandler)

	log.Fatal(app.Listen(":3000"))
}
