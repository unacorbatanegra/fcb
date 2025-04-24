FROM golang:1.24-alpine

WORKDIR /app

COPY . .

RUN go build -o order-generator .

CMD ["./order-generator"]