package main

import(
	"fmt"
	"os"
	"context"
	"os/signal"
	"syscall"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/gateaway"

)

func main(){
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		fmt.Println("DISCORD_TOKEN environment variable not set")
		return
	}
	client, err := disgo.New(token, bot.WithGatewayConfigOpts(gateway.WithIntents(gateway.IntentGuilds),
		),
		)
	if err != nil {
		fmt.Println("Failed to create Discord client: ", err)
		return
	}
	defer client.close(context.TODO())
	if err := client.OpenGateway(context.TODO()); err != nil{
		fmt.Println("error connecting:", err)
		return
	}
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
}