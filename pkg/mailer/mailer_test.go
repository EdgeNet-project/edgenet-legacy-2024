package mailer

import (
	"regexp"
	"strings"
	"testing"
)

func TestGenerateRandomString(t *testing.T) {

	var codes []string

	for i := 1; i <= 100; i++ {
		task := generateRandomString(10)
		if len(task) != 10 {
			t.Errorf("code %d has wrong length", len(task))
		}
		//string unique
		if len(codes) != 0 {
			for _, code := range codes {
				if (strings.Compare(task, code)) == 0 {
					t.Errorf("duplicate code %s received", task)
				}
			}
		}
		codes = append(codes, task)

		// if string
		var IsLetter = regexp.MustCompile(`^[a-zA-Z]+$`).MatchString

		if !IsLetter(task) {
			t.Errorf("Not string code %s received", task)
		}
	}
}
