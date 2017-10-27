package main

import (
	"fmt"
	"strings"
)

type person struct {
	discordID string
	names     []string
	prefname  string
}

type data struct {
	people []person
}

func main() {
	object := data{
		people: []person{
			person{
				prefname:  "Josh",
				names:     []string{"Josh", "Rippey", "Eragon"},
				discordID: "210220334756921345"},
			person{
				prefname:  "Rachel",
				discordID: "226895271022428160"}}}
	command := "Yuna, mute Eragon."
	interpret(command, object)
}

func sanitize(s string) []string {
	punct := []string{",", "."}
	for _, p := range punct {
		s = strings.Replace(s, p, "", -1)
	}

	words := strings.Split(s, " ")
	return words
}

func interpret(command string, data data) {
	s := sanitize(command)
	for i, word := range s {
		word = strings.ToLower(word)
		switch word {
		case "mute":
			fmt.Println("keyword mute detected")
			target := s[i+1]
			discordID := ""
			for _, person := range data.people {
				for _, name := range person.names {
					if name == target {
						fmt.Println("Target match: " + person.prefname)
						discordID = person.discordID
					}
				}
			}
			fmt.Println("Muting ID " + string(discordID))
			return
		default:
			fmt.Println("")
		}
	}
}
