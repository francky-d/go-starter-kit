package config

import (
	"fmt"
	"slices"
	"time"
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
