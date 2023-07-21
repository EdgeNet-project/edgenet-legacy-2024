/*
Copyright 2021 Contributors to the EdgeNet project.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"
)

// GenerateRandomString to have a unique code
func GenerateRandomString(n int) string {
	var letter = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

	b := make([]rune, n)
	rand.Seed(time.Now().UnixNano())
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}

// Contains returns whether slice contains the value
func Contains(slice []string, value string) (bool, int) {
	for i, ele := range slice {
		if value == ele {
			return true, i
		}
	}
	return false, 0
}

// SliceContains returns whether slice contains the slice
func SliceContains(slice [][]string, value []string) (bool, int) {
	for i, ele := range slice {
		if reflect.DeepEqual(value, ele) {
			return true, i
		}
	}
	return false, 0
}

// Assert fails the test if the condition is false.
func Assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		//tb.FailNow()
		tb.Fail()
	}
}

// OK fails the test if an err is not nil.
func OK(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		//tb.FailNow()
		tb.Fail()
	}
}

// Equals fails the test if exp is not equal to act.
func Equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		//tb.FailNow()
		tb.Fail()
	}
}

// NotEquals fails the test if exp is equal to act.
func NotEquals(tb testing.TB, exp, act interface{}) {
	if reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp different from: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		//tb.FailNow()
		tb.Fail()
	}
}

// EqualsMultipleExp fails the test if exp is not equal to one of act.
func EqualsMultipleExp(tb testing.TB, exp interface{}, act interface{}) {
	check := func(exp, act interface{}) bool {
		fail := true
		if !reflect.DeepEqual(exp, act) {
			_, file, line, _ := runtime.Caller(1)
			fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		} else {
			fail = false
		}
		return fail
	}
	if reflect.TypeOf(exp).Kind() == reflect.Slice {
		val := reflect.ValueOf(exp)
		expRaw, ok := val.Interface().([]string)
		if !ok {
			expRaw, ok := val.Interface().([]int)
			if !ok {
				expRaw, ok := val.Interface().([]bool)
				if !ok {
					Equals(tb, exp, act)
				} else {
					fail := true
					for _, expRow := range expRaw {
						fail = check(expRow, act)
						if !fail {
							break
						}
					}
					if fail {
						tb.Fail()
					}
				}
			} else {
				fail := true
				for _, expRow := range expRaw {
					fail = check(expRow, act)
					if !fail {
						break
					}
				}
				if fail {
					tb.Fail()
				}
			}
		} else {
			fail := true
			for _, expRow := range expRaw {
				fail = check(expRow, act)
				if !fail {
					break
				}
			}
			if fail {
				tb.Fail()
			}
		}
	} else {
		Equals(tb, exp, act)
	}
}
