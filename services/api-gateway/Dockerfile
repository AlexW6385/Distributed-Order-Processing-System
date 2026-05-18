FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/api-gateway ./services/api-gateway/cmd

FROM alpine:3.20
RUN adduser -D app
USER app
COPY --from=build /bin/api-gateway /bin/api-gateway
ENTRYPOINT ["/bin/api-gateway"]
