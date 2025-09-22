# Go Monitor

## Known issues and other comments

Sometimes, there is a goroutine hanging around when you terminate the program. I'm not sure if I'm impatient or it genuinely hangs around forever. I haven't had time to profile it in depth.

I didn't add any metrics. I swear I know how to do Prometheus metrics at least, I maintained the metrics library at Ecosia.

The dockerfile is a bit clunky but it should do the job.

I have had a lot less time than I expected unfortunately due to sickness and other matters, but I feel like I've spent enough time on this task.

You can potentially change the log output to use json, to make easier to parse.

## Database Schema

| logs               |                        |
|--------------------|-----------------------:|
| **timestamp**      | timestamp [primary key]|
| **url**            |  varchar [primary key] |
| duration           |                 bigint |
| status_code        |                integer |
| regexp_matches     |                boolean |
| error              |                varchar |

There is probably a better way to do this, but to be honest I haven't done anything with anything more complicated than a key value store in about three years, so I've had to have a big refresher already.

## Prerequisites

### Go

This project was built with Go 1.25.1.

### Goose

You can run `Goose` to create the tables for you. Follow the [Goose](https://github.com/pressly/goose) installation instructions. Then export to your environment:

```bash
GOOSE_DRIVER=postgres
GOOSE_DBSTRING=postgres://[username]:[password]@[hostname]:[port]/[dbname]?sslmode=require
GOOSE_MIGRATION_DIR=./migrations
GOOSE_TABLE=goose_migrations
```

From the root folder, run `goose up`.

## Usage

Configuration is done through environment variables.

```bash
DATABASE_URL=postgres://[username]:[password]@[hostname]:[port]/[dbname]?sslmode=require
BATCH_SIZE=100 # Choose a sensible variable, this is how many inserts will be batched for the database.
LOG_LEVEL=Error # Use slog-compatible variables
FILE_URL=sample-big.json # URL of the file to use as configuration.
```

## Sample Files

### sample-url-list.json

Simple sample list to enable you to test arbitrary urls. It's small, so you can easily change things around.

### sample-big.json

Partial dataset from [Kaggle](https://www.kaggle.com/datasets/bpali26/popular-websites-across-the-globe). Contains over 9000 rows: some repeated, some defunct. The monitor does not deduplicate or remove any data from configuration.
