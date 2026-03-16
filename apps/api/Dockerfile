FROM golang:1.24-bookworm AS deps

WORKDIR /workspace/apps/api

COPY apps/api/go.mod apps/api/go.sum ./
RUN go mod download

FROM deps AS dev

WORKDIR /workspace/apps/api
EXPOSE 8080

CMD ["go", "run", "./cmd/server"]

FROM deps AS build

WORKDIR /workspace

COPY apps/api /workspace/apps/api

RUN cd /workspace/apps/api && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/tether-api ./cmd/server

FROM gcr.io/distroless/base-debian12:nonroot AS runtime

WORKDIR /app

COPY --from=build /out/tether-api /app/tether-api

EXPOSE 8080

ENTRYPOINT ["/app/tether-api"]
