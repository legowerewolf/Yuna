package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

type voicedata struct {
	session        *discordgo.Session
	guildID        string
	channelID      string
	channelName    string
	commandChannel chan string
	returnChannel  chan string
	creatorID      string
}

func tempChannelManager(data voicedata) {
	if data.session == nil || data.guildID == "" || data.channelName == "" || data.creatorID == "" || data.commandChannel == nil {
		if data.returnChannel != nil {
			data.returnChannel <- "error not enough data"
		}
		fmt.Println("Not enough data.")
		return
	}

	vchan, err := data.session.GuildChannelCreate(data.guildID, data.channelName, "voice")
	checkErr(err, "create temporary voice channel")
	err = data.session.GuildMemberMove(data.guildID, data.creatorID, vchan.ID)
	checkErr(err, "user move to new channel")

	time.Sleep(time.Duration(5) * time.Second)

	for len(getUsersInVoiceChannel(data.guildID, vchan.ID)) > 0 {
		select {
		case command := <-data.commandChannel:
			switch command {
			case "disconnect":
				break
			}
		default:
		}
	}

	_, err = data.session.ChannelDelete(vchan.ID)
	checkErr(err, "delete temporary voice channel")

	if data.returnChannel != nil {
		data.returnChannel <- "done"
	}
}

func voiceService(data voicedata) {
	if data.session == nil || data.guildID == "" || data.channelID == "" || data.commandChannel == nil {
		if data.returnChannel != nil {
			data.returnChannel <- "error not enough data"
		}
		return
	}

	dvclient, err := data.session.ChannelVoiceJoin(data.guildID, data.channelID, false, false)
	checkErr(err, "connect to voice channel")

	for exitFlag := false; exitFlag == false; {
		select {
		case command := <-data.commandChannel:
			switch command {
			case "disconnect":
				exitFlag = true
			}
		}
	}

	dvclient.Disconnect()
	dvclient.Close()

	if data.returnChannel != nil {
		data.returnChannel <- "done"
	}
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
