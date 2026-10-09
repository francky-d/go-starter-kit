package config

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/labstack/gommon/bytes"
)

func mustBeOneOf(variable, value string, allowed ...string) error {
	if slices.Contains(allowed, value) {
		return nil
	}

	return fmt.Errorf("%s must be one of %v, got %q", variable, allowed, value)
}

func mustBeBetween(variable string, value, minimum, maximum int) error {
	if value >= minimum && value <= maximum {
		return nil
	}

	return fmt.Errorf("%s must be between %d and %d, got %d", variable, minimum, maximum, value)
}

func mustBeAtLeast(variable string, value, minimum int) error {
	if value >= minimum {
		return nil
	}

	return fmt.Errorf("%s must be at least %d, got %d", variable, minimum, value)
}

func mustBePositive(variable string, value time.Duration) error {
	if value > 0 {
		return nil
	}

	return fmt.Errorf("%s must be positive, got %s", variable, value)
}

// mustBeLongSecret never quotes the value: it is a secret.
func mustBeLongSecret(variable, value string, minimumBytes int) error {
	if len(value) >= minimumBytes {
		return nil
	}

	return fmt.Errorf("%s must be at least %d bytes long", variable, minimumBytes)
}

func mustNotBeBlank(variable, value string) error {
	if strings.TrimSpace(value) != "" {
		return nil
	}

	return fmt.Errorf("%s must not be empty", variable)
}

// trimEach tolerates spaces around the items of a comma-separated list.
func trimEach(values []string) []string {
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			trimmed = append(trimmed, value)
		}
	}

	return trimmed
}

// mustExceed keeps the write timeout above the request timeout, so a request
// that times out still has time to receive its 503 response.
func mustExceed(variable string, value time.Duration, other string, otherValue time.Duration) error {
	if value > otherValue {
		return nil
	}

	return fmt.Errorf("%s (%s) must be greater than %s (%s)", variable, value, other, otherValue)
}

func mustBeByteSize(variable, value string) error {
	if size, err := bytes.Parse(value); err == nil && size > 0 {
		return nil
	}

	return fmt.Errorf("%s must be a size such as 512K or 1M, got %q", variable, value)
}

func mustBeOrigins(variable string, origins []string) error {
	var problems []error
	for _, origin := range origins {
		if !isOrigin(origin) {
			problems = append(problems, fmt.Errorf("%s entries must be * or scheme://host[:port], got %q", variable, origin))
		}
	}

	return errors.Join(problems...)
}

func isOrigin(value string) bool {
	if value == "*" {
		return true
	}

	origin, err := url.Parse(value)

	return err == nil && (origin.Scheme == "http" || origin.Scheme == "https") &&
		origin.Host != "" && origin.Path == "" && origin.RawQuery == ""
}
