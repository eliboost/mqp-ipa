// Package instance provides utility functions for instance management.
package instance

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ParseSizeToBytes converts a memory/storage string (e.g., "2Gi", "500Mi") to bytes.
// Supports IEC units: Ki, Mi, Gi. Returns error for invalid formats.
func ParseSizeToBytes(sizeStr string) (int, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	if sizeStr == "" {
		return 0, fmt.Errorf("%s", "empty string is not a valid input")
	}

	re := regexp.MustCompile(`(?i)^([0-9]+)([KMGTPE]?i?)B?$`)

	matches := re.FindStringSubmatch(sizeStr)
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}

	value64, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", err.Error())
	}

	value := int(value64)

	unit := strings.ToUpper(matches[2])

	var multiplier = 1

	switch unit {
	case "":
	case "K", "KI":
		multiplier = 1024
	case "M", "MI":
		multiplier = 1024 * 1024
	case "G", "GI":
		multiplier = 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown unit: %s", unit)
	}

	return value * multiplier, nil
}
