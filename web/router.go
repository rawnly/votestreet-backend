package router

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/rawnly/votestreet/internal/database"
	"github.com/rawnly/votestreet/internal/storage"
	utils "github.com/rawnly/votestreet/internal/util"
	"github.com/rawnly/votestreet/pkg/authenticator"
	"github.com/rs/zerolog/log"
)

type (
	OauthProvider string
	GoogleUser    struct {
		ID        string `json:"sub"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		Verified  bool   `json:"email_verified"`
		LastName  string `json:"family_name"`
		FirstName string `json:"given_name"`
	}
)

const (
	ProviderGoogle OauthProvider = "google"
)

func authMiddleware(store *session.Store) fiber.Handler {
	return func(c *fiber.Ctx) error {
		session, err := store.Get(c)
		if err != nil {
			return err
		}

		oauthID, ok := session.Get("user_id").(string)
		if !ok {
			return c.SendStatus(fiber.StatusUnauthorized)
		}

		user, err := database.GetUserByOAuthID(c.Context(), oauthID)
		if err != nil {
			return err
		}

		c.Locals("user", user)

		return c.Next()
	}
}

func Init(app *fiber.App) error {
	sessionStore := session.New(session.Config{
		Storage: storage.Redis(2),
	})

	app.Route("/api", func(router fiber.Router) {
		router.Route("/v1/polls/:id", func(poll fiber.Router) {
			poll.Get("/", func(c *fiber.Ctx) error {
				id, err := strconv.Atoi(c.Params("id"))
				if err != nil {
					return err
				}

				poll, err := database.GetPollByID(c.Context(), int64(id))
				if err != nil {
					return err
				}

				poll.AuthorEmail = nil

				return c.JSON(poll)
			})

			poll.Post("/vote", func(c *fiber.Ctx) error {
				id, err := strconv.Atoi(c.Params("id"))
				if err != nil {
					return err
				}

				poll, err := database.GetPollByID(c.Context(), int64(id))
				if err != nil {
					return err
				}

				var userID string
				if s, err := sessionStore.Get(c); err != nil || s == nil {
					userID = hash(c.IP())
				} else {
					session, _ := sessionStore.Get(c)

					if session.Get("user_id") == nil {
						userID = hash(c.IP())
					} else {
						userID = session.Get("user_id").(string)
					}
				}

				var payload fiber.Map
				if err := c.BodyParser(&payload); err != nil {
					return err
				}

				if _, err := database.InsertVote(c.Context(), database.Vote{
					PollID: poll.ID,
					Value:  payload["value"].(string),
					UserID: userID,
				}); err != nil {
					return err
				}

				return c.SendStatus(fiber.StatusAccepted)
			})
		})

		router.Use(authMiddleware(sessionStore))

		router.Get("/v1/users/me", authMiddleware(sessionStore), func(c *fiber.Ctx) error {
			db := database.Get()
			if db == nil {
				return fiber.NewError(500, "Database not connected")
			}
			session, err := sessionStore.Get(c)
			if err != nil {
				return err
			}

			userID, ok := session.Get("user_id").(string)
			if !ok {
				return c.SendStatus(fiber.StatusUnauthorized)
			}

			user, err := database.GetUserByOAuthID(c.Context(), userID)
			if err != nil {
				log.Error().Err(err).Msg("Failed to get user")
				return fiber.ErrInternalServerError
			}

			return c.JSON(user)
		})

		router.Route("/v1/polls", func(polls fiber.Router) {
			polls.Get("/", func(c *fiber.Ctx) error {
				user := c.Locals("user").(*database.User)

				rows, err := database.GetPollsByUserID(c.Context(), user.ID)
				if err != nil {
					return err
				}

				return c.JSON(rows)
			})

			polls.Post("/", func(c *fiber.Ctx) error {
				user := c.Locals("user").(*database.User)

				var payload fiber.Map
				if err := c.BodyParser(&payload); err != nil {
					return err
				}

				log.Info().Interface("payload", payload).Send()
				description := payload["description"].(string)

				pollID, err := database.InsertPoll(c.Context(), database.Poll{
					Title:       payload["title"].(string),
					UserID:      &user.ID,
					AuthorEmail: &user.Email,
					Ticker:      payload["ticker"].(string),
					Description: &description,
				})
				if err != nil {
					return err
				}

				return c.Status(fiber.StatusCreated).JSON(fiber.Map{
					"inserted": pollID,
				})
			})

			polls.Delete("/:id", func(c *fiber.Ctx) error {
				user := c.Locals("user").(*database.User)

				pollID, err := strconv.Atoi(c.Params("id"))
				if err != nil {
					return err
				}

				if _, err := database.DeletePollByIDAndUserID(c.Context(), pollID, user.ID); err != nil {
					return err
				}

				return c.SendStatus(fiber.StatusAccepted)
			})
		})
	})

	app.All("/logout", func(c *fiber.Ctx) error {
		session, err := sessionStore.Get(c)
		if err != nil {
			return err
		}

		if err := session.Destroy(); err != nil {
			return err
		}

		return c.SendStatus(fiber.StatusAccepted)
	})

	app.Route("/oauth", func(router fiber.Router) {
		router.Get("/:provider/login", func(c *fiber.Ctx) error {
			session, err := sessionStore.Get(c)
			if err != nil {
				return err
			}

			state := utils.RandomStringPrefixed("google_", 7)
			session.Set("state", state)
			if err := session.Save(); err != nil {
				return err
			}

			switch c.Params("provider") {
			case string(ProviderGoogle):
				google, err := authenticator.GetGoogleAuthenticator()
				if err != nil {
					return err
				}

				url := google.AuthCodeURL(state)

				accept := c.Get(fiber.HeaderAccept)
				log.Info().Str("accept", accept).Send()

				if acceptsHTML(c) && c.Query("skip") == "" {
					return c.Render("oauth", fiber.Map{
						"Url": url,
					})
				}

				if acceptsJSON(c) {
					return c.JSON(fiber.Map{
						"url": url,
					})
				}

				return c.Status(fiber.StatusSeeOther).Redirect(url)
			default:
				return c.Render("login", fiber.Map{
					"Providers": []string{
						string(ProviderGoogle),
					},
				})
			}
		})

		router.Get("/:provider/callback", func(c *fiber.Ctx) error {
			session, err := sessionStore.Get(c)
			if err != nil {
				return err
			}

			if state, ok := session.Get("state").(string); ok {
				if !safeEqual(state, c.Query("state")) {
					log.Error().Str("state", state).Str("query_state", c.Query("state")).Msg("invalid state")
					return fiber.ErrForbidden
				}

				session.Delete("state")
			} else {
				return fiber.ErrForbidden
			}

			switch c.Params("provider") {
			case string(ProviderGoogle):
				google, err := authenticator.GetGoogleAuthenticator()
				if err != nil {
					return err
				}

				code := c.Query("code")
				token, err := google.Exchange(
					c.UserContext(),
					code,
				)
				if err != nil {
					return err
				}

				idToken, err := google.VerifyIDToken(c.Context(), token)
				if err != nil {
					return err
				}

				var user GoogleUser
				if err = idToken.Claims(&user); err != nil {
					return err
				}

				session.Set("user_id", user.ID)
				duration := time.Until(idToken.Expiry)
				session.SetExpiry(duration)

				if err := session.Save(); err != nil {
					return err
				}

				return c.SendStatus(fiber.StatusOK)
			default:
				return fiber.ErrNotFound
			}
		})
	})

	return nil
}

func safeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func acceptsJSON(c *fiber.Ctx) bool {
	return strings.Contains(c.Get(fiber.HeaderAccept), "application/json")
}

func acceptsHTML(c *fiber.Ctx) bool {
	return strings.Contains(c.Get(fiber.HeaderAccept), "text/html")
}

func hash(s string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
}
