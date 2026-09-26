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

)

var commands = []discord.ApplicationCommandCreate{
	discord.SlashCommandCreate{
		Name:        "ping",
		Description: "Check if the bot is alive",
	},
}

func main(){
	token := os.Getenv("DISCORD_TOKEN")
	guildID := snowflake.GetEnv("GUILD_ID")
	if token == "" {
		fmt.Println("DISCORD_TOKEN environment variable not set")
		return
	}
	client, err := disgo.New(token, bot.WithGatewayConfigOpts(gateway.WithIntents(gateway.IntentGuilds),
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

	if data.CommandName() == "ping" {
		err := event.CreateMessage(
			discord.NewMessageCreate().WithContent("pong 🏓"),
		)
		if err != nil {
			fmt.Println("error replying:", err)
		}
	}
}