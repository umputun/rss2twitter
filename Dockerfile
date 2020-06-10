FROM umputun/baseimage:buildgo-latest as build

WORKDIR /build/rss2twitter
ADD . /build/rss2twitter

# run tests
RUN cd app && go test -mod=vendor ./...

RUN \
    version=$(/script/git-rev.sh) && \
    echo "version=$version" && \
    go build -mod=vendor -o rss2twitter -ldflags "-X main.revision=${version} -s -w" ./app


FROM umputun/baseimage:app-latest

COPY ./exclusion-patterns.txt /srv/exclusion-patterns.txt

COPY --from=build /build/rss2twitter/rss2twitter /srv/rss2twitter
RUN \
    chown -R app:app /srv && \
    chmod +x /srv/rss2twitter

WORKDIR /srv

CMD ["/srv/rss2twitter"]
ENTRYPOINT ["/init.sh"]
