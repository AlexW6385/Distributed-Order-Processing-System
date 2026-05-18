FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/payment-service ./services/payment-service/cmd

FROM alpine:3.20
RUN adduser -D app
USER app
COPY --from=build /bin/payment-service /bin/payment-service
ENTRYPOINT ["/bin/payment-service"]
