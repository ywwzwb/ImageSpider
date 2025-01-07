
FROM golang:1.23-alpine
RUN go env -w CGO_ENABLED='1' && \
    go env -w GOPROXY=https://goproxy.cn,direct,direct && \
    sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories && \
    apk update && apk add git cmake make x265-dev jpeg-dev libpng-dev libtool libheif-dev gcc libc-dev

WORKDIR /go/src/imagespider/
COPY ./ ./
RUN go build

FROM alpine:latest
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.tuna.tsinghua.edu.cn/g' /etc/apk/repositories &&\
    apk update && apk add libheif-dev x265-dev jpeg-dev libpng-dev
COPY --from=0 /go/src/imagespider/imagespider /bin/imagespider
CMD [ "imagespider","-c","/config/config.yaml"]
