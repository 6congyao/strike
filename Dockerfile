FROM alpine:3.9
LABEL MAINTAINER "6congyao@gmail.com"

RUN apk add --no-cache bash ca-certificates wget
RUN wget -q -O /etc/apk/keys/sgerrand.rsa.pub https://raw.githubusercontent.com/sgerrand/alpine-pkg-glibc/master/sgerrand.rsa.pub
RUN wget https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.27-r0/glibc-2.27-r0.apk
RUN apk add glibc-2.27-r0.apk

ADD bin/alpine/strike /bin/

EXPOSE 8055 8056

HEALTHCHECK CMD ["/bin/strike", "ping"]

ENTRYPOINT ["/bin/strike"]