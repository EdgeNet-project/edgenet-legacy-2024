package util

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateRandomString(t *testing.T) {
	var codes []string
	for i := 1; i <= 100; i++ {
		time.Sleep(1 * time.Nanosecond)
		task := GenerateRandomString(10)
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
	}
}
