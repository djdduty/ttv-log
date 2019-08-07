package config

import (
	"context"

	"github.com/olivere/elastic/v7"
	"github.com/sirupsen/logrus"
)

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

// ElasticConnector ...
type ElasticConnector struct {
	ctx    context.Context
	url    string
	client *elastic.Client
	l      logrus.FieldLogger
}

// Init initiates the elasticsearch connection
func (e *ElasticConnector) Init(url, username, password string, l logrus.FieldLogger) error {
	e.url = url
	e.ctx = context.Background()

	client, err := elastic.NewSimpleClient(
		elastic.SetURL(url),
		elastic.SetBasicAuth(username, password),
	)
	if err != nil {
		l.Errorf("Could not initiate elasticsearch client for %s\n", url)
		return err
	}

	info, code, err := client.Ping(url).Do(e.ctx)
	if err != nil {
		l.Errorf("Could not ping elasticsearch at %s\n", url)
		return err
	}

	l.Infof("Elasticsearch returned with code %d and version %s\n", code, info.Version.Number)

	// Getting the ES version number is quite common, so there's a shortcut
	esversion, err := client.ElasticsearchVersion(url)
	if err != nil {
		// Handle error
		l.Errorf("Could not initiate elasticsearch client for %s\n", url)
		return err
	}
	l.Infof("Elasticsearch version %s\n", esversion)

	// Use the IndexExists service to check if a specified index exists.
	exists, err := client.IndexExists("twitch").Do(e.ctx)
	if err != nil {
		// Handle error
		return err
	}
	if !exists {
		// Create a new index.
		createIndex, err := client.CreateIndex("twitch").Body(mapping).Do(e.ctx)
		if err != nil {
			// Handle error
			panic(err)
		}
		if !createIndex.Acknowledged {
			// Not acknowledged
		}
	} else {
		/*/Delete an index.
		deleteIndex, err := client.DeleteIndex("twitch").Do(e.ctx)
		if err != nil {
			// Handle error
			panic(err)
		}
		if !deleteIndex.Acknowledged {
			// Not acknowledged
		}*/
	}

	e.l = l
	e.client = client

	return nil
}

// Ping pings the elasticsearch server
func (e *ElasticConnector) Ping() error {
	info, code, err := e.client.Ping(e.url).Do(e.ctx)
	if err != nil {
		e.l.Errorf("Could not ping elasticsearch at %s\n", e.url)
		return err
	}

	e.l.Debugf("Elasticsearch returned with code %d and version %s\n", code, info.Version.Number)
	return nil
}

// GetClient returns the elastic client
func (e *ElasticConnector) GetClient() *elastic.Client {
	return e.client
}

// GetContext returns the connector's context
func (e *ElasticConnector) GetContext() context.Context {
	return e.ctx
}