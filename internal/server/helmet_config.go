package server

import "github.com/gofiber/fiber/v3/middleware/helmet"

func newHelmetConfig() helmet.Config {
	cfg := helmet.ConfigDefault
	cfg.CrossOriginEmbedderPolicy = "unsafe-none"
	cfg.CrossOriginResourcePolicy = "cross-origin"
	return cfg
}
