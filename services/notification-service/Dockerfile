FROM golang:1.25-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/notification-service ./services/notification-service

FROM alpine:3.20

WORKDIR /app
COPY --from=build /bin/notification-service /bin/notification-service

EXPOSE 8082
CMD ["/bin/notification-service"]
