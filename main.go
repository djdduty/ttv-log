package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/djdduty/ttv-logbot/irc"
)

func main() {
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
	)

	<-quitChan
}
