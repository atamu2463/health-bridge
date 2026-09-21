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
}

func Load() (Config, error) {
	databaseURL, err := requiredEnvironmentVariable("DATABASE_URL")
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

	return Config{
		Port:           port,
		DatabaseURL:    databaseURL,
		AllowedOrigins: allowedOrigins,
	}, nil
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
		if err != nil ||
			(parsed.Scheme != "http" && parsed.Scheme != "https") ||
			parsed.Host == "" || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
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
