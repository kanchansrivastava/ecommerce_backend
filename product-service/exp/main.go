package main

import (
	"fmt"
	"strconv"
	"strings"
)

// RupeesToPaise converts a price from rupees (e.g., "99.99") to paise (e.g., 9999).
//func RupeesToPaise(price string) (uint64, error) {
//
//	price = strings.TrimSpace(price)
//	// Split the price into integer and fractional parts using the dot (.)
//	parts := strings.Split(price, ".")
//	if len(parts) > 2 {
//		return 0, errors.New("price cannot have more than one decimal point")
//	}
//
//	// Convert integer part to paise (e.g., "99" -> 9900)
//	integerPart, err := strconv.ParseUint(parts[0], 10, 64)
//	if err != nil {
//		return 0, errors.New("invalid integer part in price")
//	}
//
//	if integerPart > 10000000 { // greater than ten million, then we are not selling that product,
//		return 0, errors.New("price cannot be greater than 10 million")
//	}
//
//	// Handle fractional part if it exists
//	var fractionalPart uint64 = 0
//	if len(parts) > 1 {
//		// Ensure no more than two decimal places
//		if len(parts[1]) > 2 {
//			return 0, errors.New("price cannot have more than two decimal places")
//		}
//		// Add trailing zero if fractional part has only one digit
//		for len(parts[1]) < 2 {
//			parts[1] += "0"
//		}
//		// Convert fractional part to paise
//		fractionalPart, err = strconv.ParseUint(parts[1], 10, 64)
//		if err != nil {
//			return 0, errors.New("invalid fractional part in price")
//		}
//	}
//
//	return integerPart*100 + fractionalPart, nil
//}

func main() {
	fmt.Println(RupeesToPaise("99.9 "))
	fmt.Println(RupeesToPaise("99.999"))
	fmt.Println(RupeesToPaise("099.9"))
	fmt.Println(RupeesToPaise("099.99"))
	fmt.Println(RupeesToPaise("99.9v"))
	fmt.Println(RupeesToPaise("99."))
	fmt.Println(RupeesToPaise("95"))
	fmt.Println(RupeesToPaise("A9.99"))
	fmt.Println(RupeesToPaise("425.5"))

}

func RupeesToPaise(priceStr string) (uint64, error) {
	fmt.Println("Input price:", priceStr)
	//trim extra space from price
	priceStr = strings.Trim(priceStr, " ")

	//split the price based by dot(.)
	prices := strings.Split(priceStr, ".")
	var rupee, paisa uint64
	if len(prices) == 0 || len(prices) > 2 {
		return 0, fmt.Errorf("invalid price, empty price field or more than one dot(.)")
	}

	rupee, err := strconv.ParseUint(prices[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid price, not a valid number")
	}

	if len(prices) == 2 {
		paisa, err = strconv.ParseUint(prices[1], 10, 64)
		if err != nil || paisa > 99 {
			return 0, fmt.Errorf("invalid price, please provide price in valid format")
		}

		// append 0 if paisa part has only one digit
		// e.g INR 99.2 => Convert it to 9900 + 20 = 9920
		if paisa < 10 {
			paisa *= 10
		}
	}
	return rupee*100 + paisa, nil
}
