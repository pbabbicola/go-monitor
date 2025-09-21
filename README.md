# Go Monitor

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

## Sample Files

### sample-url-list.json

Simple sample list to enable you to test arbitrary urls. It's small, so you can easily change things around.

### sample-big.json

Partial dataset from [Kaggle](https://www.kaggle.com/datasets/bpali26/popular-websites-across-the-globe). Contains over 9000 rows: some repeated, some defunct. The monitor does not deduplicate or remove any data from configuration.
