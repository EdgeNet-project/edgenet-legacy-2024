package main

import "fmt"

func main() {
	s := map[string]string{
		"test": "",
	}

	if e, ok := s["test"]; true {
		fmt.Printf("ok: %t\n", ok)
		fmt.Printf("e: %q\n", e)
	}
}
