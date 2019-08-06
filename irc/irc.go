package irc

import (
	"crypto/tls"
	"fmt"
	"time"

	irc "github.com/fluffle/goirc/client"
)

// StartGoIRC ...
func StartGoIRC(messageChan chan Message, quitChan chan bool, username string, password string, streams []string) {
	cfg := irc.NewConfig(username)
	cfg.Pass = password
	cfg.SSL = true
	cfg.SSLConfig = &tls.Config{ServerName: "irc.chat.twitch.tv"}
	cfg.Server = "irc.chat.twitch.tv:6697"
	cfg.Flood = true
	c := irc.Client(cfg)

	c.HandleFunc(irc.CONNECTED, func(conn *irc.Conn, line *irc.Line) {
		go func() {
			numJoined := 0
			for _, streamName := range streams {
				conn.Join(fmt.Sprintf("#%s", streamName))
				fmt.Printf("Sent JOIN for %s\n", streamName)
				numJoined = numJoined + 1
				if numJoined%50 == 0 {
					fmt.Printf("Sleeping for 15s to avoid JOIN rate limit Joined %d / %d\n", numJoined, len(streams))
					time.Sleep(15 * time.Second)
				}
			}
			fmt.Printf("Joined all %d stream channels (input: %d)\n", numJoined, len(streams))
		}()
	})

	c.HandleFunc(irc.DISCONNECTED, func(conn *irc.Conn, line *irc.Line) {
		fmt.Println("Disconnected from IRC")
		quitChan <- true
	})

	c.HandleFunc(irc.PRIVMSG, func(conn *irc.Conn, line *irc.Line) {
		//log.Println("Received: ", line.Target(), line.Nick, line.Text())
		messageChan <- Message{User: line.Nick, Message: line.Text(), Timestamp: time.Now().UTC(), Channel: line.Target()}
	})

	// Tell client to connect.
	if err := c.Connect(); err != nil {
		fmt.Printf("Connection error: %s\n", err.Error())
	}

	<-quitChan
	c.Close()
}
