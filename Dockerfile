FROM umputun/baseimage:buildgo-latest as build

ARG COVERALLS_TOKEN
ARG CI
ARG TRAVIS
ARG TRAVIS_BRANCH
ARG TRAVIS_COMMIT
ARG TRAVIS_JOB_ID
ARG TRAVIS_JOB_NUMBER
ARG TRAVIS_OS_NAME
ARG TRAVIS_PULL_REQUEST
ARG TRAVIS_PULL_REQUEST_SHA
ARG TRAVIS_REPO_SLUG
ARG TRAVIS_TAG
ARG DRONE
ARG DRONE_TAG
ARG DRONE_COMMIT
ARG DRONE_BRANCH
ARG DRONE_PULL_REQUEST

WORKDIR /go/src/github.com/umputun/rss2twitter
ADD . /go/src/github.com/umputun/rss2twitter

# run tests
RUN cd app && go test ./...

# linters
RUN golangci-lint run --out-format=tab --disable-all --tests=false --enable=interfacer --enable=unconvert --enable=megacheck \
    --enable=structcheck --enable=gas --enable=gocyclo --enable=dupl --enable=misspell --enable=maligned --enable=unparam \
    --enable=varcheck --enable=deadcode --enable=typecheck --enable=errcheck ./...

# coverage report
RUN mkdir -p target && /script/coverage.sh

# submit coverage to coverals if COVERALLS_TOKEN in env
RUN if [ -z "$COVERALLS_TOKEN" ] ; then \
    echo "coverall not enabled" ; \
    else goveralls -coverprofile=.cover/cover.out -service=travis-ci -repotoken $COVERALLS_TOKEN || echo "coverall failed!"; fi

RUN \
    version=$(/script/git-rev.sh) && \
    echo "version=$version" && \
    go build -o rss2twitter -ldflags "-X main.revision=${version} -s -w" ./app


FROM umputun/baseimage:app-latest

COPY --from=build /go/src/github.com/umputun/rss2twitter/rss2twitter /srv/rss2twitter
RUN \
    chown -R app:app /srv && \
    chmod +x /srv/rss2twitter

WORKDIR /srv

CMD ["/srv/rss2twitter"]
ENTRYPOINT ["/init.sh"]
