# TTV Logbot

TTV Logbot is meant to be a scalable bot cluster for monitoring and storing the chat of many (>1000) twitch streams. It consists of three parts: conductor, worker, and dispatcher. Log data will be stored in elasticsearch for high performance searching and querying.

## Dispatcher
The dispatcher is responsible for maintaining the "mega" list of streams and distributing channel lists to conductors. It will check with the twitch api to get the top streamers which are currently live as well as a supplementary list of streams which should always been monitored. The dispatcher will use the streamer's average viewership, live message rate, and real time metrics from conductors to determine "capacity" requirements for job distribution. 

## Conductor
The conductor acts as a watchdog for worker processes for filling out the capacity of the host machine. Once it receives jobs (stream names) from the dispatcher it will spawn child processes to act as workers and internally dispatch jobs to them. The conductor will frequently report local message throughput to the dispatcher, as well as infrequently checking and reporting the host resource utilization.

## Worker
The worker is a simple IRC bot, it will connect to the configured network and join all the channels the conductor tells it to. Using an event driven IRC client framework it will queue received messages periodically flush them to an elasticsearch index.