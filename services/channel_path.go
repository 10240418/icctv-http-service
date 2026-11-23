package services

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidChannelPath  = errors.New("path name must be channel followed by a positive integer without leading zeros, e.g. channel1 or channel10")
	ErrReservedChannelPath = errors.New("channel path already exists or its permission ID is reserved")
)

func ValidateChannelPath(name string) error {
	if numericChannel(name) == 0 {
		return ErrInvalidChannelPath
	}
	return nil
}

func validateChannelPathConfig(name string, config map[string]interface{}) error {
	if err := ValidateChannelPath(name); err != nil {
		return err
	}
	if config == nil {
		return fmt.Errorf("%w: config is required", ErrInvalidChannelPath)
	}
	// Path identity belongs to the request, not the arbitrary config payload.
	if _, exists := config["name"]; exists {
		return fmt.Errorf("%w: config.name is not allowed", ErrInvalidChannelPath)
	}
	return nil
}
