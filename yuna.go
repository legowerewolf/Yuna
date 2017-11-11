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
	defer shutdown()

	interpret("Mute Rachel, Josh, and Rachel")
	//register listeners here

	fmt.Println("Yuna is now online.  Press CTRL-C to shut down.")
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
			for _, user := range getPeopleFromSlice(s[i+1:]) {
				mute(user.DiscordID)
			}
			return
		case "shutdown":
			shutdown()
		default:

		}
	}
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

func getPeopleFromSlice(s []string) []person {
	ret := []person{}
	if isValueInList("and", s) != -1 {
		andindex := isValueInList("and", s)
		for _, alias := range append(s[:andindex+1], s[andindex+1]) {
			nperson, err := getPersonFromAlias(alias)
			if err == nil {
				ret = append(ret, nperson)
			}
		}
	} else {
		nperson, err := getPersonFromAlias(s[1])
		if err == nil {
			ret = append(ret, nperson)
		}
	}
	return ret
}

func mute(discordID string) error {
	fmt.Println("Muting ID " + discordID)
	return nil
}

func shutdown() { //Shutdown the discord connection and save data
	client.Close()
	saveData("./data.json")

}

//Functions to load and save data

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

//Utility functions

func isValueInList(value string, list []string) int {
	for i, v := range list {
		if v == value {
			return i
		}
	}
	return -1
}

func checkErr(err error) {
	if err != nil {
		log.Fatal("ERROR:", err)
	}
}
