package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"reflect"
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
	RoleName string   `json:"roleName"`
	People   []person `json:"people"`
}

var rundata data
var client *discordgo.Session

func main() {
	rundata = getData("./data.json")

	var err error
	client, err = discordgo.New("Bot " + rundata.APIToken)
	checkErr(err)

	//register listeners here
	client.AddHandler(messageCreate)

	client.Open()    //The client is connecting.
	defer shutdown() //In the event that something happens, shut down cleanly.

	fmt.Println("Yuna is now online.  Press CTRL-C to shut down.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	chn, err := s.Channel(m.ChannelID)
	checkErr(err)

	guild, err := s.Guild(chn.GuildID)
	checkErr(err)

	mem, err := s.GuildMember(guild.ID, m.Author.ID)
	checkErr(err)
	mem.GuildID = guild.ID

	if indexOf("yuna", sanitize(strings.ToLower(m.Content))) != -1 || indexOf(client.State.User, m.Mentions) != -1 {
		_, err := s.ChannelMessageSend(m.ChannelID, interpret(m.Content, mem))
		checkErr(err)
	}
}

func interpret(command string, mem *discordgo.Member) string {
	s := sanitize(command)
	authorized := checkAuthorized(mem)
	for i, word := range s {
		word = strings.ToLower(word)
		switch word {
		case "mute":
			if !authorized {
				return "Sorry, I won't take that command from you."
			}
			if len(getPeopleFromSlice(s[i+1:])) == 0 {
				return "Sorry, but I couldn't find anybody by that name. Try again?"
			}
			ret := "Alright, I've muted: "
			for _, user := range getPeopleFromSlice(s[i+1:]) {
				mem, err := client.GuildMember(mem.GuildID, user.DiscordID)
				checkErr(err)
				mute(mem)
				ret += user.Names[0]
			}
			return ret
		case "shutdown":
			if !authorized {
				return "Sorry, I won't take that command from you."
			}
			shutdown()
		default:

		}
	}
	return "Sorry, what was that?"
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

func indexOf(value interface{}, list interface{}) int {
	switch list.(type) {
	case []string:
		list := []string(list.([]string))
		for i, v := range list {
			if v == value {
				return i
			}
		}
	case []*discordgo.User:
		list := []*discordgo.User(list.([]*discordgo.User))
		value2 := value.(*discordgo.User)
		for i, v := range list {
			if value2.ID == v.ID {
				return i
			}
		}
	default:
		fmt.Print(value)
		fmt.Println(reflect.TypeOf(value))
		fmt.Print(list)
		fmt.Println(reflect.TypeOf(list))
	}
	return -1
}

func checkErr(err error) {
	if err != nil {
		log.Fatal("ERROR: ", err)
	}
}

func checkAuthorized(mem *discordgo.Member) bool {
	guild, err := client.Guild(mem.GuildID)
	checkErr(err)
	authrole := ""
	for _, role := range guild.Roles {
		if role.Name == rundata.RoleName {
			authrole = role.ID
			break
		}
	}
	authorized := false
	for _, r := range mem.Roles {
		if r == authrole {
			authorized = true
			break
		}
	}
	return authorized
}

func sanitize(s string) []string {
	punct := []string{",", "."}
	for _, p := range punct {
		s = strings.Replace(s, p, "", -1)
	}
	words := strings.Split(s, " ")
	return words
}

func getPersonFromAlias(alias string) (person, error) {
	for _, person := range rundata.People {
		for _, name := range person.Names {
			if strings.ToLower(name) == strings.ToLower(alias) {
				return person, nil
			}
		}
	}
	return person{}, errors.New("Could not find person with alias: " + alias)
}

func getPeopleFromSlice(s []string) []person {
	ret := []person{}
	if indexOf("and", s) != -1 {
		andindex := indexOf("and", s)
		for _, alias := range append(s[:andindex+1], s[andindex+1]) {
			nperson, err := getPersonFromAlias(alias)
			if err == nil {
				ret = append(ret, nperson)
			}
		}
	} else {
		nperson, err := getPersonFromAlias(s[0])
		if err == nil {
			ret = append(ret, nperson)
		}
	}
	return ret
}

func mute(user *discordgo.Member) error {
	fmt.Println("Muting ID " + user.User.ID)
	user.Mute = true
	return nil
}

func shutdown() { //Shutdown the discord connection and save data
	client.Close()
	saveData("./data.json")
	os.Exit(0)
}
