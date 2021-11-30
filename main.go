package main

import (
	"log"

	"github.com/gofiber/fiber/v2"

	jwtware "github.com/gofiber/jwt/v3"
	"github.com/golang-jwt/jwt/v4"
)

const jwksURL = "http://127.0.0.1:5556/dex/keys"

func checkUser(c *fiber.Ctx) error {
	user := c.Locals("user").(*jwt.Token)
	claims := user.Claims.(jwt.MapClaims)
	authUserID := claims["sub"].(string)
	if authUserID != c.Params("userID") {
		return c.SendString("Shoot")
	}
	return c.Next()
}

func getUser(c *fiber.Ctx) error {
	c.SendString("Good work, " + c.Params("userID"))
	return nil
}

func main() {
	app := fiber.New()

	jwtMiddle := jwtware.New(jwtware.Config{KeySetURL: jwksURL})

	v1 := app.Group("/v1/users/:userID", jwtMiddle, checkUser)
	v1.Get("/", getUser)

	log.Fatal(app.Listen(":3000"))
}
