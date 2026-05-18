FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/product-service ./services/product-service/cmd

FROM alpine:3.20
RUN adduser -D app
USER app
COPY --from=build /bin/product-service /bin/product-service
ENTRYPOINT ["/bin/product-service"]
