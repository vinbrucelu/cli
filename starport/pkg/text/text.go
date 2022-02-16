package text

import (
	"bytes"
	"strings"
)

func WrapWordsFromString(words string, afterN int) string {
	return WrapWords(strings.Fields(words), afterN)
}

func WrapWords(words []string, afterN int) string {
	b := &bytes.Buffer{}
	for i, word := range words {
		if i != 0 {
			if i%afterN == 0 {
				b.WriteRune('\n')
			} else {
				b.WriteRune(' ')
			}
		}
		b.WriteString(word)
	}
	return b.String()
}

func AddTabToLines(text string, count int) string {
	b := &bytes.Buffer{}
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		b.WriteString(line)
		b.WriteRune('\n')
		for i := 0; i < count; i++ {
			b.WriteRune('\t')
		}
	}
	return strings.TrimSpace(b.String())
}
