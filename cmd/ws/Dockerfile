FROM ubuntu 

RUN apt-get update

RUN apt-get install -y golang
RUN apt-get install -y ca-certificates

RUN go install github.com/PlakarKorp/plakar/cmd/plakar@latest

ENV PATH=$PATH:"/root/go/bin"

RUN apt-get install -y fuse


COPY ./entrypoint.sh /entrypoint.sh
ENTRYPOINT ["/entrypoint.sh"]