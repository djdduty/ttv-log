package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/djdduty/ttv-log/config"
	"github.com/julienschmidt/httprouter"
	"github.com/olivere/elastic/v7"
	"github.com/unrolled/render"
)

const (
	// StreamPath ...
	StreamPath = "/api/streams"
	// UserPath ...
	UserPath = "/api/users"
	// MessagePath ...
	MessagePath = "/api/messages"
)

// Handler handles the http requests to api endpoints
type Handler struct {
	R *render.Render
	E *config.ElasticConnector
}

// NewHandler instantiates a handler.
func NewHandler(r *render.Render, e *config.ElasticConnector) *Handler {
	return &Handler{
		R: r,
		E: e,
	}
}

// SetRoutes registers this handler's routes.
func (h *Handler) SetRoutes(r *httprouter.Router) {
	r.GET(StreamPath, h.ListStreams)
	r.GET(UserPath, h.ListUsers)
	r.GET(MessagePath, h.ListMessages)
}

type healthStatus struct {
	Status string `json:"status"`
}

// ListStreams ...
func (h *Handler) ListStreams(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	queryValues := r.URL.Query()
	limit, err := url.QueryUnescape(queryValues.Get("limit"))
	if err != nil {
		h.R.Text(rw, http.StatusBadRequest, err.Error())
	}

	client := h.E.GetClient()
	ctx, cancel := context.WithTimeout(h.E.GetContext(), 5*time.Second)
	defer cancel()

	search := client.Search().Index("twitch")
	search = search.Query(elastic.NewBoolQuery()).Size(0)
	agg := elastic.NewTermsAggregation().Field("Channel.keyword").NumPartitions(1)
	if limit != "" {
		limit, err := strconv.Atoi(limit)
		if err != nil {
			h.R.Text(rw, http.StatusBadRequest, err.Error())
		}
		if limit > 1000 {
			h.R.Text(rw, http.StatusBadRequest, "limit cannot exceed 1000")
		}
		agg = agg.Size(limit)
	}
	search = search.Aggregation("channels", agg)

	sr, err := search.Do(ctx)
	if err != nil {
		panic(err)
	}

	var channels []*Channel
	// Deserialize aggregations
	if agg, found := sr.Aggregations.Terms("channels"); found {
		for _, bucket := range agg.Buckets {
			channel := &Channel{
				Name:        strings.TrimPrefix(bucket.Key.(string), "#"),
				NumMessages: bucket.DocCount,
			}
			channels = append(channels, channel)
		}
	}

	h.R.JSON(rw, http.StatusOK, &channels)
}

// ListUsers ...
func (h *Handler) ListUsers(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	queryValues := r.URL.Query()
	limit, err := url.QueryUnescape(queryValues.Get("limit"))
	if err != nil {
		h.R.Text(rw, http.StatusBadRequest, err.Error())
	}

	client := h.E.GetClient()
	ctx, cancel := context.WithTimeout(h.E.GetContext(), 5*time.Second)
	defer cancel()

	search := client.Search().Index("twitch")
	search = search.Query(elastic.NewBoolQuery()).Size(0)
	agg := elastic.NewTermsAggregation().Field("User.keyword")
	if limit != "" {
		limit, err := strconv.Atoi(limit)
		if err != nil {
			h.R.Text(rw, http.StatusBadRequest, err.Error())
		}
		if limit > 1000 {
			h.R.Text(rw, http.StatusBadRequest, "limit cannot exceed 1000")
		}
		agg = agg.Size(limit)
	}
	search = search.Aggregation("users", agg)

	sr, err := search.Do(ctx)
	if err != nil {
		panic(err)
	}

	var users []*Channel
	// Deserialize aggregations
	if agg, found := sr.Aggregations.Terms("users"); found {
		for _, bucket := range agg.Buckets {
			user := &Channel{
				Name:        bucket.Key.(string),
				NumMessages: bucket.DocCount,
			}
			users = append(users, user)
		}
	}

	h.R.JSON(rw, http.StatusOK, &users)
}

// ListMessages ...
func (h *Handler) ListMessages(rw http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	queryValues := r.URL.Query()
	limit, err := url.QueryUnescape(queryValues.Get("limit"))
	if err != nil {
		h.R.Text(rw, http.StatusBadRequest, err.Error())
		return
	}

	afterTime, err := url.QueryUnescape(queryValues.Get("after_timestamp"))
	if err != nil {
		h.R.Text(rw, http.StatusBadRequest, err.Error())
		return
	}

	afterID, err := url.QueryUnescape(queryValues.Get("after_id"))
	if err != nil {
		h.R.Text(rw, http.StatusBadRequest, err.Error())
		return
	}

	channelName, err := url.QueryUnescape(queryValues.Get("stream"))
	if err != nil {
		h.R.Text(rw, http.StatusBadRequest, err.Error())
		return
	}

	client := h.E.GetClient()
	ctx, cancel := context.WithTimeout(h.E.GetContext(), 5*time.Second)
	defer cancel()

	search := client.Search().Index("twitch")
	q := elastic.NewBoolQuery()
	if channelName != "" {
		q = q.Must(elastic.NewTermQuery("Channel.keyword", fmt.Sprintf("#%s", channelName)))
	}

	search = search.Query(q)
	search = search.Sort("Timestamp", false)
	search = search.Sort("_id", false)

	if limit != "" {
		limit, err := strconv.ParseInt(limit, 10, 32)
		if err != nil {
			h.R.Text(rw, http.StatusBadRequest, err.Error())
			return
		}
		if limit > 1000 {
			h.R.Text(rw, http.StatusBadRequest, "limit cannot exceed 1000")
			return
		}
		search = search.Size(int(limit))
	}

	if afterTime != "" && afterID != "" {
		_, err := strconv.ParseInt(afterTime, 10, 64)
		if err != nil {
			h.R.Text(rw, http.StatusBadRequest, err.Error())
			return
		}
		search = search.SearchAfter(afterTime, afterID)
	}

	sr, err := search.Do(ctx)
	if err != nil {
		h.R.Text(rw, http.StatusInternalServerError, err.Error())
		panic(err)
	}

	var resp StreamMessagesResponse
	resp.ChannelName = channelName
	var lastMessage *Message
	if sr.TotalHits() > 0 {
		for _, hit := range sr.Hits.Hits {
			message := new(Message)
			if err := json.Unmarshal(hit.Source, message); err != nil {
				h.R.Text(rw, http.StatusInternalServerError, err.Error())
				panic(err)
			}
			message.ID = hit.Id
			lastMessage = message
			resp.Messages = append(resp.Messages, message)
		}

		if lastMessage != nil {
			queryValues.Set("after_timestamp", strconv.FormatInt(lastMessage.Timestamp.UnixNano()/int64(time.Millisecond), 10))
			queryValues.Set("after_id", lastMessage.ID)
			resp.Next = fmt.Sprintf(
				"%s?%s",
				r.URL.Path,
				queryValues.Encode(),
			)
		}
	}

	h.R.JSON(rw, http.StatusOK, &resp)
}
