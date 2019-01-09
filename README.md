# rss2twitter [![Build Status](https://travis-ci.org/umputun/rss2twitter.svg?branch=master)](https://travis-ci.org/umputun/rss2twitter) [![Coverage Status](https://coveralls.io/repos/github/umputun/rss2twitter/badge.svg?branch=master)](https://coveralls.io/github/umputun/rss2twitter?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/umputun/rss2twitter)](https://goreportcard.com/report/github.com/umputun/rss2twitter) [![Docker Automated build](https://img.shields.io/docker/automated/jrottenberg/ffmpeg.svg)](https://hub.docker.com/r/umputun/rss2twitter/)

The service publishes RSS updates to twitter. The reason is simple - I needed self-hosted thingy to post tweets on a feed change for my sites (podcasts and blogs). Tried several "cloud services" for this and lately switched to IFTTT. It worked, but slow and unreliable. Sometimes it took hours to get twit posted, sometimes I had to trigger it manually. In addition IFTTT can't have multiple twitter accounts defined for the same IFTTT account and I had to deal with multiple IFTTT accounts just to post to different twitter's timelines.

## Install

Use provided `docker-compose.yml` and change `FEED` value. All twitter-api credentials can be retrieved from https://developer.twitter.com/en/apps and should be set in environment or directly in the compose file.

## Templates

`--template` parameter (env `$TEMPLATE`) defines output tweet's format with:

- `{{.Title}}` - title fo rss item (entry) 
- `{{.Link}}` - rss link
- `{{.Text}}` - item description

_default is `{{.Title}} - {{.Link}}`_
  
## Parameters

```
Application Options:
  -r, --refresh=         refresh interval (default: 30s) [$REFRESH]
  -t, --timeout=         rss feed timeout (default: 5s) [$TIMEOUT]
  -f, --feed=            rss feed url [$FEED]
      --consumer-key=    twitter consumer key [$TWI_CONSUMER_KEY]
      --consumer-secret= twitter consumer secret [$TWI_CONSUMER_SECRET]
      --access-token=    twitter access token [$TWI_ACCESS_TOKEN]
      --access-secret=   twitter access secret [$TWI_ACCESS_SECRET]
      --template=        twitter message template (default: {{.Title}} - {{.Link}}) [$TEMPLATE]
      --dry              dry mode [$DRY]
      --dbg              debug mode [$DEBUG]
```
