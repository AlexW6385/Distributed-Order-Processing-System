FROM golang:1.25-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /bin/order-service ./services/order-service

FROM alpine:3.20

WORKDIR /app
COPY --from=build /bin/order-service /bin/order-service

EXPOSE 8080

CMD ["/bin/order-service"]
