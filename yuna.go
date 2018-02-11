package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/m90/go-chatbase"
)

//All fields are exported because of the JSON package
type person struct {
	DiscordID string   `json:"discordID"`
	Names     []string `json:"names"`
}

var (
	rundata  database
	dclient  *discordgo.Session
	dvclient *discordgo.VoiceConnection
	cclient  *chatbase.Client
)

func main() {
	//Load database
	rundata = getData()

	//Build the Chatbase client
	cclient = chatbase.New(rundata.APITokens["chatbase"])

	//Build the Discord client
	var err error
	dclient, err = discordgo.New("Bot " + rundata.APITokens["discord"])
	checkErr(err)

	//register discord listeners here
	dclient.AddHandler(messageCreate)

	//connect to Discord servers
	checkErr(dclient.Open())
	dvclient, err = dclient.ChannelVoiceJoin(rundata.Guild, rundata.VoiceChannel, true, false)
	checkErr(err)

	//Wait for a manual shutdown
	defer shutdown()
	fmt.Println("Yuna is now online.  Press CTRL-C to shut down.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	//Check and see if the person who sent the message is the bot itself - if so, don't respond.
	if m.Author.ID == s.State.User.ID {
		return
	}

	if indexOf(s.State.User, m.Mentions) != -1 { //Check and see if the bot is @mentioned

		//Get the guildmember who sent the message
		chn, err := s.Channel(m.ChannelID)
		checkErr(err)
		guild, err := s.Guild(chn.GuildID)
		checkErr(err)
		mem, err := s.GuildMember(guild.ID, m.Author.ID)
		checkErr(err)
		mem.GuildID = guild.ID

		//Send a message back on the same channel with the feedback returned by interpret()
		_, err = s.ChannelMessageSend(m.ChannelID, interpret(m.Content, mem))
		checkErr(err)
		fmt.Println(intentOf(m.Content, rundata.Models) + " " + m.Content)
	}
}

func interpret(command string, mem *discordgo.Member) string {
	authorized := checkAuthorized(mem)
	returnValue := ""

	messageReport := cclient.UserMessage(mem.User.ID, "Discord")
	messageReport.SetMessage(command)
	messageReport.SetIntent(intentOf(command, rundata.Models))

	switch intentOf(command, rundata.Models) {
	case "mute":
		if !authorized {
			returnValue = "Sorry, I won't take that command from you."
			break
		}
		s := sanitize(command)
		if len(getPeopleFromSlice(s)) == 0 {
			returnValue = "Sorry, but I couldn't find anybody by that name. Try again?"
			break
		}
		returnValue = "Alright, I've muted "
		users := []string{}
		for _, user := range getPeopleFromSlice(s) {
			mem, err := dclient.GuildMember(mem.GuildID, user.DiscordID)
			checkErr(err)
			mute(mem)

			users = append(users, user.Names[0])
		}
		returnValue += toEnglishList(users)
	case "shutdown":
		if !authorized {
			returnValue = "Sorry, I won't take that command from you."
			break
		}
		returnValue = "Alright. Goodbye!"
		defer shutdown()
	case "nrgreeting":
		responses := []string{"Hello!", "Hi!", "Greetings."}
		returnValue = responses[rand.Intn(len(responses))]
	case "reload_data":
		rundata = getData()
		returnValue = "Alright, I've reloaded my database from disk."
	case "list_names":
		person, _ := getPersonFromAlias(sanitize(command)[indexOf("for", sanitize(command))+1])
		returnValue = "That user is known as " + toEnglishList(person.Names)
	case "play_music":
		returnValue = "I understand you want me to play music. I don't quite know how to do that yet."
	case "create_voice_channel":
	case "export":
		if !authorized {
			returnValue = "I'm not telling you that."
			break
		}
		dat, err := json.Marshal(rundata)
		checkErr(err)
		returnValue = symmetricEncrypt(string(dat), sanitize(command)[len(sanitize(command))-1])
	default:
		messageReport.SetNotHandled(true)
		returnValue = "Sorry, what was that?"
	}
	messageReport.Submit()
	return returnValue
}

//Utility functions

func checkErr(err error) {
	if err != nil {
		log.Fatal("ERROR: ", err)
	}
}

func checkAuthorized(mem *discordgo.Member) bool { //Check and see if the member has advanced permissions based on what roles they have.
	guild, err := dclient.Guild(mem.GuildID)
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

func sanitize(s string) []string { //Take a string, remove the punctuation, and return it as a []string.
	punct := []string{",", ".", "!"}
	for _, p := range punct {
		s = strings.Replace(s, p, "", -1)
	}
	words := strings.Split(s, " ")
	return words
}

func getPersonFromAlias(alias string) (person, error) {
	for _, person := range rundata.People { //Scan through people the bot is aware of
		if alias[2:len(alias)-1] == person.DiscordID { //See if this is a mention (<@ ... >) of the person
			return person, nil
		}
		for _, name := range person.Names { //Scan through names the bot knows for this person, ignoring case
			if strings.ToLower(name) == strings.ToLower(alias) {
				return person, nil
			}
		}

	}

	//The bot doesn't know this person. Add it to the registry and return the new person.
	matched, _ := regexp.MatchString("(:?)([0-9])+", alias[2:len(alias)-1])
	if len(alias[2:len(alias)-1]) == 18 && matched {
		rundata.People = append(rundata.People, person{DiscordID: alias[2 : len(alias)-1]})
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
	dvclient.Disconnect()
	dvclient.Close()
	dclient.Close()
	saveData()
	os.Exit(0)
}
