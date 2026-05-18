FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/order-service ./services/order-service/cmd

FROM alpine:3.20
RUN adduser -D app
USER app
COPY --from=build /bin/order-service /bin/order-service
ENTRYPOINT ["/bin/order-service"]
