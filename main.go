package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/novnod/barista-bot/config"
	"github.com/novnod/barista-bot/parser"
)

func main() {
	// Load only easy SGF problems
	pg := parser.GoParser{}
	if err := pg.LoadProblems("./files/cho-easy.sgf"); err != nil {
		log.Fatalf("failed to load easy problems: %v", err)
	}

	// Load bot configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize Discord session
	dg, err := discordgo.New("Bot " + cfg.BotToken)
	if err != nil {
		log.Fatalf("error creating Discord session: %v", err)
	}
	dg.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions

	// Register event handlers
	dg.AddHandler(onReady)
	dg.AddHandler(commandHandler(&pg))

	// Open WebSocket connection
	if err := dg.Open(); err != nil {
		log.Fatalf("could not open WebSocket connection: %v", err)
	}
	defer dg.Close()

	// Fetch application (bot) ID for command registration
	botUser, err := dg.User("@me")
	if err != nil {
		log.Fatalf("could not fetch bot user: %v", err)
	}

	// Register slash commands
	registerCommands(dg, botUser.ID, "1314429177230921840")

	log.Println("Bot is now running. Press Ctrl+C to exit.")

	// Wait for interrupt signal to gracefully shut down
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt)
	<-sig
}

func onReady(s *discordgo.Session, _ *discordgo.Ready) {
	log.Printf("Logged in as %s#%s", s.State.User.Username, s.State.User.Discriminator)
}

func commandHandler(pg *parser.GoParser) interface{} {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.ApplicationCommandData().Name {
		case "test":
			respond(s, i, "Test response")

		case "daily":
			handleDaily(s, i, pg)

		default:
			log.Printf("unknown command: %s", i.ApplicationCommandData().Name)
		}
	}
}

func registerCommands(s *discordgo.Session, appID, guildID string) {
	commands := []*discordgo.ApplicationCommand{
		{Name: "test", Description: "Just a test"},
		{Name: "daily", Description: "Starts a daily Go problem thread"},
	}
	for _, cmd := range commands {
		if _, err := s.ApplicationCommandCreate(appID, guildID, cmd); err != nil {
			log.Printf("could not create command '%s': %v", cmd.Name, err)
		}
	}
}

func handleDaily(s *discordgo.Session, i *discordgo.InteractionCreate, pg *parser.GoParser) {
	// Ensure invoked in main channel, not inside a thread
	channel, err := s.Channel(i.ChannelID)
	if err != nil || channel.IsThread() {
		respond(s, i, "Please use this command in a main channel.")
		return
	}

	// Create a thread for the user
	threadName := fmt.Sprintf("%s's Daily Thread", i.Member.User.Username)
	thread, err := s.ThreadStart(i.ChannelID, threadName, discordgo.ChannelTypeGuildPublicThread, 1440)
	if err != nil {
		respondError(s, i, "could not create thread")
		return
	}

	// Acknowledge interaction
	respond(s, i, "Daily practice thread created!")

	// Determine today's problem index deterministically
	days := time.Now().UTC().Unix() / 86400
	problems := pg.Problems
	count := len(problems)
	if count == 0 {
		respondError(s, i, "no problems available")
		return
	}
	idx := int(days % int64(count))
	prob := problems[idx]

	// Render problem image
	imgPath, err := parser.RenderProblem(prob, "./out", 800, 40)
	if err != nil {
		respondError(s, i, fmt.Sprintf("failed to render problem: %v", err))
		return
	}

	// Open image file for sending
	file, err := os.Open(imgPath)
	if err != nil {
		respondError(s, i, fmt.Sprintf("failed to open image: %v", err))
		return
	}
	defer file.Close()

	// Send problem image to thread
	if _, err := s.ChannelFileSend(thread.ID, filepath.Base(imgPath), file); err != nil {
		respondError(s, i, fmt.Sprintf("failed to send image: %v", err))
	}
}

func respond(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: msg},
	})
}

func respondError(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	log.Print(msg)
	respond(s, i, msg)
}
