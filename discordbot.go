package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

var (
	messageToBingo map[string]*Bingo
)

func InitBot() {

	messageToBingo = make(map[string]*Bingo)

	authtoken, err := ioutil.ReadFile("authtoken.txt")
	if err != nil {
		log.WithError(err).Error("Could no load authtoken")
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + string(authtoken))
	if err != nil {
		log.WithError(err).Error("error creating Discord session")
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)
	dg.AddHandler(reactionAdded)

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages + discordgo.IntentsGuildMessageReactions + discordgo.IntentsDirectMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.WithError(err).Error("error opening connection")
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	log.Info("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	command := strings.Split(m.Content, " ")
	length := len(command)

	if command[0] != "!bingo" || length < 2 {
		return
	}

	switch command[1] {
	case "help":
		s.ChannelMessageSend(m.ChannelID, "No")
		break
	case "create":
		if length < 3 {
			return
		}
		bin, err := Create(command[2], 25)
		if err != nil {
			log.WithError(err).Error("Error creating bingo")
			s.ChannelMessageSend(m.ChannelID, "Error")
			return
		}

		AddBingo(bin)

		msg, err := s.ChannelMessageSend(m.ChannelID, "Bingo created with id: "+bin.Id+". React with 🎫 to join.")
		messageToBingo[msg.ID] = bin
		if err != nil {
			log.WithError(err).Error("Error sending the message")
			s.ChannelMessageSend(m.ChannelID, "Error")
			return
		}

		err = s.MessageReactionAdd(msg.ChannelID, msg.ID, "🎫")
		if err != nil {
			log.WithError(err).Error("Error reacting to the message")
			return
		}
		break
	case "continue":
		if length < 3 {
			return
		}

		jsonBingo, err := ioutil.ReadFile(command[2])
		if err != nil {
			log.WithError(err).Error("Error reading bingo")
			s.ChannelMessageSend(m.ChannelID, "Error")
			return
		}

		var bin *Bingo = &Bingo{}
		err = json.Unmarshal(jsonBingo, bin)
		if err != nil {
			log.WithError(err).Error("Error unmarshaling bingo")
			s.ChannelMessageSend(m.ChannelID, "Error")
			return
		}

		AddBingo(bin)

		msg, err := s.ChannelMessageSend(m.ChannelID, "Bingo continued with id: "+bin.Id+". React with 🎫 to join.")
		messageToBingo[msg.ID] = bin
		if err != nil {
			log.WithError(err).Error("Error sending the message")
			s.ChannelMessageSend(m.ChannelID, "Error")
			return
		}

		err = s.MessageReactionAdd(msg.ChannelID, msg.ID, "🎫")
		if err != nil {
			log.WithError(err).Error("Error reacting to the message")
			return
		}
		break
	}
}

func reactionAdded(s *discordgo.Session, rea *discordgo.MessageReactionAdd) {

	if rea.UserID == s.State.User.ID {
		return
	}

	if rea.Emoji.Name != "🎫" {
		return
	}

	bin := messageToBingo[rea.MessageID]

	if bin == nil {
		return
	}

	dmChannel, err := s.UserChannelCreate(rea.UserID)
	if err != nil {
		log.WithError(err).Error("Could not create Userchannel")
		return
	}

	board := bin.CreateBoard(rea.UserID)

	s.ChannelMessageSend(dmChannel.ID, "Here is a link to your Bingo board: http://droppel.ddns.net:8080/bingo/"+bin.Id+"/"+board.Id)
}
