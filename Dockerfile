FROM alpine:3.9
LABEL MAINTAINER "6congyao@gmail.com"

RUN apk --no-cache add ca-certificates wget
RUN wget -q -O /etc/apk/keys/sgerrand.rsa.pub https://alpine-pkgs.sgerrand.com/sgerrand.rsa.pub
RUN wget https://github.com/sgerrand/alpine-pkg-glibc/releases/download/2.29-r0/glibc-2.29-r0.apk
RUN apk add glibc-2.29-r0.apk

ADD bin/alpine/strike /bin/
ADD bin/alpine/mqgateway.json /bin/

EXPOSE 8055 8056

HEALTHCHECK CMD ["/bin/strike", "ping"]

ENTRYPOINT ["/bin/strike", "-c", "mqgateway.json"]