package products

import (
	"fmt"
	"strconv"
	"strings"
)

func ValidatePrice(price string) (uint, error) {
	price = strings.TrimSpace(price)
	parts := strings.Split(price, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid price format: too many parts")
	}

	rsPart := parts[0]
	if len(rsPart) == 0 || !isUint(rsPart) {
		return 0, fmt.Errorf("invalid rupee part: must be a valid positive integer")
	}

	rupee, err := strconv.Atoi(rsPart)
	if err != nil || rupee < 100 || rupee > 1000000 {
		return 0, fmt.Errorf("rupee part must be between 100 and 1,000,000")
	}

	paisaPart := "00"
	if len(parts) == 2 {
		paisaPart = parts[1]
		if !isUint(paisaPart) || len(paisaPart) > 2 {
			return 0, fmt.Errorf("invalid paisa part: must be a valid number with at most two digits")
		}
		if len(paisaPart) == 1 {
			paisaPart = "0" + paisaPart
		}
	}

	// Calculate the final price in paisa (rupee * 100 + paisa)
	paisa, err := strconv.Atoi(paisaPart)
	if err != nil {
		return 0, fmt.Errorf("invalid paisa part: %w", err)
	}
	finalPrice := rupee*100 + paisa

	return uint(finalPrice), nil
}

func isUint(value string) bool {
	_, err := strconv.Atoi(value)
	return err == nil
}
