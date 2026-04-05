# Signal CLI REST API

> **This is a fork of [bbernhard/signal-cli-rest-api](https://github.com/bbernhard/signal-cli-rest-api) with significant changes:**
> - **Runs exclusively in json-rpc-native mode** — no JVM, no MODE env var, single fast execution path
> - **Redesigned API** — all account-specific routes live under `/v1/accounts/{number}/...`
> - **Message polling** — `GET /v1/accounts/{number}/messages` returns unread messages from the DB and marks them retrieved
> - **WebSocket stream** — `GET /v1/accounts/{number}/messages/stream` streams live messages
> - **PostgreSQL message storage** — received messages are persisted to PostgreSQL and queryable via `GET /v1/accounts/{number}/messages/history`
> - **GitHub Container Registry** — Docker images published to `ghcr.io/strenkml/signal-cli-rest-api`

---

This project wraps [signal-cli](https://github.com/AsamK/signal-cli) in a REST API running inside Docker.

## Getting Started

### 1. Start a container

```bash
docker run -d --name signal-api --restart=always -p 8080:8080 \
  -v $HOME/.local/share/signal-api:/home/.local/share/signal-cli \
  ghcr.io/strenkml/signal-cli-rest-api:latest
```

### 2. Link your Signal number

Open `http://localhost:8080/v1/accounts/{number}/qr-code?device_name=signal-api` in your browser, then scan the QR code in Signal on your phone under _Settings → Linked devices_.

### 3. Send a test message

```bash
curl -X POST -H "Content-Type: application/json" \
  'http://localhost:8080/v1/accounts/+441234567890/messages' \
  -d '{"message": "Hello from the API!", "recipients": ["+449876543210"]}'
```

---

## Docker Compose

### Basic

```yaml
services:
  signal-cli-rest-api:
    image: ghcr.io/strenkml/signal-cli-rest-api:latest
    ports:
      - "8080:8080"
    volumes:
      - "./signal-cli-config:/home/.local/share/signal-cli"
```

### With PostgreSQL message storage

```yaml
services:
  signal-cli-rest-api:
    image: ghcr.io/strenkml/signal-cli-rest-api:latest
    environment:
      - DATABASE_URL=postgres://user:password@db:5432/signal
    ports:
      - "8080:8080"
    volumes:
      - "./signal-cli-config:/home/.local/share/signal-cli"

  db:
    image: postgres:16
    environment:
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=signal
    volumes:
      - "./postgres-data:/var/lib/postgresql/data"
```

---

## API Overview

All account-specific endpoints are under `/v1/accounts/{number}/...`. The `{number}` is your registered Signal number in international format (e.g. `+441234567890`).

### Messages

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/accounts/{number}/messages` | Send a message |
| `GET` | `/v1/accounts/{number}/messages` | Poll unread messages (marks them retrieved) |
| `GET` | `/v1/accounts/{number}/messages/stream` | WebSocket stream of live messages |
| `GET` | `/v1/accounts/{number}/messages/history` | Query full message archive |
| `POST` | `/v1/accounts/{number}/messages/remote-delete` | Remote-delete a sent message |

### Registration & Devices

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/accounts/{number}/register` | Register a phone number |
| `POST` | `/v1/accounts/{number}/register/verify/{token}` | Verify registration |
| `DELETE` | `/v1/accounts/{number}` | Unregister a number |
| `GET` | `/v1/accounts/{number}/qr-code` | QR code PNG for device linking |
| `GET` | `/v1/accounts/{number}/qr-code/raw` | Raw device link URI |
| `GET` | `/v1/accounts/{number}/devices` | List linked devices |
| `POST` | `/v1/accounts/{number}/devices` | Link a device by URI |
| `DELETE` | `/v1/accounts/{number}/devices/{deviceId}` | Remove a linked device |
| `DELETE` | `/v1/accounts/{number}/local-data` | Delete local account data |

### Groups, Contacts, Identities

Standard CRUD operations under `/v1/accounts/{number}/groups`, `/v1/accounts/{number}/contacts`, and `/v1/accounts/{number}/identities`. See the Swagger UI for full details.

### Other

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/accounts` | List all registered accounts |
| `GET` | `/v1/about` | API version info |
| `GET` | `/v1/health` | Health check |
| `GET` | `/v1/attachments` | List attachments |
| `GET` | `/v1/attachments/{id}` | Serve an attachment |
| `DELETE` | `/v1/attachments/{id}` | Delete an attachment |
| `GET` | `/v1/search` | Search for registered numbers |
| `GET/POST` | `/v1/configuration` | Get/set API logging configuration |

Full interactive docs are available at `http://localhost:8080/swagger/index.html`.

---

## Message Storage

When `DATABASE_URL` is set, every received Signal message is automatically saved to PostgreSQL. The schema is created on startup — no migrations needed.

### Polling unread messages

```
GET /v1/accounts/{number}/messages
```

Returns all messages that have not yet been retrieved and marks them as retrieved. Messages are **never deleted** from the database — use the `history` endpoint to query the full archive.

Returns `503` when `DATABASE_URL` is not set.

### Querying message history

```
GET /v1/accounts/{number}/messages/history
```

| Parameter | Description | Default |
|-----------|-------------|---------|
| `sender` | Filter by sender phone number | — |
| `group_id` | Filter by group ID | — |
| `start_time` | Minimum timestamp (ms since epoch) | — |
| `end_time` | Maximum timestamp (ms since epoch) | — |
| `envelope_type` | Comma-separated types to include | `dataMessage` |
| `retrieved` | Filter by retrieved flag (`true`/`false`) | all |
| `limit` | Max results (hard cap: 1000) | `100` |
| `offset` | Pagination offset | `0` |

### WebSocket stream

```
GET /v1/accounts/{number}/messages/stream
```

Upgrade to WebSocket to receive messages in real-time as they arrive. Does not require `DATABASE_URL`.

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string — enables message storage | unset |
| `RECEIVE_WEBHOOK_URL` | Forward received messages to this HTTP endpoint | unset |
| `PORT` | HTTP port | `8080` |
| `LOG_LEVEL` | Log level: `debug`, `info`, `warn`, `error` | `info` |
| `SWAGGER_HOST` | Host shown in Swagger UI | container IP:PORT |
| `SWAGGER_IP` | IP shown in Swagger UI | container IP |
| `SWAGGER_USE_HTTPS_AS_PREFERRED_SCHEME` | Use HTTPS in Swagger UI | `false` |
| `DEFAULT_SIGNAL_TEXT_MODE` | Default text mode for send: `normal` or `styled` | `normal` |
| `SIGNAL_CLI_CONFIG_DIR` | Path to signal-cli config inside container | `/home/.local/share/signal-cli` |
| `SIGNAL_CLI_UID` | UID of the signal-api user | `1000` |
| `SIGNAL_CLI_GID` | GID of the signal-api group | `1000` |
| `SIGNAL_CLI_CHOWN_ON_STARTUP` | Chown config dir on startup | `true` |
| `JSON_RPC_IGNORE_ATTACHMENTS` | Skip auto-downloading attachments | `false` |
| `JSON_RPC_IGNORE_STORIES` | Skip auto-downloading stories | `false` |
| `JSON_RPC_IGNORE_AVATARS` | Skip auto-downloading avatars | `false` |
| `JSON_RPC_IGNORE_STICKERS` | Skip auto-downloading sticker packs | `false` |
| `JSON_RPC_TRUST_NEW_IDENTITIES` | Identity trust policy: `on-first-use`, `always`, `never` | `on-first-use` |

---

## Plugins

Custom endpoints can be registered without forking the project. Set `ENABLE_PLUGINS=true` and point `SIGNAL_CLI_REST_API_PLUGIN_SHARED_OBJ_DIR` at your plugin directory. See the [upstream plugin docs](https://github.com/bbernhard/signal-cli-rest-api/tree/master/plugins) for details.
