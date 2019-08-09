package api

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/olivere/elastic/v7"
)

// Channel represents a twitch stream
type Channel struct {
	Name        string `json:"name"`
	NumMessages int64  `json:"num_messages_logged"`
}

// Message represents a twitch chat message.
type Message struct {
	ID        string    `json:"Id"`
	Timestamp time.Time `json:"Timestamp"`
	Message   string    `json:"Message"`
	Channel   string    `json:"Channel"`
	User      string    `json:"User"`
}

// StreamMessagesResponse represents a stream with all it's messages
type StreamMessagesResponse struct {
	ChannelName string `json:"channel_name"`
	Next        string `json:"next_page"`
	Messages    []*Message
}

// Finder specifies a finder for messages.
type Finder struct {
	name       string
	from, size int
	sort       []string
	pretty     bool
}

// FinderResponse is the outcome of calling StreamFinder.Find.
type FinderResponse struct {
	Total    int64
	Messages []*Message
	Channels []*Channel
}

// NewFinder creates a new finder for messages.
// Use the funcs to set up filters and search properties,
// then call Find to execute.
func NewFinder() *Finder {
	return &Finder{}
}

// From specifies the start index for pagination.
func (f *Finder) From(from int) *Finder {
	f.from = from
	return f
}

// Size specifies the number of items to return in pagination.
func (f *Finder) Size(size int) *Finder {
	f.size = size
	return f
}

// Sort specifies one or more sort orders.
// Use a dash (-) to make the sort order descending.
// Example: "name" or "-year".
func (f *Finder) Sort(sort ...string) *Finder {
	if f.sort == nil {
		f.sort = make([]string, 0)
	}
	f.sort = append(f.sort, sort...)
	return f
}

// Pretty when enabled, asks the server to return the
// response formatted and indented.
func (f *Finder) Pretty(pretty bool) *Finder {
	f.pretty = pretty
	return f
}

// Find executes the search and returns a response.
func (f *Finder) Find(ctx context.Context, client *elastic.Client) (FinderResponse, error) {
	var resp FinderResponse

	// Create service and use query, aggregations, sort, filter, pagination funcs
	search := client.Search().Index("twitch").Pretty(f.pretty)
	search = f.query(search)
	search = f.aggs(search)
	search = f.sorting(search)
	search = f.paginate(search)

	// TODO Add other properties here, e.g. timeouts, explain or pretty printing

	// Execute query
	sr, err := search.Do(ctx)
	if err != nil {
		return resp, err
	}

	// Decode response
	messages, err := f.decodeMessages(sr)
	if err != nil {
		return resp, err
	}
	resp.Messages = messages
	resp.Total = sr.Hits.TotalHits.Value

	// Deserialize aggregations
	if agg, found := sr.Aggregations.Terms("channels"); found {
		for _, bucket := range agg.Buckets {
			channel := &Channel{
				Name:        bucket.Key.(string),
				NumMessages: bucket.DocCount,
			}
			resp.Channels = append(resp.Channels, channel)
		}
	}

	/*/ Use the correct function on sr.Aggregations.XXX. It must match the
	// aggregation type specified at query time.
	// See https://github.com/olivere/elastic/blob/release-branch.v6/search_aggs.go
	// for all kinds of aggregation types.
	if agg, found := sr.Aggregations.Terms("years_and_genres"); found {
		resp.YearsAndGenres = make(map[int][]NameCount)
		for _, bucket := range agg.Buckets {
			// JSON doesn't have integer types: All numeric values are float64
			floatValue, ok := bucket.Key.(float64)
			if !ok {
				panic("expected a float64")
			}
			var (
				year          = int(floatValue)
				genresForYear []NameCount
			)
			// Iterate over the sub-aggregation
			if subAgg, found := bucket.Terms("genres_by_year"); found {
				for _, subBucket := range subAgg.Buckets {
					genresForYear = append(genresForYear, NameCount{
						Name:  subBucket.Key.(string),
						Count: subBucket.DocCount,
					})
				}
			}
			resp.YearsAndGenres[year] = genresForYear
		}
	}*/

	return resp, nil
}

// query sets up the query in the search service.
func (f *Finder) query(service *elastic.SearchService) *elastic.SearchService {
	/*if f.genre == "" && f.year == 0 {
		service = service.Query(elastic.NewMatchAllQuery())
		return service
	}*/

	q := elastic.NewBoolQuery()
	/*if f.genre != "" {
		q = q.Must(elastic.NewTermQuery("genre", f.genre))
	}
	if f.year > 0 {
		q = q.Must(elastic.NewTermQuery("year", f.year))
	}*/

	// TODO Add other queries and filters here, maybe differentiating between AND/OR etc.

	service = service.Query(q)
	return service
}

// aggs sets up the aggregations in the service.
func (f *Finder) aggs(service *elastic.SearchService) *elastic.SearchService {
	// Terms aggregation by channel
	agg := elastic.NewTermsAggregation().Field("Channel.keyword")
	/*if f.from > 0 {
		agg = agg.From(f.from)
	}
	if f.size > 0 {
		agg = agg.Size(f.size)
	}*/
	service = service.Aggregation("channels", agg)

	/*/ Add a terms aggregation of Year, and add a sub-aggregation for Genre
	subAgg := elastic.NewTermsAggregation().Field("genre")
	agg = elastic.NewTermsAggregation().Field("year").
		SubAggregation("genres_by_year", subAgg)
	service = service.Aggregation("years_and_genres", agg)*/

	return service
}

// paginate sets up pagination in the service.
func (f *Finder) paginate(service *elastic.SearchService) *elastic.SearchService {
	if f.from > 0 {
		service = service.From(f.from)
	}
	if f.size > 0 {
		service = service.Size(f.size)
	}
	return service
}

// sorting applies sorting to the service.
func (f *Finder) sorting(service *elastic.SearchService) *elastic.SearchService {
	if len(f.sort) == 0 {
		// Sort by score by default
		service = service.Sort("_score", false)
		return service
	}

	// Sort by fields; prefix of "-" means: descending sort order.
	for _, s := range f.sort {
		s = strings.TrimSpace(s)

		var field string
		var asc bool

		if strings.HasPrefix(s, "-") {
			field = s[1:]
			asc = false
		} else {
			field = s
			asc = true
		}

		// Maybe check for permitted fields to sort

		service = service.Sort(field, asc)
	}
	return service
}

// decodeMessages takes a search result and deserializes the films.
func (f *Finder) decodeMessages(res *elastic.SearchResult) ([]*Message, error) {
	if res == nil || res.TotalHits() == 0 {
		return nil, nil
	}

	var messages []*Message
	for _, hit := range res.Hits.Hits {
		message := new(Message)
		if err := json.Unmarshal(*&hit.Source, message); err != nil {
			return nil, err
		}
		// TODO Add Score here, e.g.:
		// film.Score = *hit.Score
		messages = append(messages, message)
	}
	return messages, nil
}
