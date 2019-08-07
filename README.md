# TTV Logbot

TTV Logbot is meant to be a scalable bot cluster for monitoring and storing the chat of many (>1000) twitch streams. It consists of three parts: conductor, worker, and dispatcher. Log data will be stored in Elasticsearch for high-performance searching and querying.

## Dispatcher
The dispatcher is responsible for maintaining the "mega" list of streams and distributing channel lists to conductors. It will check with the twitch API for the top streamers that are currently live, as a supplementary list of streams that should always be monitored. The dispatcher will use the streamer's average viewership, live message rate, and real-time metrics from conductors to determine "capacity" requirements for job distribution. 

## Conductor
The conductor acts as a watchdog for worker processes for filling out the capacity of the host machine. Once it receives jobs (stream names) from the dispatcher, it will spawn child processes to act as workers and internally dispatch jobs to them. The conductor will frequently report local message throughput to the dispatcher, as well as infrequently checking and reporting the host resource utilization.

## Worker
The worker is a simple IRC bot. It will connect to the configured network and join all the channels the conductor tells it to. Using an event-driven IRC client framework it will queue received messages and periodically flush them to an Elasticsearch index.

### Commands
First, copy `docs/config.yaml` to `./config.yaml` in the working directory of the executable

#### Run the web server, allowing non https access

    ./ttv-log serve --dangerous-force-http

#### Run the irc bot

    ./ttv-log bot