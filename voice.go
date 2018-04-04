package main

import (
	"errors"
	"time"

	"github.com/bwmarrin/discordgo"
)

type voicedata struct {
	session      *discordgo.Session
	guildID      string
	vChannelID   string
	vChannelName string
	tChannelID   string
	announceChan chan spdata
	creatorID    string
}

func tempChannelManager(data voicedata) {
	if data.session == nil || data.guildID == "" || data.vChannelName == "" || data.creatorID == "" || data.announceChan == nil {
		return
	}

	vchan, err := data.session.GuildChannelCreate(data.guildID, data.vChannelName, "voice")
	checkErr(err, "create temporary voice channel")
	vchan, err = data.session.ChannelEditComplex(vchan.ID, &discordgo.ChannelEdit{Bitrate: 32000})
	checkErr(err, "fix temporary channel bitrate")
	err = data.session.GuildMemberMove(data.guildID, data.creatorID, vchan.ID)
	checkErr(err, "user move to new channel")

	commandChan := make(chan string)
	data.announceChan <- spdata{command: "register", id: "temp_chan " + vchan.ID, channel: commandChan}

	time.Sleep(time.Duration(5) * time.Second)

	for len(getUsersInVoiceChannel(data.guildID, vchan.ID)) > 0 {
		select {
		case command := <-commandChan:
			switch command {
			case "terminate":
				break
			}
		default:
		}
	}

	_, err = data.session.ChannelDelete(vchan.ID)
	checkErr(err, "delete temporary voice channel")

	data.announceChan <- spdata{command: "delete", id: "temp_chan " + vchan.ID}
}

func voiceService(data voicedata) {
	if data.session == nil || data.guildID == "" || data.vChannelID == "" || data.announceChan == nil || data.tChannelID == "" {
		return
	}

	dvclient, err := data.session.ChannelVoiceJoin(data.guildID, data.vChannelID, false, false)
	if err != nil {
		if err.Error() == "timeout waiting for voice" {
			data.session.ChannelMessageSend(data.tChannelID, "Sorry, I couldn't connect. Try again later.")
			return
		}
	}
	checkErr(err, "connect to voice channel")

	commandChan := make(chan string)
	data.announceChan <- spdata{command: "register", id: "vconn " + data.guildID, channel: commandChan}

	for exitFlag := false; exitFlag == false; {
		select {
		case command := <-commandChan:
			switch command {
			case "terminate":
				exitFlag = true
			}
		}
	}

	dvclient.Disconnect()
	dvclient.Close()

	data.announceChan <- spdata{command: "delete", id: "vconn " + data.guildID}
}

func getCurrentVoiceChannel(mem *discordgo.Member) (string, error) {
	guild, err := dclient.Guild(mem.GuildID)
	checkErr(err, "get guild for voice channels")
	for _, instance := range guild.VoiceStates {
		if instance.UserID == mem.User.ID {
			return instance.ChannelID, nil
		}
	}
	return "", errors.New("person not in voice channel")
}

func getUsersInVoiceChannel(guildid, chanid string) []string {
	guild, err := dclient.Guild(guildid)
	checkErr(err, "get guild for voice channels")

	users := []string{}

	for _, instance := range guild.VoiceStates {
		if instance.ChannelID == chanid {
			users = append(users, instance.UserID)
		}
	}

	return users
}
