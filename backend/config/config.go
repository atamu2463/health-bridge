package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const defaultPort = "8080"

type Config struct {
	Port           string
	DatabaseURL    string
	AllowedOrigins []string
	CookieSecure   bool
	IsRender       bool
}

// HTTPサーバーに必要な環境変数を起動前に検証し、利用可能な設定だけを返す。
func Load() (Config, error) {
	databaseURL, err := LoadDatabaseURL()
	if err != nil {
		return Config{}, err
	}

	allowedOriginsValue, err := requiredEnvironmentVariable("ALLOWED_ORIGINS")
	if err != nil {
		return Config{}, err
	}

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = defaultPort
	}
	if err := validatePort(port); err != nil {
		return Config{}, err
	}

	allowedOrigins, err := parseAllowedOrigins(allowedOriginsValue)
	if err != nil {
		return Config{}, err
	}

	cookieSecure, err := loadCookieSecure()
	if err != nil {
		return Config{}, err
	}

	return Config{
		Port:           port,
		DatabaseURL:    databaseURL,
		AllowedOrigins: allowedOrigins,
		CookieSecure:   cookieSecure,
		IsRender:       os.Getenv("RENDER") == "true",
	}, nil
}

// Cookieは未設定時もHTTPS向けの安全な設定を既定値とする。
func loadCookieSecure() (bool, error) {
	value := strings.TrimSpace(os.Getenv("COOKIE_SECURE"))
	if value == "" {
		return true, nil
	}

	cookieSecure, err := strconv.ParseBool(value)
	if err != nil || (value != "true" && value != "false") {
		return false, fmt.Errorf("環境変数 COOKIE_SECURE にはtrueまたはfalseを設定してください")
	}

	return cookieSecure, nil
}

// HTTPサーバー以外のDB利用コマンドでも接続設定だけを読み込めるようにする。
func LoadDatabaseURL() (string, error) {
	return requiredEnvironmentVariable("DATABASE_URL")
}

func requiredEnvironmentVariable(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("必須環境変数 %s が設定されていません", name)
	}

	return value, nil
}

func validatePort(port string) error {
	portNumber, err := strconv.Atoi(port)
	if err != nil || portNumber < 1 || portNumber > 65535 {
		return fmt.Errorf("環境変数 PORT には1から65535までの整数を設定してください")
	}

	return nil
}

func parseAllowedOrigins(value string) ([]string, error) {
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		origin := strings.TrimSpace(part)
		parsed, err := url.Parse(origin)
		if err != nil {
			return nil, fmt.Errorf("環境変数 ALLOWED_ORIGINS に不正なOriginが含まれています")
		}

		canonicalOrigin := parsed.Scheme + "://" + parsed.Host
		if (parsed.Scheme != "http" && parsed.Scheme != "https") ||
			parsed.Host == "" || parsed.Hostname() == "" || parsed.User != nil ||
			parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" ||
			origin != canonicalOrigin {
			return nil, fmt.Errorf("環境変数 ALLOWED_ORIGINS に不正なOriginが含まれています")
		}

		if _, exists := seen[origin]; exists {
			continue
		}
		seen[origin] = struct{}{}
		origins = append(origins, origin)
	}

	return origins, nil
}
