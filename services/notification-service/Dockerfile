FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/notification-service ./services/notification-service/cmd

FROM alpine:3.20
RUN adduser -D app
USER app
COPY --from=build /bin/notification-service /bin/notification-service
ENTRYPOINT ["/bin/notification-service"]
