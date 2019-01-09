FROM umputun/baseimage:buildgo-latest as build

WORKDIR /go/src/github.com/umputun/rss2twitter
ADD . /go/src/github.com/umputun/rss2twitter

# run tests
RUN cd app && go test ./...

# linters
RUN golangci-lint run --deadline=300s --out-format=tab --disable-all --tests=false --enable=interfacer --enable=unconvert \
    --enable=megacheck --enable=structcheck --enable=gas --enable=gocyclo --enable=dupl --enable=misspell \
    --enable=maligned --enable=unparam --enable=varcheck --enable=deadcode --enable=typecheck --enable=errcheck ./...

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
