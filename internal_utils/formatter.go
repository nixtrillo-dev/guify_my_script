package internal_utils

import (
	"strings"
)

func SplitCamelCase(src string) []string {
	if len(src) == 0 {
		return nil
	}

	var words []string
	var l int

	for r := 1; r < len(src); r++ {
		isUpper := src[r] >= 'A' && src[r] <= 'Z'
		if isUpper {
			prevIsLower := src[r-1] >= 'a' && src[r-1] <= 'z'
			nextIsLower := r+1 < len(src) && src[r+1] >= 'a' && src[r+1] <= 'z'

			// Split if transitioning from lowercase to uppercase (e.g., camelCase)
			// OR if transitioning from acronym to a standard word (e.g., ABCMethod -> ABC Method)
			if prevIsLower || nextIsLower {
				if l < r {
					words = append(words, src[l:r])
				}
				l = r
			}
		}
	}

	if l < len(src) {
		words = append(words, src[l:])
	}

	return words
}

func ToTitleCase(input string) string {
	// First, run it through the camel case splitter just in case the input
	// is a raw camelCase string rather than snake/kebab case.
	camelSplit := SplitCamelCase(input)
	input = strings.Join(camelSplit, " ")

	// Replace standard delimiters with spaces
	replacer := strings.NewReplacer("_", " ", "-", " ")
	spaced := replacer.Replace(input)

	// Capitalize the first letter of each word
	words := strings.Fields(spaced)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + word[1:]
		}
	}

	return strings.Join(words, " ")
}
