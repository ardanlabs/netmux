FROM alpine:latest

RUN apk add nmap tcpdump libpcap-dev
COPY ./bin/server /bin/server

CMD "/bin/server"