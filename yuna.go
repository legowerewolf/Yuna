package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

var (
	rundata *database
	dclient *discordgo.Session

	spcontrol     map[string]chan string
	spcontrolchan chan spdata
)

func main() {
	fmt.Println("Starting...")

	var err error

	//Load database
	rundata, err = getData()
	rundata.checkForUpdates()

	//initialize subprocess control
	spcontrol = make(map[string]chan string)
	spcontrolchan = make(chan spdata)

	//Build the Discord client
	dclient, err = discordgo.New("Bot " + rundata.APITokens["discord"])
	checkErr(err, "construct discord client")

	//register discord listeners here
	dclient.AddHandler(messageCreate)

	//connect to Discord servers
	checkErr(dclient.Open(), "open discord connection")

	//Wait for a manual shutdown
	defer shutdown()
	fmt.Println("Yuna is now online.  Press CTRL-C to shut down.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)

	for flag := false; flag == false; {
		select {
		case <-sc:
			flag = true
		case cd := <-spcontrolchan:
			if cd.command == "register" {
				spcontrol[cd.id] = cd.channel
			} else if cd.command == "delete" {
				delete(spcontrol, cd.id)
			}
		}
	}
}

type spdata struct {
	command string
	id      string
	channel chan string
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

		//intent, _ := rundata.intentOf(m.Content)
		//fmt.Println(intent + " " + m.Content)
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
			temp, err := getData()
			if err != nil {
				returnValue = getRandomString(rundata.Errors["unable_to_reload_database"])
			} else {
				rundata = temp
			}

			returnValue = "Alright, I've reloaded my database from disk."
		case "list_names":
			person, _, _ := rundata.getPersonFromAlias(sanitize(command)[indexOf("for", sanitize(command))+1])
			returnValue = "That user is known as " + toEnglishList(person.Names)
		case "play_music":
			returnValue = getRandomString(rundata.Intents[intent].Responses)
		case "start_voice_connection":
			id, err := getCurrentVoiceChannel(mem)
			if err != nil && err.Error() == "person not in voice channel" {
				returnValue = getRandomString(rundata.Errors["user_not_in_voice_channel"])
				break
			}
			_, prs := spcontrol[mem.GuildID]
			if prs {
				spcontrol[mem.GuildID] <- "terminate"
			}
			go voiceService(voicedata{session: dclient, guildID: mem.GuildID, vChannelID: id, announceChan: spcontrolchan, tChannelID: channelID})
			returnValue = getRandomString(rundata.Intents[intent].Responses)
		case "end_voice_connection":
			spcontrol["vconn "+mem.GuildID] <- "terminate"
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
				go tempChannelManager(voicedata{session: dclient, guildID: mem.GuildID, vChannelName: channame, creatorID: mem.User.ID, announceChan: spcontrolchan})
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

func restart() {
	cleanup(false)
	go main()
}

func cleanup(exit bool) {
	fmt.Println("Sending disconnect signal...")
	for _, c := range spcontrol {
		c <- "terminate"
	}
	spcontrol = nil
	dclient.Close()
	dclient = nil
	fmt.Println("Saving...")
	rundata.save("./data/config.json")
	rundata = nil
	fmt.Println("Done.")
	if exit {
		os.Exit(0)
	}
}

func shutdown() { //Shutdown the discord connection and save data
	cleanup(true)
}
