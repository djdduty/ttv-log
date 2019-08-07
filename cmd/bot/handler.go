package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/djdduty/ttv-log/config"
	"github.com/djdduty/ttv-log/irc"
	"github.com/spf13/cobra"
)

type channel struct {
	Name string `json:"name"`
}

type stream struct {
	Channel channel `json:"channel"`
}

type streamResponse struct {
	Total   int      `json:"_total"`
	Streams []stream `json:"streams"`
}

// AppendIfMissing adds element to slice if missing
func AppendIfMissing(slice []string, val string) []string {
	for _, ele := range slice {
		if ele == val {
			return slice
		}
	}
	return append(slice, val)
}

// RunBot starts the elastic goroutine to start queue flush and the IRC bot
func RunBot(c *config.Config) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		run(c)
	}
}

func run(config *config.Config) {
	// Start by getting the top 500 streams currently live, won't come out to exactly 500 thanks to duplicates
	url := "https://api.twitch.tv/kraken/streams/"
	httpClient := http.Client{
		Timeout: time.Second * 30, // Maximum of 2 secs
	}
	streams := []string{}
	streams = append(streams, config.StreamWhilelist...)
	numStreams := 1000

	for i := 0; i < numStreams/100; i = i + 1 {
		offset := i * 100
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s?limit=100&offset=%d", url, offset), nil)
		if err != nil {
			log.Fatal(err)
		}

		req.Header.Set("Accept", "application/vnd.twitchtv.v5+json")
		req.Header.Set("Client-ID", config.TwitchClientID)

		res, getErr := httpClient.Do(req)
		if getErr != nil {
			log.Fatal(getErr)
		}

		body, readErr := ioutil.ReadAll(res.Body)
		if readErr != nil {
			log.Fatal(readErr)
		}

		streamData := streamResponse{}
		jsonErr := json.Unmarshal(body, &streamData)
		if jsonErr != nil {
			log.Fatal(jsonErr)
		}

		for _, stream := range streamData.Streams {
			streams = AppendIfMissing(streams, stream.Channel.Name)
		}
	}
	fmt.Printf("Got list of %d streams\n", len(streams))

	sigs := make(chan os.Signal, 1)

	//signal.Notify registers the given channel to receive notifications of the specified signals.
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	messageChan, quitChan, err := irc.CreateElasticFlusher(config.Context().ElasticConnection, 1*time.Second)

	if err != nil {
		panic(err)
	}

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		quitChan <- true
	}()

	go irc.StartGoIRC(
		messageChan,       // channel for IRC to feed messages in to
		quitChan,          // chanel for done signal or disconnect
		config.TwitchUser, // twitch IRC username
		config.TwitchPass, // twitch IRC password "oauth:..."
		streams,           // twitch live streams to join
	)

	<-quitChan
}
