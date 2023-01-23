FROM golang:1.19-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download && go mod verify

COPY . .

RUN go build -v -o /neveralone ./cmd/neveralone/.

FROM alpine:latest

COPY --from=build /neveralone .

EXPOSE 8080

ENTRYPOINT ["./neveralone"]