package config

import "strings"

type Robots struct {
	Enable    bool   `koanf:"enable"`
	Override  bool   `koanf:"override"`
	UserAgent string `koanf:"user_agent"`
	Allow     string `koanf:"allow"`
	Disallow  string `koanf:"disallow"`
	Sitemap   string `koanf:"sitemap"`
	Host      string `koanf:"host"`
}

func (r Robots) NormalizedUserAgent() string {
	userAgent := strings.TrimSpace(r.UserAgent)
	if userAgent == "" {
		return "*"
	}
	return userAgent
}
