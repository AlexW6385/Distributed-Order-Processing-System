FROM golang:1.25-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /bin/product-service ./services/product-service

FROM alpine:3.20

WORKDIR /app
COPY --from=build /bin/product-service /bin/product-service

EXPOSE 50051

CMD ["/bin/product-service"]
