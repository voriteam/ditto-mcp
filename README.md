# ditto-mcp

A minimal Model Context Protocol server that exposes the
[Ditto HTTP API](https://docs.ditto.live/cloud/http-api/api/post-storeexecute)
as a single `execute_query` tool over streamable HTTP.

The server is a thin pass-through: MCP clients send a DQL `statement` (plus
optional named `args`), the server POSTs `{ statement, args }` to
`DITTO_API_URL` with `Authorization: Bearer $DITTO_API_KEY`, and the JSON
response is returned to the client. The server does not parse or filter
statements.

Written in Go using the official
[`github.com/modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk).

## Build

```bash
go mod tidy          # once, to materialize go.sum
go build ./...
```

## Run

```bash
DITTO_API_URL='https://<app-id>.cloud.ditto.live/api/v4/store/execute' \
DITTO_API_KEY='<api-key>' \
  ./ditto-mcp
```

## Docker

CI publishes `ghcr.io/voriteam/ditto-mcp:<short-sha>` and
`ghcr.io/voriteam/ditto-mcp:latest` on every push to `main` (see
`.github/workflows/ci.yml`). To build locally:

```bash
docker buildx build --platform linux/amd64,linux/arm64 \
  --build-arg COMMIT_SHA=$(git rev-parse HEAD) \
  -t ghcr.io/voriteam/ditto-mcp:latest \
  --load \
  .
```

Pass `--push` instead of `--load` to publish (requires `docker login ghcr.io`).
The `COMMIT_SHA` build arg is baked into the image as an env var and logged at
startup, so a running container's commit is visible in its logs.

## Environment variables

| Variable        | Required | Description                                                                                   |
| --------------- | -------- | --------------------------------------------------------------------------------------------- |
| `DITTO_API_URL` | Yes      | Full URL of the Ditto HTTP API execute endpoint (e.g. `https://<app-id>.cloud.ditto.live/api/v4/store/execute`). Operators choose `v4` vs. `v5` here. |
| `DITTO_API_KEY` | Yes      | Ditto API key; sent as `Authorization: Bearer $DITTO_API_KEY` on each request.                |
| `PORT`          | No       | Listen port. Defaults to `8002`.                                                              |

## Endpoints

- `POST`/`GET`/`DELETE /mcp` — MCP streamable-HTTP transport.
- `GET /healthz` — liveness probe, always returns `200 ok`.
