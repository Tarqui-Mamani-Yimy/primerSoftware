# Handoff: Backend foundation for diagram realtime collaboration

Status: BACKEND FOUNDATION CORRECTED — only Go + migrations touched. Web and
mobile integration is the next stage (Copilot after this work unit lands).

The contract documented here is the **canonical** one; web/mobile clients
must read it before opening WS or sending a PUT. Any code-side difference
between the on-disk implementation and the contract is a bug, not a
behavioural choice.

## Context

- Repo `ai-uml-architect`, branch `feat/relationship-reconfiguration`.
- Backend runtime: `github.com/ai-uml-architect/gobackend` at `backend/`,
  Go 1.26, `pgx/v5`, `golang.org/x/crypto`, `gorilla/websocket`. PG dev
  database at `postgres://yimy:***@localhost:5432/uml_architect?sslmode=disable`.
  Migrations are Flyway-style embedded (`V1..V6`) and journalled via
  `go_schema_migrations`.
- Current REST surface (from previous work units, untouched here):
  - `POST /api/v1/auth/login`, `GET /api/v1/projects`,
    `POST /api/v1/projects`, `POST /api/v1/projects/join`,
    `GET|POST /api/v1/projects/{projectId}/diagrams`,
    `GET|PUT /api/v1/projects/{projectId}/diagrams/{id}`,
    `POST /api/v1/projects/{projectId}/diagrams/{id}/checkpoints`,
    `GET /api/v1/projects/{projectId}/diagrams/{id}/versions`,
    `POST /api/v1/projects/{projectId}/diagrams/{id}/versions/{n}/restore`,
    `POST /api/v1/projects/{projectId}/diagrams/{id}/ws-ticket`
    (added by this work unit; one-shot Bearer-authenticated ticket).
- Autosave is split into working document (PUT) and explicit checkpoints
  (`POST /checkpoints`); concurrency is gated by the body `version`,
  the `If-Match` header, and the new `X-Diagram-Review` header that
  carries the `review_number` baseline.
- CORS allowed headers added for `If-Match` / `X-Checkpoint-Message` /
  `X-Diagram-Review` (commit `c751974`).
- The WebSocket route does NOT require a Bearer header. Clients exchange
  the Bearer token for a one-shot ticket via REST, then open the WS
  with `?ticket=<base64>` so the browser avoids double authentication.

## Goals and explicit non-goals

Goals
- Stop silent overwrites with a monotonic work `review_number` independent
  of `version_number`. Autosave and explicit checkpoints honor it.
- Atomic CAS at save time: single PG transaction for diagrams row,
  monotonic review bump, no torn writes.
- Atomic checkpoint without races: diagrams row update + review bump +
  diagram_versions insert inside one PG transaction
  (SERIALIZABLE; SQLSTATE 40001 is mapped to a service-level 409).
- Authenticated, per-diagram-scoped WebSocket: presence (join/leave,
  heartbeat, snapshot) and change notifications with actor + review.
  The server notifies — it never ships the new document over the
  socket, because there is no CRDT/OT and no automatic merge.

Non-goals
- No CRDT, no OT, no automatic merging. Concurrent readers must
  REST-GET.
- No Live Room / meetings / voice / Whisper. Hub only knows about
  diagrams.
- No server-side fan-out of the document body over WS.

## Migrations (V6)

V6 adds the per-diagram work counter to two tables, in one file so the
order is unambiguous:

- `diagrams.review_number BIGINT NOT NULL DEFAULT 1`.
- `diagram_versions.review_number BIGINT NOT NULL DEFAULT 1` plus a
  backfill: existing version rows are re-stamped so the counter grows
  monotonically per diagram (1, 2, 3, …). The backfill is its own
  UPDATE statement written as `WITH ranked AS (... ROW_NUMBER() OVER
  PARTITION BY diagram_id ORDER BY version_number ...) UPDATE …v…`,
  so concurrent INSERTs at the moment of the ALTER cannot end up with
  NULL on either column.
- `CREATE INDEX IF NOT EXISTS idx_diagram_versions_diagram_review ON
  diagram_versions(diagram_id, review_number DESC)`. Drivers querying
  "what is the latest review number for this diagram" hit this index.
- Both columns are nullable in the intermediate `ALTER TABLE` step
  (so the rewrite never errors on legacy rows), then backfilled, then
  tightened to `NOT NULL DEFAULT 1`.
- Idempotency: every statement is guarded by `IF NOT EXISTS` or `IS
  NULL` so re-running V6 on a partially-migrated DB is a no-op.

