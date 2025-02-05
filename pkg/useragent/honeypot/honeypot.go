package honeypot

import (
	"sync"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var mu sync.Mutex

type Config struct {
	Storage fiber.Storage
	Paths   []string
}

var scamPaths = []string{
	"/wp-admin",
	"/wp-login",
	"/wp-content",
	"/wp-includes",
	"/.env",
	"/.git",
	"/.github",
	"/.gitignore",
	"/.htaccess",
	"/etc/passwd",
}

func New(storage fiber.Storage) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ip := c.IP()

		if ip == "127.0.0.1" {
			return c.Next()
		}

		mu.Lock()
		defer mu.Unlock()

		data, err := storage.Get(c.IP())
		if err != nil {
			log.Error().Err(err).Msg("Error getting data")
		}

		if data != nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "You are not allowed to access this resource",
			})
		}

		for _, path := range scamPaths {
			if c.Path() == path {
				if err = storage.Set(ip, []byte("1"), 0); err != nil {
					log.Error().Err(err).Msg("Error setting data")

					return c.Status(fiber.StatusOK).JSON(fiber.Map{
						"token": uuid.New(),
					})
				}

				return err
			}
		}

		return c.Next()
	}
}
