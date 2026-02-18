# Contract Service

Ein Go-Service zum Anlegen von **Zeitscheiben** für Verträge mit Bearer-Token-Authentifizierung und PostgreSQL-Persistenz.

---

## Architektur

```
cmd/server/main.go          → Einstiegspunkt, Konfiguration, HTTP-Server
internal/
  handler/handler.go        → HTTP-Handler, Request-Validierung
  middleware/auth.go        → Bearer-Token-Middleware
  repository/repository.go  → PostgreSQL-Datenbanklogik
  model/model.go            → Datenstrukturen
migrations/001_init.sql     → DB-Schema
```

---

## Schnellstart (Docker)

```bash
# 1. Repo klonen und ins Verzeichnis wechseln
git clone ... && cd contract-service

# 2. Secret anpassen
#    In docker-compose.yml den API_BEARER_TOKEN-Wert ersetzen

# 3. Service starten (baut automatisch, migriert DB)
docker compose up --build
```

---

## Schnellstart (lokal)

```bash
# Voraussetzung: Go 1.22+, laufende Postgres-Instanz

# 1. Schema anlegen
psql "$DATABASE_URL" -f migrations/001_init.sql

# 2. Umgebungsvariablen setzen
cp .env.example .env
# .env anpassen

# 3. Abhängigkeiten laden und starten
go mod tidy
go run ./cmd/server
```

---

## Umgebungsvariablen

| Variable           | Pflicht | Beschreibung                              |
|--------------------|---------|-------------------------------------------|
| `DATABASE_URL`     | ✅      | PostgreSQL-Connection-String              |
| `API_BEARER_TOKEN` | ✅      | Statisches Secret für Bearer-Auth         |
| `PORT`             | ❌      | HTTP-Port (Standard: `8080`)              |

---

## API

### `POST /time-slices`

Legt eine neue Zeitscheibe für einen Vertrag an.

**Header**
```
Authorization: Bearer <API_BEARER_TOKEN>
Content-Type: application/json
```

**Request Body**
```json
{
  "contract_id":   "CONTRACT-42",
  "article_ids":   [10, 99, 55, 201],
  "validity_tag":  "2024-Q1",
  "invoice_date":  "2024-03-31T00:00:00Z"
}
```

| Feld           | Typ            | Beschreibung                                    |
|----------------|----------------|-------------------------------------------------|
| `contract_id`  | string         | Eindeutiger Vertragsidentifikator               |
| `article_ids`  | array of int64 | Liste der Artikel-IDs; es wird die höchste gespeichert |
| `validity_tag` | string         | Gültigkeits-Tag der Zeitscheibe                 |
| `invoice_date` | ISO 8601       | Rechnungsdatum; weicht es ab → alle bestehenden Scheiben werden aktualisiert |

**Response `201 Created`**
```json
{
  "message": "time slice created",
  "time_slice_id": 7
}
```

**Fehler**

| Status | Beschreibung                       |
|--------|------------------------------------|
| 400    | Ungültiger JSON-Body               |
| 401    | Kein Authorization-Header          |
| 403    | Ungültiger Token                   |
| 422    | Validierungsfehler (fehlendes Feld)|
| 500    | Datenbankfehler                    |

---

### `GET /health`

Health-Check-Endpunkt (kein Auth erforderlich).

```
HTTP/1.1 200 OK
{"status":"ok"}
```

---

## Beispiel-Curl

```bash
curl -X POST http://localhost:8080/time-slices \
  -H "Authorization: Bearer change-me-to-a-strong-secret" \
  -H "Content-Type: application/json" \
  -d '{
    "contract_id": "CONTRACT-42",
    "article_ids": [10, 99, 55, 201],
    "validity_tag": "2024-Q1",
    "invoice_date": "2024-03-31T00:00:00Z"
  }'
```

---

## Datenbankschema

```sql
CREATE TABLE time_slices (
    id              BIGSERIAL PRIMARY KEY,
    contract_id     TEXT        NOT NULL,
    top_article_id  BIGINT      NOT NULL,   -- höchste ID aus article_ids
    validity_tag    TEXT        NOT NULL,
    invoice_date    DATE        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Geschäftslogik (Transaktional)

1. Prüfen, ob für `contract_id` bereits ein `invoice_date` existiert.
2. Weicht das neue Datum vom gespeicherten ab → `UPDATE time_slices SET invoice_date = $new WHERE contract_id = $id`.
3. Neue Zeitscheibe mit `top_article_id = MAX(article_ids)` inserieren.
4. Alles in einer Postgres-Transaktion — entweder alles oder nichts.
