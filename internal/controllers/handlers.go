package controllers

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

type ErrorResponse struct {
	ErrorMessage string `json:"errorMessage"`
}

func errorResponseHandler(c *fiber.Ctx, err error, status int) error {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	return c.Status(status).JSON(ErrorResponse{msg})
}

func getUserID(c *fiber.Ctx) string {
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	userID := claims["sub"].(string)
	return userID
}

func getUserEthAddr(c *fiber.Ctx) *common.Address {
	token := c.Locals("user").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)

	val, exists := claims["ethereum_address"]
	if exists {
		ethAddr := common.HexToAddress(val.(string))
		return &ethAddr
	}
	return nil
}