Backfill algorithm (idempotent): for each diagram, set
`diagram_versions.review_number = ROW_NUMBER() OVER (PARTITION BY
diagram_id ORDER BY version_number)` and `diagrams.review_number =
COALESCE(MAX(review_number), 1)`. The migration file is the single
source of truth for both columns; future V7+ migrations must not
re-add columns already declared in V6.

## REST contract (review_number)

The `UMLDiagramDocument` JSON gains an integer `reviewNumber` (orthogonal
to `version`).
- PUT and POST /checkpoints both honor an **explicit CAS baseline** sent
  via the new `X-Diagram-Review` header. The header is the only
  authoritative baseline path: clients should send it and ignore the
  body field. Body `reviewNumber` is kept for legacy clients but the
  service is **strict** when it is absent or 0.
- Strict semantics: if neither the body nor the header provides a
  positive integer, the service applies the LIVE review number to the
  save and bumps it; this is treated as a "legacy write" (logged on
  the broadcaster as `kind: "working-document"` for autosave and
  `kind: "checkpoint"` for explicit save) and never produces a 409.
  Sending stale 0 always bumps the counter; it never causes a 409.
- Stale positive integer in either the body or the header → 409 with
  `{message, current: {…full document with the live reviewNumber…}}`.
  The 409 envelope is byte-compatible with autosave conflicts added in
  commit `1891a8d`.
- `UMLDiagramVersion` response and `DiagramSummary` are unchanged in
  shape; the `reviewNumber` field on `UMLDiagramVersion` carries the
  per-row counter at the moment of the checkpoint append.

## WS ticket endpoint (REST, Bearer required)

`POST /api/v1/projects/{projectId}/diagrams/{id}/ws-ticket`
- Auth: same Bearer as the rest of the API. Membership gate enforced
  before issuing the ticket.
- Response: `{ticket: "<base64>", expiresAt: "<ISO-8601 UTC>",
  hostname: "<config.allowed-hosts entry that matched the request>"}`.
- Ticket payload: `random 32 bytes + HMAC-SHA256(payload, serverSecret)
  + exp (Unix seconds) + diagramID`. The server keeps only the HMAC
  key + an in-process map of un-redeemed tickets; a successful WS
  upgrade **consumes** the ticket (single-use).
- Ticket lifetime: 60 seconds. The ticket store prunes expired entries
  on every `JoinTicket` attempt.
- CORS allows `GET /api/v1/projects/.../ws` only for the configured
  allowed origins; `ws://...` and `wss://...` are mirrored in
  `Access-Control-Allow-Origin` for the WebSocket-upgrade preflight.

## WebSocket contract (presence + change signal)

