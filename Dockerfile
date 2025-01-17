FROM golang:1.23-alpine

RUN mkdir /api-server
WORKDIR /api-server


# COPY go.mod and go.sum files to the workspace
COPY go.mod .
COPY go.sum .

# Get dependancies - will also be cached if we won't change mod/sum
#RUN go mod download

COPY pkg/ pkg/
COPY docs/ docs/
COPY main.go .

RUN if [ ! -d "/api-server/vendor" ]; then  go mod vendor; fi

RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor -a -installsuffix cgo -o bin/app .



FROM alpine:3.21
RUN apk add --no-cache tzdata ca-certificates && \
    update-ca-certificates
ENV TZ=Asia/Taipei
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime
RUN mkdir -p /etc/api-server
COPY ./zoneinfo.zip /usr/local/go/lib/time/

COPY --from=0 /api-server/bin/app .
COPY conf/api-config-template.json /etc/api-server/api-config.json

CMD ["/app", "--logtostderr=true"]
