package util

import (
	"strings"
	"testing"
	"time"
)

func TestGetConfigView(t *testing.T) {
	result, err := getConfigView()
	if result == "" {
		t.Errorf("fail")
	}
	if err != nil {
		t.Errorf("fail error")
	}
}

func TestGetClusterServerOfCurrentContext(t *testing.T) {
	_, _, _, err := GetClusterServerOfCurrentContext()
	if err != nil {
		t.Errorf("get cluster server of current context failed")
	}
}

func TestGetServerOfCurrentContext(t *testing.T) {
	_, err := GetServerOfCurrentContext()
	if err != nil {
		t.Errorf("get server of current context failed")
	}
}

func TestGetNamecheapCredentials(t *testing.T) {
	_, _, _, err := GetNamecheapCredentials()
	if err != nil {
		t.Log(err)
		t.Errorf("Get namecheap credentials failed")
	}
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
