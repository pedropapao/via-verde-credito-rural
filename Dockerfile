FROM golang:1.23-alpine AS build
WORKDIR /app
COPY . .
RUN go test ./... && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /via-verde .

FROM alpine:3.20
RUN adduser -D -H -u 10001 app
USER app
COPY --from=build /via-verde /via-verde
EXPOSE 10000
CMD ["/via-verde"]
