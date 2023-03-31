FROM golang:alpine3.14 as compiler

RUN apk add git

WORKDIR /app

COPY . .

RUN go build

FROM alpine:3.14

WORKDIR /app

COPY --from=compiler /app/birthday-reminder .

CMD ["./birthday-reminder"]