package main

import (
	"fmt"
	"strings"
)

func main() {
	command := "Yuna, mute Josh."
	fmt.Println(sanitize(command))
}

func sanitize(s string) []string {
	punct := []string{",", "."}
	for _, p := range punct {
		s = strings.Replace(s, p, "", -1)
	}

	words := strings.Split(s, " ")
	return words
}
