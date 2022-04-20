package util

import (
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestGetOperations(t *testing.T) {
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

func FuzzContains(f *testing.F) {
	slice := []string{"EdgeNet", " ", "12345", "!@#123EdgeNet"} // Hard coded with 4 elements, as fuzz need a determinent number of paramenter
	for _, tc := range slice {
		f.Add(tc)
	}
	f.Fuzz(func(t *testing.T, value string) {
		ret_1, index_1 := Contains(slice, value)
		if ret_1 {
			slice[index_1] = value + "~!@#$%^&()_+" // Assume the possibility that {slice} contains this new string is very low
			ret_2, _ := Contains(slice, value)
			if ret_2 {
				t.Errorf("slice does not contains {%q} but return true", value)
			}
		} else {
			rand.Seed(time.Now().UnixNano())
			slice[rand.Intn(len(slice))] = StrDeepCopy(value)
			ret_2, index_2 := Contains(slice, value)
			if !ret_2 {
				t.Errorf("slice contains {%q} but return false", value)
			}
			if index_2 < 0 || index_2 >= len(slice) {
				t.Errorf("The index returned is {%d} out of range of {0,%d}", index_2, len(slice))
			}
		}
	})
}

func FuzzSliceContains(f *testing.F) {
	slice := [][]string{
		{"EdgeNet", " ", "12345", "!@#123EdgeNet"},
		{"()", " ", "[]", "{}"},
		{"abc", " ", "abc123", "123"},
	}
	for _, tc := range slice {
		f.Add(tc[0], tc[1], tc[2], tc[3])
	}
	f.Fuzz(func(t *testing.T, tc_0 string, tc_1 string, tc_2 string, tc_3 string) {
		value := []string{tc_0, tc_1, tc_2, tc_3}
		ret_1, _ := SliceContains(slice, value)
		if ret_1 {
			rand.Seed(time.Now().UnixNano())
			i := rand.Intn(len(value))
			value[i] = value[i] + "~!@#$%^&()_+" // Assume the possibility that {slice} contains this new string array is very low
			ret_2, _ := SliceContains(slice, value)
			if ret_2 {
				t.Errorf("slice does not contains {%q} but return true", value)
			}
		} else {
			rand.Seed(time.Now().UnixNano())
			i := rand.Intn(len(slice))
			slice[i] = StrArrayDeepCopy(value)
			ret_2, index_2 := SliceContains(slice, value)
			if !ret_2 {
				t.Errorf("slice contains {%q} but return false", value)
			}
			if ret_2 {
				if index_2 < 0 || index_2 >= len(slice) {
					t.Errorf("The index returned is {%d} out of range of {0,%d}", index_2, len(slice))
				}
			}
		}
	})
}

func StrDeepCopy(a string) string {
	b := make([]byte, len(a))
	copy(b, a)
	return string(b)
}

func StrArrayDeepCopy(a []string) []string {
	b := make([]string, len(a))
	for j, v := range a {
		b[j] = StrDeepCopy(v)
	}
	return b
}
