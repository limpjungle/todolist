FROM golang:1.27.1-alpine3.24 AS build
WORKDIR /src/
COPY go.mod go.sum ./ 
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-w -s" -o main ./cmd/api

FROM alpine:3.24
RUN apk --no-cache add ca-certificates && addgroup -g 1000 appgroup && adduser -D -u 1000 -G appgroup appuser
WORKDIR /app/
COPY --from=build /src/main .
RUN chown -R appuser:appgroup /app/
USER 1000
EXPOSE 8080
CMD ["./main"]
