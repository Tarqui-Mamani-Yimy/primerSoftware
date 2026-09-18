# UML Architect API

## Local requirements
- JDK 21
- Docker and Docker Compose

```bash
cd backend
docker compose up -d
./mvnw spring-boot:run
```

PostgreSQL listens on `localhost:5432` by default. **Flyway is the sole schema authority**: migrations under `src/main/resources/db/migration` create the schema and local development seed. Do not run separate schema SQL.

## Development login
The initial local-only seed creates `ana@example.com`, `bruno@example.com`, and `camila@example.com`; each password is `Password123!`. Passwords are BCrypt hashes, never plaintext. Replace this development provisioning before production.

## API contract
`POST /api/v1/auth/login` returns an opaque bearer token. Send `Authorization: Bearer <accessToken>`. Authenticated identity is derived only from this token.

- `GET /api/v1/projects` lists assigned projects.
- `/api/v1/projects/{projectId}/diagrams` lists, creates, updates and versions diagrams.

A membership is required for each project/diagram operation. Every write saves an immutable JSONB snapshot with `schemaVersion`, `classes`, and `relationships`.
