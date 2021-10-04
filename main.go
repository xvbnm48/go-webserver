package main

import (
	"go-webserver/data"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	jwtware "github.com/gofiber/jwt/v3"
	"github.com/golang-jwt/jwt/v4"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type SignupRequest struct {
	Name     string
	Email    string
	Password string `json:"*"`
}

type loginRequest struct {
	Email    string
	Password string `json:"*"`
}

func main() {
	app := fiber.New()
	engine, err := data.CreateDBEngine()
	if err != nil {
		panic(err)
	}

	app.Post("/signup", func(c *fiber.Ctx) error {
		req := new(SignupRequest)
		if err := c.BodyParser(req); err != nil {
			return err
		}

		if req.Email == "" || req.Name == "" || req.Password == "" {
			return fiber.NewError(fiber.StatusBadRequest, "invalid signup credential")
		}

		// save this info in the database
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}

		user := &data.User{
			Name:     req.Name,
			Email:    req.Email,
			Password: string(hash),
		}

		_, err = engine.Insert(user)
		if err != nil {
			return err
		}
		token, exp, err := createJWTToken(*user)
		if err != nil {
			return err
		}

		// create jwt token

		return c.JSON(fiber.Map{"token": token, "exp": exp, "user": user})
	})

	app.Post("/login", func(c *fiber.Ctx) error {
		req := new(loginRequest)
		if err := c.BodyParser(req); err != nil {
			return err
		}

		if req.Email == "" || req.Password == "" {
			return fiber.NewError(fiber.StatusBadRequest, "invalid signup credential")
		}
		user := new(data.User)
		has, err := engine.Where("email = ?", req.Email).Desc("id").Get(user)
		if err != nil {
			return err
		}
		if !has {
			return fiber.NewError(fiber.StatusBadRequest, "invalid login")
		}
		token, exp, err := createJWTToken(*user)
		if err != nil {
			return err
		}

		// create jwt token

		return c.JSON(fiber.Map{"token": token, "exp": exp, "user": user})
	})
	
	private := app.Group("/private")
	private.Use(jwtware.New(jwtware.Config{
		SigningKey: []byte("secret"),
	}))
	private.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true, "path": "private"})
	})

	app.Get("/public", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"success": true, "path": "public"})
	})

	log.Fatal(app.Listen(":8080"))
}

func createJWTToken(user data.User) (string, int64, error) {
	exp := time.Now().Add(time.Minute * 30).Unix()

	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["user_id"] = user.Id
	claims["exp"] = exp
	t, err := token.SignedString([]byte("secret"))

	if err != nil {
		return "", 0, err
	}

	return t, exp, nil
}
