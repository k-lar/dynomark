package main

import (
	"strconv"
	"unicode"
)

func NaturalSort(s1, s2 string) bool {
	i, j := 0, 0
	for i < len(s1) && j < len(s2) {
		c1, c2 := s1[i], s2[j]

		if unicode.IsDigit(rune(c1)) && unicode.IsDigit(rune(c2)) {
			num1, nextI := extractNumber(s1, i)
			num2, nextJ := extractNumber(s2, j)
			if num1 != num2 {
				return num1 < num2
			}
			i, j = nextI, nextJ
		} else {
			// Compare as strings instead
			if c1 != c2 {
				return c1 < c2
			}
			i++
			j++
		}
	}

	// If reached the end of one of the strings, the shorter string is less
	return len(s1) < len(s2)
}

func extractNumber(s string, i int) (int, int) {
	start := i
	for i < len(s) && unicode.IsDigit(rune(s[i])) {
		i++
	}
	num, _ := strconv.Atoi(s[start:i])
	return num, i
}
