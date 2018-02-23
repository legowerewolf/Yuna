package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/legowerewolf/cryptowrapper"
	"github.com/m90/go-chatbase"
)

var (
	rundata   database
	dclient   *discordgo.Session
	dvcontrol map[string]chan string
	cclient   *chatbase.Client
)

func main() {
	//Load database
	rundata = getData()

	//Build the Chatbase client
	cclient = chatbase.New(rundata.APITokens["chatbase"])

	//Build the Discord client
	var err error
	dclient, err = discordgo.New("Bot " + rundata.APITokens["discord"])
	checkErr(err, "construct discord client")

	//register discord listeners here
	dclient.AddHandler(messageCreate)

	dvcontrol = make(map[string]chan string)
	//connect to Discord servers
	checkErr(dclient.Open(), "open discord connection")

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
		checkErr(err, "message recieved - get channel")
		guild, err := s.Guild(chn.GuildID)
		checkErr(err, "message recieved - get guild")
		mem, err := s.GuildMember(guild.ID, m.Author.ID)
		checkErr(err, "message recieved - get guildmember")
		mem.GuildID = guild.ID

		//Send a message back on the same channel with the feedback returned by interpret()
		_, err = s.ChannelMessageSend(m.ChannelID, interpret(m.Content, m.ChannelID, mem))
		checkErr(err, "message recieved - send response")
		fmt.Println(intentOf(m.Content, rundata.Models) + " " + m.Content)
	}
}

func interpret(command, channelID string, mem *discordgo.Member) string {
	authorized := checkAuthorized(mem)
	returnValue := ""

	messageReport := cclient.UserMessage(mem.User.ID, "Discord")
	messageReport.SetMessage(command)
	messageReport.SetIntent(intentOf(command, rundata.Models))

	switch intentOf(command, rundata.Models) {
	case "mute":
		if !authorized {
			returnValue = rundata.getRandomResponse("not_authorized")
			break
		}
		s := sanitize(command)
		p := rundata.getPeopleFromSlice(s)
		if len(p) == 0 {
			returnValue = "Sorry, but I couldn't find anybody by that name. Try again?"
			break
		}
		returnValue = "Alright, I've muted "
		users := []string{}
		for _, user := range p {
			mem, err := dclient.GuildMember(mem.GuildID, user.DiscordID)
			checkErr(err, "interpret mute - get guildmember for alias")
			mute(mem)

			users = append(users, user.Names[0])
		}
		returnValue += toEnglishList(users)
	case "shutdown":
		if !authorized {
			returnValue = rundata.getRandomResponse("not_authorized")
			break
		}
		returnValue = "Alright. Goodbye!"
		defer shutdown()
	case "nrgreeting":
		returnValue = rundata.getRandomResponse("greeting")
	case "reload_data":
		rundata = getData()
		returnValue = "Alright, I've reloaded my database from disk."
	case "list_names":
		person, _ := rundata.getPersonFromAlias(sanitize(command)[indexOf("for", sanitize(command))+1])
		returnValue = "That user is known as " + toEnglishList(person.Names)
	case "play_music":
		returnValue = "I understand you want me to play music. I don't quite know how to do that yet."
	case "start_voice_connection":
		_, prs := dvcontrol[mem.GuildID]
		if prs {
			dvcontrol[mem.GuildID] <- "disconnect"
		}
		id, err := getCurrentVoiceChannel(mem)
		if err != nil && err.Error() == "person not in voice channel" {
			returnValue = rundata.getRandomResponse("user_not_in_channel")
			break
		}
		c := make(chan string)
		dvcontrol[mem.GuildID] = c
		go voiceService(voicedata{session: dclient, guildID: mem.GuildID, vChannelID: id, commandChan: c, tChannelID: channelID})
		returnValue = rundata.getRandomResponse("start_voice_connection")
	case "end_voice_connection":
		dvcontrol[mem.GuildID] <- "disconnect"
		delete(dvcontrol, mem.GuildID)
		returnValue = rundata.getRandomResponse("end_voice_connection")
	case "create_temp_channel":
		c := make(chan string, 5)
		dvcontrol[strconv.Itoa(len(dvcontrol))] = c
		channame := rundata.getRandomResponse("temp_channel_names")
		go tempChannelManager(voicedata{session: dclient, guildID: mem.GuildID, vChannelName: channame, creatorID: mem.User.ID, commandChan: c})
		returnValue = "I've created a temporary channel for you: " + channame
	case "export":
		if !authorized {
			returnValue = rundata.getRandomResponse("not_authorized")
			break
		}
		dat, err := json.Marshal(rundata)
		checkErr(err, "export config")
		returnValue = cryptowrapper.SymmetricEncrypt(string(dat), sanitize(command)[len(sanitize(command))-1])
	default:
		messageReport.SetNotHandled(true)
		returnValue = rundata.getRandomResponse("unknown_intent")
	}
	messageReport.Submit()
	return returnValue
}

//Utility functions

func checkErr(err error, key string) {
	if err != nil {
		log.Fatal("ERROR at \""+key+"\": ", err)
	}
}

func checkAuthorized(mem *discordgo.Member) bool { //Check and see if the member has advanced permissions based on what roles they have.
	guild, err := dclient.Guild(mem.GuildID)
	checkErr(err, "authorization check - get guild")
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

func mute(user *discordgo.Member) error {
	fmt.Println("Muting ID " + user.User.ID)
	user.Mute = true
	return nil
}

func shutdown() { //Shutdown the discord connection and save data
	fmt.Println("Sending disconnect signal...")
	for _, c := range dvcontrol {
		c <- "disconnect"
	}
	dclient.Close()
	fmt.Println("Saving...")
	saveData()
	fmt.Println("Done.")
	os.Exit(0)
}
