package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gofiber/contrib/fiberzerolog"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"github.com/gofiber/template/html/v2"
	"github.com/rawnly/votestreet/internal/database"
	"github.com/rawnly/votestreet/internal/storage"
	utils "github.com/rawnly/votestreet/internal/util"
	"github.com/rawnly/votestreet/pkg/useragent/honeypot"
	router "github.com/rawnly/votestreet/web"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	LimiterRedisDB  = 0
	HoneypotRedisDB = 1
)

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// panic 5
	// fatal 4
	// error 3
	// warn 2
	// info 1
	// debug 0
	// trace -1
	level := flag.Int("log-level", 1, "Set log level")
	port := flag.Int("port", 8080, "set http port")
	debug := flag.Bool("debug", false, "set debug mode")
	flag.Parse()

	if err := database.Connect(); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}

	if err := database.CreateTables(); err != nil {
		log.Fatal().Err(err).Msg("Failed to create tables")
	}

	defer database.Close()

	engine := html.New("./views", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Use(
		helmet.New(),
		healthcheck.New(healthcheck.Config{
			ReadinessEndpoint: "/healthz",
			ReadinessProbe: func(c *fiber.Ctx) bool {
				return storage.IsRedisHealthy(c.Context())
			},
			LivenessProbe: func(c *fiber.Ctx) bool {
				return true
			},
		}),
		limiter.New(limiter.Config{
			Storage: storage.Redis(LimiterRedisDB),
			Next: func(c *fiber.Ctx) bool {
				return c.IP() == "127.0.0.1" || *debug
			},
		}),
		fiberzerolog.New(fiberzerolog.Config{
			Logger: &log.Logger,
			Fields: []string{"ip", "ua", "latency", "requestId", "status", "method", "url", "error"},
		}),
		requestid.New(requestid.Config{
			Generator: func() string { return utils.RandomStringPrefixed("req_", 7) },
		}),
		honeypot.New(storage.Redis(HoneypotRedisDB)),
	)

	logLevel := zerolog.Level(*level)
	zerolog.SetGlobalLevel(logLevel)

	if *debug {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out: os.Stderr,
		})
	} else {
		log.Logger = log.With().Caller().Logger()
	}

	if err := router.Init(app); err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize router")
	}

	log.Info().Int("port", *port).Msg("Server started")
	log.Fatal().Err(app.Listen(fmt.Sprintf(":%d", *port))).Msg("Server stopped")
}
