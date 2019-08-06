package main

import (
	"context"
	"fmt"
	"time"

	"github.com/olivere/elastic/v7"
	irc "github.com/thoj/go-ircevent"
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
	"settings":{
		"number_of_shards": 1,
		"number_of_replicas": 0
	},
	"mappings":{
		"message":{
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
					"type":"text"
				},
				"Timestamp":{
					"type":"date"
				}
			}
		}
	}
}`

var streamers = []string{
	"shroud",
	"xqcow",
	"moonmoon_ow",
	"scarra",
	"lord_kebun",
	"fextralife",
	"alanzoka",
	"kitboga",
	"p10e",
	"aydan",
	"nl_kripp",
	"brookeab",
	"gronkh",
	"mizkif",
	"elded",
	"aspectfn",
	"thedarkness",
	"cdnthe3rd",
	"ness",
	"vinesauce",
	"riotgames",
	"dasmehdi",
	"surefour",
	"zilioner",
	"lospollostv",
	"mym_alkapone",
	"calebhart42",
	"overwatchleague",
	"corinnakopf",
	"uberhaxornova",
	"jinsooo0",
	"corinnakopf",
	"overwatchleague",
	"nairomk",
	"skipnho",
	"calebhart42",
	"masondota2",
	"robomaster",
	"aspectfn",
	"mandiocaa1",
	"rhdgurwns",
	"chocotaco",
	"illidanstr",
	"kinggeorge",
	"kingrichard",
	"problemwright",
	"hasanabi",
	"never_loses",
	"moistcr1tikal",
	"nanayango3o",
	"mrfreshasian",
	"iamcristinini",
	"goldglove",
	"rakanoolive",
	"symfuhny",
	"ratedepicz",
	"arigameplays",
	"trick2g",
	"ma_mwa",
	"goldglove",
	"arigameplays",
	"trick2g",
	"kyo1984123",
	"ma_mwa",
	"wingsofdeath",
	"cyr",
	"aurateur",
	"strippin",
	"bebelolz",
	"sardoche",
	"penta",
	"asmodaitv",
	"dellor",
	"lilypichu",
	"purgegamers",
	"jdotb",
	"bikeman",
	"amouranth",
	"vargskelethor",
	"donutoperator",
	"ashlynn",
	"trihex",
	"karasmai",
	"shotz",
	"yamatonjp",
	"calebdmtg",
	"juanjuegajuegos",
	"grimmmz",
	"tanovich",
	"juanjuegajuegos",
	"grimmmz",
	"calebdmtg",
	"gaules",
	"uzra",
	"peachsaliva",
	"tanovich",
	"emongg",
	"jennajulien",
	"gladd",
	"spaceboy",
	"bazzagazza",
	"cdewx",
	"mch_agg",
	"gabepeixe",
	"ratirl",
	"datmodz",
	"formal",
	"jltomy",
	"burkeblack",
	"chicalive",
	"nicewigg",
	"immortalhd",
	"quarterjade",
	"heelmike",
	"zrush",
	"pangaeapanga",
	"livekiss",
	"protonjon",
	"shrimp9710",
	"paymoneywubby",
	"djdduty",
	"maiyadanny",
}

func schedule(what func(), delay time.Duration) chan bool {
	stop := make(chan bool)

	go func() {
		for {
			what()
			select {
			case <-time.After(delay):
			case <-stop:
				return
			}
		}
	}()

	return stop
}

func main() {
	ctx := context.Background()
	client, err := elastic.NewClient()
	if err != nil {
		panic(err)
	}

	info, code, err := client.Ping("http://127.0.0.1:9200").Do(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Elasticsearch returned with code %d and version %s\n", code, info.Version.Number)

	// Getting the ES version number is quite common, so there's a shortcut
	esversion, err := client.ElasticsearchVersion("http://127.0.0.1:9200")
	if err != nil {
		// Handle error
		panic(err)
	}
	fmt.Printf("Elasticsearch version %s\n", esversion)

	/*/ Delete an index.
	deleteIndex, err := client.DeleteIndex("twitch").Do(ctx)
	if err != nil {
		// Handle error
		panic(err)
	}
	if !deleteIndex.Acknowledged {
		// Not acknowledged
	}*/

	// Use the IndexExists service to check if a specified index exists.
	exists, err := client.IndexExists("twitch").Do(ctx)
	if err != nil {
		// Handle error
		panic(err)
	}
	if !exists {
		// Create a new index.
		createIndex, err := client.CreateIndex("twitch").BodyString(mapping).Do(ctx)
		if err != nil {
			// Handle error
			panic(err)
		}
		if !createIndex.Acknowledged {
			// Not acknowledged
		}
	}

	con := irc.IRC("ttvlogger", "ttvlogger")
	con.Password = "oauth:hardcodedbutshouldbeenvvariable"
	err = con.Connect("irc.chat.twitch.tv:6667")

	if err != nil {
		fmt.Println("Failed connecting")
		return
	}

	con.AddCallback("001", func(e *irc.Event) {
		for _, streamName := range streamers {
			con.Join(fmt.Sprintf("#%s", streamName))
			fmt.Printf("Sent choin for stream chat for %s\n", streamName)
		}
	})

	con.AddCallback("JOIN", func(e *irc.Event) {
		//con.Privmsg(roomName, "Hello! I am a friendly IRC bot who will echo everything you say.")
		fmt.Printf("Joined stream chat for %s\n", e.Arguments[0])
	})

	con.AddCallback("PRIVMSG", func(e *irc.Event) {
		//con.Privmsg(roomName, e.Message())
		//fmt.Printf("%s %s:%s: %s\n", time.Now(), e.Arguments[0], e.User, e.Message())
		message := Message{User: e.User, Message: e.Message(), Timestamp: time.Now().UTC(), Channel: e.Arguments[0]}
		_, err := client.Index().Index("twitch").Type("message").BodyJson(message).Do(ctx)
		if err != nil {
			panic(err)
		}
		//fmt.Printf("Indexed message %s to index %s, type %s\n", put.Id, put.Index, put.Type)
	})

	flush := func() { // Periodically flush chat messages, may be some race condition with the callback?
		_, err = client.Flush().Index("twitch").Do(ctx)
		if err != nil {
			panic(err)
		}
	}

	stop := schedule(flush, 1000*time.Millisecond)
	con.Loop()
	stop <- true
}
