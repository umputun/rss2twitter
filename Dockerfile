FROM umputun/baseimage:buildgo-latest as build

ENV CGO_ENABLED=0

WORKDIR /build/rss2twitter
ADD . /build/rss2twitter

# run tests
RUN cd app && go test -mod=vendor ./...

RUN \
    version=$(/script/git-rev.sh) && \
    echo "version=$version" && \
    go build -mod=vendor -o rss2twitter -ldflags "-X main.revision=${version} -s -w" ./app


FROM umputun/baseimage:app-latest

COPY --from=build /build/rss2twitter/rss2twitter /srv/rss2twitter
RUN \
    chown -R app:app /srv && \
    chmod +x /srv/rss2twitter

WORKDIR /srv

ENTRYPOINT ["/srv/rss2twitter"]
