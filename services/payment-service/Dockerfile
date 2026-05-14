FROM golang:1.25-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /bin/payment-service ./services/payment-service

FROM alpine:3.20

WORKDIR /app
COPY --from=build /bin/payment-service /bin/payment-service

EXPOSE 50052

CMD ["/bin/payment-service"]
