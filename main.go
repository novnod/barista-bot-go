package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/novnod/barista-bot/config"
	"github.com/novnod/barista-bot/parser"
	"github.com/novnod/barista-bot/repo"
)

var (
	dailyRepo *repo.DailyRepository
)

func main() {
	// Load bot configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Open a connection to the database
	sqlDB, err := repo.InitDBConnection(cfg.DatabaseUrl)
	if err != nil {
		log.Fatal(err)
	}

	dailyRepo = repo.InitDailyRepository(sqlDB)

	// Load only easy SGF problems
	pg := parser.GoParser{}
	if err := pg.LoadProblems("./files/cho-easy.sgf"); err != nil {
		log.Fatalf("failed to load easy problems: %v", err)
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
	dg.AddHandler(onMessage)
	dg.AddHandler(onMessageReaction)

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

func onMessage(s *discordgo.Session, m *discordgo.MessageCreate) {
	log.Printf("Message from: %s and they said: %s", m.Author.Username, m.Content)
}

func onMessageReaction(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	log.Printf("Message from: %s and they said: %s", r.MessageID, r.Emoji.Name)
}

func commandHandler(pg *parser.GoParser) any {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			switch i.ApplicationCommandData().Name {
			case "test":
				respond(s, i, "Test response")

			case "daily":
				handleDaily(s, i, pg)

			case "edit_daily":
				handleEditDaily(s, i)

			default:
				log.Printf("unknown command: %s", i.ApplicationCommandData().Name)
			}
		case discordgo.InteractionModalSubmit:
			handleModalSubmit(s, i)
		}
	}
}

func registerCommands(s *discordgo.Session, appID, guildID string) {
	commands := []*discordgo.ApplicationCommand{
		{Name: "test", Description: "Just a test"},
		{Name: "daily", Description: "Starts a daily Go problem thread"},
		{Name: "edit_daily", Description: "Edit daily settings"},
	}
	for _, cmd := range commands {
		if _, err := s.ApplicationCommandCreate(appID, guildID, cmd); err != nil {
			log.Printf("could not create command '%s': %v", cmd.Name, err)
		}
	}
}

func handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Saving updated time",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		respondError(s, i, "error saving updated time")
	}

	data := i.ModalSubmitData()

	if !strings.HasPrefix(data.CustomID, "edit_daily") {
		return
	}

	daily_time := data.Components[0].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	channel_id := data.Components[1].(*discordgo.ActionsRow).Components[0].(*discordgo.TextInput).Value
	log.Printf("the submitted updated time is %s", daily_time)
	log.Printf("the channel id is %s", channel_id)
	err = dailyRepo.SetConfig(i.GuildID, channel_id, daily_time)
	if err != nil {
		log.Printf("error occured updating config: %s", err)
		respondError(s, i, "error occured updating config")
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

func handleEditDaily(s *discordgo.Session, i *discordgo.InteractionCreate) {
	isStaff := false
	var staffID string

	config, err := dailyRepo.GetConfig(i.GuildID)
	if err != nil && err != sql.ErrNoRows {
		respondError(s, i, "error occured retreiving settings from db: "+err.Error())
		return
	}

	if err == sql.ErrNoRows {
		config = &repo.DailyConfig{}
	}

	guild, err := s.Guild(i.GuildID)
	if err != nil {
		respondError(s, i, "an internal server error occured getting the guild information")
		return
	}

	for _, role := range guild.Roles {
		if role.Name == "Staff" {
			staffID = role.ID
		}
	}
	log.Printf("%s roles are: %v", i.Member.User.GlobalName, i.Member.Roles)
	for _, id := range i.Member.Roles {
		if id == staffID {
			isStaff = true
		}
	}

	if !isStaff {
		respondError(s, i, fmt.Sprintf("not a staff memeber"))
		return
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: "edit_daily",
			Title:    "Edit Daily",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "daily-time",
							Label:       "time for dailies?",
							Style:       discordgo.TextInputShort,
							Placeholder: "Daily start time (HH:MM) tz",
							Value:       config.TimeHHMM,
							Required:    true,
							MaxLength:   10,
							MinLength:   4,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "daily-channel",
							Label:       "Channel for dailies (#mention or ID)",
							Style:       discordgo.TextInputShort,
							Placeholder: "#barista-bot-prototype or 1351203956591693921",
							Value:       config.ChannelID,
							Required:    true,
							MinLength:   1,
							MaxLength:   100,
						},
					},
				}},
		},
	})

	if err != nil {
		log.Printf("error is: %s", err)
		respondError(s, i, "an error occured with submitting your changes")
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
