FROM golang:alpine

ADD . $GOAPTH/src/github.com/jtaylorcpp/gammacorp-defaultbackend
WORKDIR $GOAPTH/src/github.com/jtaylorcpp/gammacorp-defaultbackend

RUN go build -o /usr/bin/default

CMD /usr/bin/default
