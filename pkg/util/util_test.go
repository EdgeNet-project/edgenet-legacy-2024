package util

import (
	"flag"
	"strings"
	"testing"
	"time"
)

func TestGetOperations(t *testing.T) {
	flag.String("configs-path", "../../configs", "Set Namecheap path.")
	flag.Parse()

	t.Run("config view", func(t *testing.T) {
		_, err := getConfigView()
		OK(t, err)
	})
	t.Run("cluster-server of current context", func(t *testing.T) {
		_, _, _, err := GetClusterServerOfCurrentContext()
		OK(t, err)
	})
	t.Run("server from current context", func(t *testing.T) {
		_, err := GetServerOfCurrentContext()
		OK(t, err)
	})
	t.Run("namecheap credentials", func(t *testing.T) {
		_, _, _, err := GetNamecheapCredentials()
		OK(t, err)
	})
}

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
