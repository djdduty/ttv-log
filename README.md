# TTV Logbot

TTV Logbot is meant to be a high-throughput scalable bot for monitoring the chat of many (>1000) twitch streams and log the messages to an elasticsearch cluster. It consists of three parts: conductor, worker, and dispatcher.

## Dispatcher
The dispatcher is responsible for maintaining the "mega" list of streamers. It will check with the twitch api to get the top streamers which are currently live as well as a pre-defined (but dynamic) list of streams which should always been monitored. The dispatcher will use the streamer's average viewership (and then live message rate for the stream) to determine "capacity" requirements; creating a collection of smaller channel lists for each conductor process. The dispatcher process will listen on TCP which conductor clients will connect to.

## Conductor
The conductor acts as a watchdog for spawning local workers and managing total process capacity. It will connect to the Dispatcher via a simple TCP protocol for liveness checks and dynamic job assignment. Once it receives jobs from the dispatcher it will spawn child processes to act as workers and internally dispatch it's sub-set of jobs to them. The conductor will monitor assigned message throughput frequent and report to the dispatcher, as well as infrequently checking and reporting the host resource utilization.

## Worker
The worker is a simple IRC bot, it will connect to the configured network and join all the channels the conductor tells it to. Using an event driven IRC framework it will queue messages to be added to an elasticsearch index and periodically flush the elasticsearch queue to the cluster.