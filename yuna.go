package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	
	"github.com/bwmarrin/discordgo"
)

type person struct {
	DiscordID string   `json:"discordID"`
	Names     []string `json:"names"`
}

type data struct {
	ApiToken string `json:"apitoken"`
	People []person `json:"people"`
}

var rundata data
var client Session

func main() {
	rundata = getData("./data.json")
	
	client = discordgo.New(rundata.ApiToken)
	discord.Open()

	command := "Yuna, mute adria."
	interpret(command)
}

func sanitize(s string) []string {
	punct := []string{",", "."}
	for _, p := range punct {
		s = strings.Replace(s, p, "", -1)
	}

	words := strings.Split(s, " ")
	return words
}

func interpret(command string) {
	s := sanitize(command)
	for i, word := range s {
		word = strings.ToLower(word)
		switch word {
		case "mute":
			fmt.Println("keyword mute detected")
			if (isValueInList("and", s[i+1:])){
				andindex := isValueInList("and", s[i+1:])
				for _, alias := range s[i+1, andindex + i] {
					mute(alias)
				}
				mute(s[andindex+1])
			}
			else {
				mute(s[i+1])
			}
			return
		case "shutdown":
			shutdown()
		default:

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

func saveData(path string) {
	dat, err := json.Marshal(rundata)
	if err != nil {
		fmt.Println(err.Error())
	}
	ioutil.WriteFile(path, dat, 0644)
}

func getPersonFromAlias(alias string) person {
	for _, person := range rundata.People {
				for _, name := range person.Names {
					if strings.ToLower(name) == strings.ToLower(alias) {
						fmt.Println("Target match: " + person.Names[0])
						return person
					}
				}
			}
	return nil
}

func isValueInList(value string, list []string) bool {
    for i, v := range list {
        if v == value {
            return i
        }
    }
    return -1
}

func mute(alias string) {
	discordID := getPersonFromAlias(target).DiscordID
	fmt.Println("Muting ID " + string(discordID))
}

func shutdown(){
	client.Close()
	saveData("./data.json")
	
}
