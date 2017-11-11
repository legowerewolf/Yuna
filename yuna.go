package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

type person struct {
	DiscordID string   `json:"discordID"`
	Names     []string `json:"names"`
}

type data struct {
	APIToken string   `json:"apitoken"`
	People   []person `json:"people"`
}

var rundata data
var client discordgo.Session

func main() {
	rundata = getData("./data.json")

	client, err := discordgo.New(rundata.APIToken)
	checkErr(err)
	client.Open()

	command := "Yuna, mute adria."
	interpret(command)

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	shutdown()
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
			if isValueInList("and", s[i+1:]) != -1 {
				andindex := isValueInList("and", s[i+1:])
				for _, alias := range s[i+1 : andindex+i] {
					mute(alias)
				}
				mute(s[andindex+1])
			} else {
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

func getPersonFromAlias(alias string) (person, error) {
	for _, person := range rundata.People {
		for _, name := range person.Names {
			if strings.ToLower(name) == strings.ToLower(alias) {
				fmt.Println("Target match: " + person.Names[0])
				return person, nil
			}
		}
	}
	return person{}, errors.New("Could not find person with alias: " + alias)
}

func isValueInList(value string, list []string) int {
	for i, v := range list {
		if v == value {
			return i
		}
	}
	return -1
}

func mute(alias string) {
	aperson, err := getPersonFromAlias(alias)
	if err != nil {
		checkErr(err)
		return
	}
	discordID := aperson.DiscordID
	fmt.Println("Muting ID " + string(discordID))
}

func shutdown() {
	client.Close()
	saveData("./data.json")

}

func checkErr(err error) {
	if err != nil {
		log.Fatal("ERROR:", err)
	}
}
