# rss2twitter - [![Build Status](https://travis-ci.org/umputun/rss2twitter.svg?branch=master)](https://travis-ci.org/umputun/rss2twitter) [![Docker Automated build](https://img.shields.io/docker/automated/jrottenberg/ffmpeg.svg)](https://hub.docker.com/r/umputun/rss2twitter/)

The service publishes RSS updates to twitter. 

## install

Use provided `docker-compose.yml` and change `FEED` value. All twiter-api credentials can be retrieved from https://developer.twitter.com/en/apps and should be set in environment or directly in the compose file.

## parameters

```
Application Options:
  -r, --refresh=         refresh interval (default: 30s) [$REFRESH]
  -f, --feed=            rss feed url [$FEED]
      --consumer-key=    twitter consumer key [$CONSUMER_KEY]
      --consumer-secret= twitter consumer secret [$CONSUMER_SECRET]
      --access-token=    twitter access token [$ACCESS_TOKEN]
      --access-secret=   twitter access secret [$ACCESS_SECRET]
      --template=        twitter message template (default: {{.Title}} - {{.Link}}) [$TEMPLATE]
      --dbg              debug mode [$DEBUG]
```
