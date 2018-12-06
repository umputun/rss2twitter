# rss2twitter [![Build Status](https://travis-ci.org/umputun/rss2twitter.svg?branch=master)](https://travis-ci.org/umputun/rss2twitter) [![Docker Automated build](https://img.shields.io/docker/automated/jrottenberg/ffmpeg.svg)](https://hub.docker.com/r/umputun/rss2twitter/)

The service publishes RSS updates to twitter. 

## Install

Use provided `docker-compose.yml` and change `FEED` value. All twiter-api credentials can be retrieved from https://developer.twitter.com/en/apps and should be set in environment or directly in the compose file.

## Parameters

```
Application Options:
  -r, --refresh=         refresh interval (default: 30s) [$REFRESH]
  -t, --timeout=         twitter timeout (default: 5s) [$TIMEOUT]
  -f, --feed=            rss feed url [$FEED]
      --consumer-key=    twitter consumer key [$TWI_CONSUMER_KEY]
      --consumer-secret= twitter consumer secret [$TWI_CONSUMER_SECRET]
      --access-token=    twitter access token [$TWI_ACCESS_TOKEN]
      --access-secret=   twitter access secret [$TWI_ACCESS_SECRET]
      --template=        twitter message template (default: {{.Title}} - {{.Link}}) [$TEMPLATE]
      --dry              dry mode [$DRY]
      --dbg              debug mode [$DEBUG]
```