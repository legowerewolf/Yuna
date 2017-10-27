package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type person struct {
	DiscordID string   `json:"discordID"`
	Names     []string `json:"names"`
	Prefname  string   `json:"prefname"`
}

type data struct {
	People []person `json:"people"`
}

func main() {
	object := getData("./data.json")

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
			for _, person := range data.People {
				for _, name := range person.Names {
					if name == target {
						fmt.Println("Target match: " + person.Prefname)
						discordID = person.DiscordID
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

func getData(path string) data {
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	var c data
	json.Unmarshal(raw, &c)
	return c
}
