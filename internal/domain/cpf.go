package domain

import "strings"

// NormalizeCPF strips everything that is not a digit, so "123.456.789-09"
// and "12345678909" are stored the same way.
func NormalizeCPF(cpf string) string {
	var b strings.Builder
	for _, r := range cpf {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// IsValidCPF checks length and both verification digits of an already
// normalized CPF.
func IsValidCPF(cpf string) bool {
	if len(cpf) != 11 {
		return false
	}

	digits := make([]int, 11)
	allEqual := true
	for i, r := range cpf {
		if r < '0' || r > '9' {
			return false
		}
		digits[i] = int(r - '0')
		if digits[i] != digits[0] {
			allEqual = false
		}
	}
	if allEqual {
		return false
	}

	return checkDigit(digits[:9]) == digits[9] && checkDigit(digits[:10]) == digits[10]
}

func checkDigit(digits []int) int {
	sum := 0
	weight := len(digits) + 1
	for _, d := range digits {
		sum += d * weight
		weight--
	}
	rest := (sum * 10) % 11
	if rest == 10 {
		return 0
	}
	return rest
}
