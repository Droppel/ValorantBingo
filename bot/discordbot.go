package bot

import (
	"Bingo/bingo"
	"Bingo/config"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

var (
	Commands = []*discordgo.ApplicationCommand{
		{
			Name:        "create",
			Description: "Creates a new bingo",
			Options: []*discordgo.ApplicationCommandOption{

				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "bingo-type",
					Description: "Kind of bingo",
					Required:    true,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{
							Name:  "valorant",
							Value: "valorant",
						},
					},
				},
			},
		},
		{
			Name:        "continue",
			Description: "Continues an existing bingo",

			Options: []*discordgo.ApplicationCommandOption{

				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "bingo-id",
					Description: "ID of the bingo",
					Required:    true,
				},
			},
		},
	}

	CommandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"create": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			userID := i.Member.User.ID

			bin, err := bingo.Create(userID, options[0].StringValue(), 25)
			if err != nil {
				log.WithError(err).Error("Error creating bingo")
				s.ChannelMessageSend(i.ChannelID, "Error")
				return
			}

			bingo.AddBingo(bin)

			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Bingo created successfully",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			if err != nil {
				log.WithError(err).Error("Error sending response")
				s.ChannelMessageSend(i.ChannelID, "Error")
				return
			}

			dmChannel, err := s.UserChannelCreate(userID)
			if err != nil {
				log.WithError(err).Error("Could not create Userchannel")
				return
			}
			s.ChannelMessageSend(dmChannel.ID, "Here is the link to your Bingo boards Management plane: http://droppel.ddns.net:8080/main/"+bin.Id+"/?pass="+bin.Password)

			msg, err := s.ChannelMessageSend(i.ChannelID, "Bingo created with id: "+bin.Id+". React with 🎫 to join.")
			MessageToBingo[msg.ID] = bin
			if err != nil {
				log.WithError(err).Error("Error sending the message")
				s.ChannelMessageSend(i.ChannelID, "Error")
				return
			}

			err = s.MessageReactionAdd(msg.ChannelID, msg.ID, "🎫")
			if err != nil {
				log.WithError(err).Error("Error reacting to the message")
				return
			}
		},
		"continue": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options
			jsonBingo, err := ioutil.ReadFile(config.Json.StoragePath + options[0].StringValue())
			if err != nil {
				log.WithError(err).Error("Error reading bingo")
				s.ChannelMessageSend(i.ChannelID, "Error")
				return
			}

			var bin *bingo.Bingo = &bingo.Bingo{}
			err = json.Unmarshal(jsonBingo, bin)
			if err != nil {
				log.WithError(err).Error("Error unmarshaling bingo")
				s.ChannelMessageSend(i.ChannelID, "Error")
				return
			}

			bingo.AddBingo(bin)

			msg, err := s.ChannelMessageSend(i.ChannelID, "Bingo continued with id: "+bin.Id+". React with 🎫 to join.")
			MessageToBingo[msg.ID] = bin
			if err != nil {
				log.WithError(err).Error("Error sending the message")
				s.ChannelMessageSend(i.ChannelID, "Error")
				return
			}

			err = s.MessageReactionAdd(msg.ChannelID, msg.ID, "🎫")
			if err != nil {
				log.WithError(err).Error("Error reacting to the message")
				return
			}
		},
	}
)

var (
	MessageToBingo map[string]*bingo.Bingo
)

func InitBot() {

	MessageToBingo = make(map[string]*bingo.Bingo)

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
	dg.AddHandler(reactionAdded)
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := CommandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
	dg.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		fmt.Printf("Logged in as: %v#%v\n", s.State.User.Username, s.State.User.Discriminator)
	})

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages + discordgo.IntentsGuildMessageReactions + discordgo.IntentsDirectMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		log.WithError(err).Error("error opening connection")
		return
	}

	log.Info("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(Commands))
	for i, v := range Commands {
		cmd, err := dg.ApplicationCommandCreate(dg.State.User.ID, "", v)
		if err != nil {
			log.Warnf("Cannot create '%s' command: %v", v.Name, err)
			return
		}
		registeredCommands[i] = cmd
		log.Infof("Added command: %s", v.Name)
	}

	// Wait here until CTRL-C or other term signal is received.
	log.Info("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	dg.Close()
	log.Println("Removing commands...")
	// // We need to fetch the commands, since deleting requires the command ID.
	// // We are doing this from the returned commands on line 375, because using
	// // this will delete all the commands, which might not be desirable, so we
	// // are deleting only the commands that we added.
	// registeredCommands, err := s.ApplicationCommands(s.State.User.ID, *GuildID)
	// if err != nil {
	// 	log.Fatalf("Could not fetch registered commands: %v", err)
	// }

	for _, v := range registeredCommands {
		err := dg.ApplicationCommandDelete(dg.State.User.ID, "", v.ID)
		if err != nil {
			log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
		}
		fmt.Printf("Removed command: %s", v.Name)
	}
}

func reactionAdded(s *discordgo.Session, rea *discordgo.MessageReactionAdd) {

	if rea.UserID == s.State.User.ID {
		return
	}

	if rea.Emoji.Name != "🎫" {
		return
	}

	bin := MessageToBingo[rea.MessageID]

	if bin == nil {
		return
	}

	dmChannel, err := s.UserChannelCreate(rea.UserID)
	if err != nil {
		log.WithError(err).Error("Could not create Userchannel")
		return
	}

	user, err := s.User(rea.UserID)
	if err != nil {
		return
	}

	board := bin.CreateBoard(rea.UserID, user.Username, config.Json.GameSettings.TotalRerolls)

	s.ChannelMessageSend(dmChannel.ID, "Here is a link to your Bingo board: http://droppel.ddns.net:8080/bingo/"+bin.Id+"/"+board.Id+"?pass="+board.Password)
}
