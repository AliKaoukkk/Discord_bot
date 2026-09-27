package main

import(
	"fmt"
	"os"
	"context"
	"os/signal"
	"syscall"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/snowflake/v2"
	"time"

	"github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/godave/golibdave"

)
var commands = []discord.ApplicationCommandCreate{
	discord.SlashCommandCreate{
		Name:        "ping",
		Description: "Check if the bot is alive",
	},
	discord.SlashCommandCreate{
		Name:        "join",
		Description: "Join a voice channel",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionChannel{
				Name:         "channel",
				Description:  "The voice channel to join",
				Required:     true,
				ChannelTypes: []discord.ChannelType{discord.ChannelTypeGuildVoice},
			},
		},
	},
	discord.SlashCommandCreate{
		Name:        "leave",
		Description: "Leave the voice channel",
	},
}

func main(){
	token := os.Getenv("DISCORD_TOKEN")
	guildID := snowflake.GetEnv("GUILD_ID")
	if token == "" {
		fmt.Println("DISCORD_TOKEN environment variable not set")
		return
	}
	client, err := disgo.New(token,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(gateway.IntentGuilds, gateway.IntentGuildVoiceStates),
		),
		bot.WithVoiceManagerConfigOpts(
			voice.WithDaveSessionCreateFunc(golibdave.NewSession),
		),
		bot.WithEventListenerFunc(onCommand),
	)
	if err != nil {
		fmt.Println("Failed to create Discord client: ", err)
		return
	}
	if _, err := client.Rest.SetGuildCommands(client.ApplicationID, guildID, commands); err != nil {
		fmt.Println("error registering commands:", err)
		return
	}
	defer client.Close(context.TODO())
	if err := client.OpenGateway(context.TODO()); err != nil{
		fmt.Println("error connecting:", err)
		return
	}
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
}

func onCommand(event *events.ApplicationCommandInteractionCreate) {
	data := event.SlashCommandInteractionData()
	guildID := *event.GuildID()

	switch data.CommandName() {
	case "ping":
		event.CreateMessage(discord.NewMessageCreate().WithContent("pong 🏓"))

	case "join":
		channelID := data.Snowflake("channel")
		event.CreateMessage(discord.NewMessageCreate().WithContent("Joining 🔊"))
		go joinChannel(event.Client(), guildID, channelID)

	case "leave":
		event.CreateMessage(discord.NewMessageCreate().WithContent("Leaving 👋"))
		go leaveChannel(event.Client(), guildID)
	}
}
func joinChannel(client *bot.Client, guildID, channelID snowflake.ID) {
	conn := client.VoiceManager.CreateConn(guildID)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := conn.Open(ctx, channelID, false, false); err != nil {
		fmt.Println("error joining voice:", err)
		return
	}
	fmt.Println("joined voice channel", channelID)
}

func leaveChannel(client *bot.Client, guildID snowflake.ID) {
	conn := client.VoiceManager.GetConn(guildID)
	if conn == nil {
		fmt.Println("not in a voice channel")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn.Close(ctx)
	fmt.Println("left voice channel")
}