package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var (
	rundata   database
	dclient   *discordgo.Session
	dvcontrol map[string]chan string
)

func main() {
	fmt.Println("Starting...")

	//Load database
	rundata = getData()
	rundata = rundata.checkForUpdates()

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
		intent, _ := rundata.intentOf(m.Content)
		fmt.Println(intent + " " + m.Content)
	}
}

func interpret(command, channelID string, mem *discordgo.Member) string {

	intent, data := rundata.intentOf(command)

	returnValue, notHandled := "", false

	if rundata.checkAuthorized(mem.User.ID, intent) {
		switch intent {
		case "shutdown":
			returnValue = "Alright. Goodbye!"
			defer shutdown()
		case "reload_data":
			rundata = getData()
			returnValue = "Alright, I've reloaded my database from disk."
		case "list_names":
			person, _, _ := rundata.getPersonFromAlias(sanitize(command)[indexOf("for", sanitize(command))+1])
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
				returnValue = getRandomString(rundata.Errors["user_not_in_voice_channel"])
				break
			}
			c := make(chan string)
			dvcontrol[mem.GuildID] = c
			go voiceService(voicedata{session: dclient, guildID: mem.GuildID, vChannelID: id, commandChan: c, tChannelID: channelID})
			returnValue = getRandomString(rundata.Intents[intent].Responses)
		case "end_voice_connection":
			dvcontrol[mem.GuildID] <- "disconnect"
			delete(dvcontrol, mem.GuildID)
			returnValue = getRandomString(rundata.Intents[intent].Responses)
		case "create_temp_channel":
			var channame string
			if cn, prs := data["CHANNAME"]; prs {
				channame = cn
			} else {
				channame = getRandomString(rundata.Intents[intent].Extra1)
			}
			if len(channame) < 2 || len(channame) > 100 {
				returnValue = getRandomString(rundata.Errors["channel_name_too_short_long"])
			} else {
				c := make(chan string, 5)
				dvcontrol[strconv.Itoa(len(dvcontrol))] = c
				go tempChannelManager(voicedata{session: dclient, guildID: mem.GuildID, vChannelName: channame, creatorID: mem.User.ID, commandChan: c})
				returnValue = "I've created a temporary channel for you: " + channame
			}
		default:
			if _, prs := rundata.Intents[intent]; prs {
				returnValue = getRandomString(rundata.Intents[intent].Responses)
			} else {
				notHandled = true
				returnValue = getRandomString(rundata.Errors["unknown_intent"])
			}
		}

	} else {
		returnValue = getRandomString(rundata.Errors["not_authorized"])
	}

	chatbaseSubmit(Message{Apikey: rundata.APITokens["chatbase"], Creator: "user", Userid: mem.User.ID, Platform: "Discord", Message: command, Intent: intent, Nothandled: notHandled})
	chatbaseSubmit(Message{Apikey: rundata.APITokens["chatbase"], Creator: "agent", Userid: mem.User.ID, Platform: "Discord", Message: returnValue})

	if notHandled {
		ioutil.WriteFile("./data/errors.log", []byte(command+"\n"), 0644)
	}
	return returnValue
}

//Utility functions

func checkErr(err error, key string) {
	if err != nil {
		log.Fatal("ERROR at \""+key+"\": ", err)
	}
}

func sanitize(s string) []string { //Take a string, remove the punctuation, and return it as a []string.
	punct := []string{",", ".", "!"}
	for _, p := range punct {
		s = strings.Replace(s, p, "", -1)
	}
	words := strings.Split(s, " ")
	return words
}

func shutdown() { //Shutdown the discord connection and save data
	fmt.Println("Sending disconnect signal...")
	for _, c := range dvcontrol {
		c <- "disconnect"
	}
	dclient.Close()
	fmt.Println("Saving...")
	rundata.save("./data/config.json")
	fmt.Println("Done.")
	os.Exit(0)
}
