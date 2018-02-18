package main

import "github.com/bwmarrin/discordgo"

func voiceService(dclient *discordgo.Session, guild string, channel string, control <-chan string) {
	dvclient, err := dclient.ChannelVoiceJoin(guild, channel, false, false)
	checkErr(err, "connect to voice channel")

	for exitFlag := false; exitFlag == false; {
		select {
		case command := <-control:
			switch command {
			case "disconnect":
				exitFlag = true
			}
		}
	}

	dvclient.Disconnect()
	dvclient.Close()
}
