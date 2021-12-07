package main

// func updateUserHandler(c *fiber.Ctx) error {
// 	token := c.Locals("user").(*jwt.Token)
// 	claims := token.Claims.(jwt.MapClaims)
// 	userID := claims["sub"].(string)

// 	user, err := getOrCreateUser(userID, c.Context())
// 	if err != nil {
// 		return err
// 	}

// 	var body UserResponse
// 	c.BodyParser(&body)
// 	user.Email = body.Email
// 	user.Update(c.Context(), db, boil.Infer())

// 	return c.JSON(user)
// }

// func sendEmailHandler(c *fiber.Ctx) error {
// 	token := c.Locals("user").(*jwt.Token)
// 	claims := token.Claims.(jwt.MapClaims)
// 	userID := claims["sub"].(string)

// 	user, err := getOrCreateUser(userID, c.Context())
// 	auth := smtp.PlainAuth("", config.Email.Username, config.Email.Password, config.Email.From)
// 	addr := fmt.Sprintf("%s:%d", config.Email.Host, config.Email.Port)
// 	err = smtp.SendMail(addr, auth, config.Email.From, []string{user.Email.String}, []byte{})
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }
