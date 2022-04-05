package util

import (
	"math/rand"
	"testing"
	"time"
)

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
