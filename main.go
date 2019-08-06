package main

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

	"github.com/djdduty/ttv-logbot/irc"
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

func main() {
	// Start by getting the top 500 streams currently live, won't come out to exactly 500 thanks to duplicates
	url := "https://api.twitch.tv/kraken/streams/"
	httpClient := http.Client{
		Timeout: time.Second * 2, // Maximum of 2 secs
	}
	var streams []string

	for i := 0; i < 5; i = i + 1 {
		offset := i * 100
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s?limit=100&offset=%d", url, offset), nil)
		if err != nil {
			log.Fatal(err)
		}

		req.Header.Set("Accept", "application/vnd.twitchtv.v5+json")
		req.Header.Set("Client-ID", os.Getenv("TTV_CLIENTID"))

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

	messageChan, quitChan, err := irc.CreateElasticFlusher(
		"http://127.0.0.1:9200", // elasticsearch host
		1*time.Second,           // elasticsearch flush interval
	)

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
		messageChan,               // channel for IRC to feed messages in to
		quitChan,                  // chanel for done signal or disconnect
		os.Getenv("TTV_USERNAME"), // twitch IRC username
		os.Getenv("TTV_PASSWORD"), // twitch IRC password "oauth:..."
		streams,                   // twitch live streams to join
	)

	<-quitChan
}