- Endpoint: `GET /api/v1/projects/{projectId}/diagrams/{id}/ws?ticket=<base64>`.
- Auth: the upgrade consumes the one-shot ticket (validated, removed,
  then re-validates the user's project membership against the store).
  The Bearer header is NOT required for the upgrade.
- The route refuses (HTTP 403) if the ticket is missing, expired, or
  already redeemed; it refuses the WS upgrade if membership fails.
- Frame format: one JSON message per WebSocket frame, UTF-8 text.

Server → client frames (typed):
- `hello` (sent once on connect, BEFORE `snapshot`): the requesting
  client's identity as `{userId, displayName, reviewNumber}` so the
  UI can render "you are online as <name>".
- `snapshot` (sent exactly once on join): full working document plus
  the current roster
  `actors: [{userId, displayName, reviewNumber, connectedAt}]`. The
  roster is sorted by userId for determinism in tests.
- `presence` (full roster after every join/leave): same `actors`
  shape as `snapshot`. Sent after `snapshot` after the first join,
  and on every subsequent roster change.
- `joined` / `left` (one event per peer): actor object as above.
- `change` (after the service confirms an autosave or checkpoint):
  `{actor: {userId, displayName}, reviewNumber, kind:
  "working-document"|"checkpoint", createdAt,
  versionNumber?}`. The server does NOT ship the new document; clients
  REST-GET to refresh.

Client → server frames:
- `heartbeat` — JSON-only. The read deadline also acts as a transport
  keepalive (gorilla/websocket ping). Interval contract: 20s suggested;
  idle eviction after 60s of silence (the hub's `freshnessWindow`).
- `bye` — optional explicit leave; the read-loop close already cleans
  up.

Presence scope: a client only ever sees presence for the diagram it
joined. The hub is keyed by `(projectId, diagramId)` and refuses any
fan-out across diagrams.

## Hub semantics

- In-memory, single-instance. Persistence is not required for v1: a
  process restart ends all active presences, which is acceptable for
  collaboration on a single backend node. Multi-instance fan-out is
  out of scope here.
- Concurrency:
  - Per-client `lastSeen` is guarded by `c.mu`; the room roster read
    uses `LastSeenOf(c)` everywhere.
  - `byClient` is unexported and only touched under `h.mu`. Leave
    handling goes through `h.leave` channel which is drained in
    `Run`; the synchronous `handleLeave` is only used when the
    channel is full so the contract is always "either async via the
    channel or synchronous, never racy".
  - Slow consumers are kicked explicitly: a write pump that detects
    the outbound buffer at capacity closes the WS and broadcasts a
    `left` envelope with the same `actor` payload; the consumer is
    told via close-code 1008 (policy violation) with message
    "slow consumer".
- Backpressure: per-client buffer size 16 envelopes; on overflow the
  write pump closes with close-code 1008.
- Origin enforcement: `CheckOrigin` consults the Hub's configured
  allow-list (defaults to `[]`, deny-all). WS upgrades from origins
  not on the list close with 1008.
- Reconnect cleanup: when the WS read loop exits for any reason, the
  hub removes the client and broadcasts a `left` event. Heartbeat
  eviction runs every `freshnessWindow / 6`.

## Ordered commits (landed this round)

The work-unit commits in this branch land in this order:

1. `3c908a2 feat(migrate): add per-diagram review_number to diagrams and
   diagram_versions` — V6 migration adds both columns, indexes
   `diagram_versions(diagram_id, review_number DESC)` so future
   review-aware reads hit the index.
2. `584954e feat(store): atomic AppendCheckpoint rewrites diagrams row +
   version row in one tx` — Service `appendCheckpointAt` runs the
   diagrams-row update + diagram_versions insert + review bump inside
   one PG transaction. Memory store mirrors the contract.
3. `13526cb feat(service): legacy write bumps silently; positive baseline
   is strict CAS` — `UpdateDiagram` and `CreateCheckpoint` honour two
   baselines: `nil` or `0` reads the live review_number and bumps it
   silently (legacy clients never 409); `> 0` is strict CAS and
   returns 409 with the live document on mismatch. Broadcasts Kind
   `working-document` for autosave and `checkpoint` for explicit
   saves. Router tests updated for the new POST /checkpoints route.
4. `84fbdfc feat(realtime): hub foundation with Origin gate, ticket
   signer, focused tests` — realtime package on top of
   gorilla/websocket (hub.go + room.go + client.go +
   hub_lifecycle.go + ws_upgrade.go + ticket.go + the matching tests).
   `HubOptions{AllowOrigin, Tickets}` configures the upgrade half the
   httpapi.Router ends up wiring. TicketSigner issues/trims HMAC
   signatures with cap MaxTicketTTL=5m. Tests cover the round-trip,
   past-exp rejection, signature mismatch, tampered body, malformed
   input, TTL clamping, hub authorization, snapshot delivery,
   diagram-changed broadcast (no echo back to originator), and the
   strict Origin allow-list.
5. `3beedbe feat(realtime): wire main + adapter + ticket secret into
   config` — `main.go` boots the hub + signer + server with the
   4-arg `NewServer(svc, corsOrigin, hub, tickets)` constructor;
   `config.go` exposes `RealtimeTicketSecret` (REALTIME_TICKET_SECRET
   env, dev fallback clearly marked). The adapter in
   `internal/httpapi/realtime_adapter.go` keeps realtor decoupled from
   `service`.

The 409 envelope, the CORS preflight headers, the If-Match /
X-Diagram-Review parsers, and the realtime REST route
`POST .../realtime-tickets` (mints a short-lived signed bearer between
bearer login and `?ticket=...` WS upgrade so the browser never sends
the long-lived token in a non-standard header) ship in the same round.

## Constraints

- No CRDT/OT, no merge logic, no automatic document splicing.
- No Live Room, voice, meetings, Whisper in this scope.
- No changes to `frontend/`, `mobile/`, or any foreign/untracked file.
  See `odd/tasks/go-backend-and-flutter-mobile.md` for parallel work.
- Conventional commits only, no `Co-Authored-By`.
- Keep the existing REST contract byte-compatible where possible: the
  body field `reviewNumber` is preserved for legacy clients but the
  service is strict (legacy writes never 409; positive stale always
  409).

## Status

- 2026-09-19: BACKEND only. Tracker corrected for the v1 contract;
  commits 1–7 land in order. Web/mobile is the next stage (Copilot).
