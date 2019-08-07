package irc

import (
	"errors"
	"fmt"
	"time"

	"github.com/djdduty/ttv-log/config"
	"github.com/olivere/elastic/v7"
)

//Message ...
type Message struct {
	User      string
	Message   string
	Channel   string
	Timestamp time.Time
}

// CreateElasticFlusher ...
func CreateElasticFlusher(connector *config.ElasticConnector, flushInterval time.Duration) (chan Message, chan bool, error) {
	input := make(chan Message)
	quit := make(chan bool)

	client := connector.GetClient()
	ctx := connector.GetContext()

	go func() {
		bulk := client.Bulk().Index("twitch").Type("message")

		go func() {
			for {
				numActions := bulk.NumberOfActions()
				if numActions > 0 {
					res, err := bulk.Do(ctx)
					if err != nil {
						panic(err)
					}
					fmt.Printf("Flushed %d messages\n", numActions)
					if res.Errors {
						// Look up the failed documents with res.Failed(), and e.g. recommit
						panic(errors.New("bulk commit failed"))
					}
				}
				select {
				case <-time.After(flushInterval):
				case <-quit:
					return
				}
			}
		}()

		for {
			select {
			case message := <-input:
				bulk.Add(elastic.NewBulkIndexRequest().Doc(message))
			case <-quit:
				// Commit the final batch before exiting
				if bulk.NumberOfActions() > 0 {
					fmt.Printf("Attempting to flush %d messages\n", bulk.NumberOfActions())
					_, err := bulk.Do(ctx)
					if err != nil {
						panic(err)
					}
				}
				return
			}
		}
	}()

	return input, quit, nil
}
