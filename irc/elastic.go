package irc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/olivere/elastic/v7"
)

//Message ...
type Message struct {
	User      string
	Message   string
	Channel   string
	Timestamp time.Time
}

const mapping = `
{
	"mappings":{
		"properties":{
			"User":{
				"type":"keyword"
			},
			"Message":{
				"type":"text",
				"store": true,
				"fielddata": true
			},
			"Channel":{
				"type":"text",
				"store": true,
				"fielddata": true
			},
			"Timestamp":{
				"type":"date"
			}
		}
	}
}`

// CreateElasticFlusher ...
func CreateElasticFlusher(flushInterval time.Duration) (chan Message, chan bool, error) {
	input := make(chan Message)
	quit := make(chan bool)

	ctx := context.Background()
	client, err := elastic.NewSimpleClient(
		elastic.SetURL(os.Getenv("TTV_ELASTIC_URL")),
		elastic.SetBasicAuth(os.Getenv("TTV_ELASTIC_USER"), os.Getenv("TTV_ELASTIC_PASS")),
	)
	if err != nil {
		return nil, nil, err
	}

	info, code, err := client.Ping(os.Getenv("TTV_ELASTIC_URL")).Do(ctx)
	if err != nil {
		return nil, nil, err
	}

	fmt.Printf("Elasticsearch returned with code %d and version %s\n", code, info.Version.Number)

	// Getting the ES version number is quite common, so there's a shortcut
	esversion, err := client.ElasticsearchVersion(os.Getenv("TTV_ELASTIC_URL"))
	if err != nil {
		// Handle error
		return nil, nil, err
	}
	fmt.Printf("Elasticsearch version %s\n", esversion)

	// Use the IndexExists service to check if a specified index exists.
	exists, err := client.IndexExists("twitch").Do(ctx)
	if err != nil {
		// Handle error
		return nil, nil, err
	}
	if !exists {
		// Create a new index.
		createIndex, err := client.CreateIndex("twitch").Body(mapping).Do(ctx)
		if err != nil {
			// Handle error
			panic(err)
		}
		if !createIndex.Acknowledged {
			// Not acknowledged
		}
	} else {
		//Delete an index.
		deleteIndex, err := client.DeleteIndex("twitch").Do(ctx)
		if err != nil {
			// Handle error
			panic(err)
		}
		if !deleteIndex.Acknowledged {
			// Not acknowledged
		}
	}

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
					_, err = bulk.Do(ctx)
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
