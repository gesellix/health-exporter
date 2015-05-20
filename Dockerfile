FROM alpine:edge
MAINTAINER Tobias Gesellchen <tobias@gesellix.de> (@gesellix)
EXPOSE 9990

ENV GOPATH /go
ENV APPPATH $GOPATH/src/github.com/gesellix/simple-exporter
COPY . $APPPATH

RUN apk add --update -t build-deps go git mercurial libc-dev gcc libgcc \
    && cd $APPPATH && go get -d && go build -o /bin/simple-exporter \
    && apk del --purge build-deps && rm -rf $GOPATH

ENTRYPOINT [ "/bin/simple-exporter", "-telemetry.address=0.0.0.0:9990" ]
CMD [ "-logtostderr" ]
