package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var (
	GuildID        = flag.String("guild", "", "Test guild ID. If not passed - bot registers commands globally")
	BotToken       = flag.String("token", "", "Bot access token")
	RemoveCommands = flag.Bool("rmcmd", true, "Remove all commands after shutdowning or not")
)

var roles map[string]Role

type Role struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

const (
	Townsfolk int = 3447003
	Outsider  int = 1752220
	Minion    int = 15105570
	Demon     int = 15548997
	Fabled    int = 15844367
	Traveler  int = 10181046
)

var s *discordgo.Session

func init() { flag.Parse() }

func init() {
	var err error
	s, err = discordgo.New("Bot " + *BotToken)
	if err != nil {
		log.Fatalf("Invalid bot parameters: %v", err)
	}
}

var (
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "role",
			Description: "Get information about the requested role, with possible jinxes.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "role",
					Description: "The role you want information on.",
					Required:    true,
				},
			},
		},
		{
			Name:        "reload-roles",
			Description: "Reload role information.",
		},
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"role": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			options := i.ApplicationCommandData().Options

			optionMap := make(map[string]*discordgo.ApplicationCommandInteractionDataOption, len(options))
			for _, opt := range options {
				optionMap[opt.Name] = opt
			}

			var roleOpt string = strings.ToLower(optionMap["role"].StringValue())
			role, exists := roles[roleOpt]
			if !exists {
				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Content: fmt.Sprintf("No role that matches '%v' found. Try again with a different role.", optionMap["role"].StringValue()),
					},
				})
			} else {
				var color int

				switch role.Type {
				default:
				case "townsfolk":
					color = Townsfolk
					break
				case "outsider":
					color = Outsider
					break
				case "minion":
					color = Minion
					break
				case "demon":
					color = Demon
					break
				case "fabled":
					color = Fabled
					break
				case "traveler":
					color = Traveler
					break
				}

				s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
					Type: discordgo.InteractionResponseChannelMessageWithSource,
					Data: &discordgo.InteractionResponseData{
						Embeds: []*discordgo.MessageEmbed{
							{
								Type:        discordgo.EmbedTypeRich,
								Title:       role.Name,
								Color:       color,
								Description: role.Description,
								Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: role.Icon},
							},
						},
					},
				})
			}
		},
		"reload-roles": func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			loadRoles()
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("Reloaded roles, now indexing %v roles!", len(roles)),
				},
			})
		},
	}
)

func init() {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})
}

func main() {

	loadRoles()

	s.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Printf("Logged in as: %v#%v", s.State.User.Username, s.State.User.Discriminator)
	})
	err := s.Open()
	if err != nil {
		log.Fatalf("Cannot open the session: %v", err)
	}

	log.Println("Adding commands...")
	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, *GuildID, v)
		if err != nil {
			log.Panicf("Cannot create '%v' command: %v", v.Name, err)
		}
		registeredCommands[i] = cmd
	}

	defer s.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	log.Println("Press Ctrl+C to exit")
	<-stop

	// if *RemoveCommands {
	// 	log.Println("Removing commands...")

	// 	for _, v := range registeredCommands {
	// 		err := s.ApplicationCommandDelete(s.State.User.ID, *GuildID, v.ID)
	// 		if err != nil {
	// 			log.Panicf("Cannot delete '%v' command: %v", v.Name, err)
	// 		}
	// 	}
	// }

	log.Println("Gracefully shutting down.")
}

func loadRoles() {
	jsonFile, errJ := os.Open("roleData.json")
	if errJ != nil {
		log.Panic(errJ)
	}

	defer jsonFile.Close()

	byteValue, _ := io.ReadAll(jsonFile)

	var roleData []Role

	json.Unmarshal(byteValue, &roleData)

	roles = map[string]Role{}

	for _, e := range roleData {
		roles[e.ID] = e
	}

	log.Printf("Loaded %v roles", len(roleData))
}
