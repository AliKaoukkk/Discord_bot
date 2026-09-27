package main

import(
	"fmt"
	"os"
	"context"
	"os/signal"
	"syscall"
	"os/exec"

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
	discord.SlashCommandCreate{
		Name:        "play",
		Description: "Play audio from a YouTube link",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionString{
				Name:        "url",
				Description: "YouTube link",
				Required:    true,
			},
		},
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

	case "play":
		url := data.String("url")
		if event.Client().VoiceManager.GetConn(guildID) == nil {
			event.CreateMessage(discord.NewMessageCreate().WithContent("Use /join first 🔊"))
			return
		}
		event.CreateMessage(discord.NewMessageCreate().WithContent("Loading 🎵"))
		go playSong(event.Client(), guildID, url)
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

func playSong(client *bot.Client, guildID snowflake.ID, url string) {
	conn := client.VoiceManager.GetConn(guildID)
	if conn == nil {
		return
	}

	ytdlp := exec.Command("yt-dlp",
		"-f", "bestaudio",
		"--no-playlist",
		"--quiet",
		"-o", "-",
		"--", url,
	)
	ffmpeg := exec.Command("ffmpeg",
		"-loglevel", "error",
		"-i", "pipe:0",
		"-c:a", "libopus",
		"-b:a", "128k",
		"-ar", "48000",
		"-ac", "2",
		"-frame_duration", "20",
		"-f", "ogg",
		"pipe:1",
	)

	ytOut, err := ytdlp.StdoutPipe()
	if err != nil {
		fmt.Println("yt-dlp pipe error:", err)
		return
	}
	ffmpeg.Stdin = ytOut

	ffOut, err := ffmpeg.StdoutPipe()
	if err != nil {
		fmt.Println("ffmpeg pipe error:", err)
		return
	}

	ytdlp.Stderr = os.Stderr
	ffmpeg.Stderr = os.Stderr

	if err := ytdlp.Start(); err != nil {
		fmt.Println("yt-dlp start error:", err)
		return
	}
	if err := ffmpeg.Start(); err != nil {
		fmt.Println("ffmpeg start error:", err)
		return
	}

	provider := NewOggOpusProvider(ffOut)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.SetSpeaking(ctx, voice.SpeakingFlagMicrophone); err != nil {
		fmt.Println("speaking error:", err)
		return
	}
	conn.SetOpusFrameProvider(provider)
	fmt.Println("playing:", url)

	<-provider.done

	ffmpeg.Wait()
	ytdlp.Wait()
	fmt.Println("finished:", url)
}