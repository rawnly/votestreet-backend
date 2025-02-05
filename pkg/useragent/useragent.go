package useragent

import (
	"github.com/gofiber/fiber/v2"
	userAgent "github.com/mileusna/useragent"
)

type UserAgent struct {
	userAgent.UserAgent
}

func (ua *UserAgent) IsStripe() bool {
	return ua.String == "Stripe/1.0 (+https://stripe.com/docs/webhooks)"
}

func (ua *UserAgent) CanSkipChecks() bool {
	return ua.IsStripe()
}

func FromCtx(c *fiber.Ctx) *UserAgent {
	return &UserAgent{userAgent.Parse(c.Get(fiber.HeaderUserAgent))}
}

type Config struct {
	Next func(ua *UserAgent) bool
}

func New(config Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ua := FromCtx(c)

		if config.Next != nil && config.Next(ua) {
			return c.Next()
		}

		if ua.Bot {
			return c.Status(fiber.StatusNotFound).Send(nil)
		}

		return c.Next()
	}
}
