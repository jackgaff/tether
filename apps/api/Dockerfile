FROM golang:1.24-bookworm

WORKDIR /workspace/apps/api

COPY apps/api/go.mod ./
RUN go mod download

EXPOSE 8080

CMD ["go", "run", "./cmd/server"]

