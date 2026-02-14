# Quick Reference Examples

## Minimal Configuration

```yaml
server:
  port: 8080

tables:
  - name: users
    primaryKey:
      field: userId
      pattern: "^[A-Za-z_][A-Za-z0-9._-]*$"
    schema:
      type: object
      properties:
        userId:
          type: string
          pattern: "^[A-Za-z_][A-Za-z0-9._-]*$"
        name:
          type: string
        email:
          type: string
        status:
          type: string
      required:
        - userId
        - name
    allowTableScan: true
    indexes:
      - name: by_status
        primaryKey:
          field: status
        allowIndexScan: true
```

## Example Requests

The examples below assume the server is running at `http://localhost:8080` with the configuration above.

### 1. Create an item

```bash
curl -X PUT http://localhost:8080/v1/users/data/alice/_item \
  -H "Content-Type: application/json" \
  -d '{"userId": "alice", "name": "Alice", "email": "alice@example.com", "status": "active"}'
```

Response (`200 OK`):

```json
{
  "userId": "alice",
  "name": "Alice",
  "email": "alice@example.com",
  "status": "active"
}
```

### 2. Get the item

```bash
curl http://localhost:8080/v1/users/data/alice/_item
```

Response (`200 OK`):

```json
{
  "userId": "alice",
  "name": "Alice",
  "email": "alice@example.com",
  "status": "active"
}
```

### 3. Patch the item

```bash
curl -X PATCH http://localhost:8080/v1/users/data/alice/_item \
  -H "Content-Type: application/json" \
  -d '{"email": "newalice@example.com"}'
```

Response (`200 OK`):

```json
{
  "userId": "alice",
  "name": "Alice",
  "email": "newalice@example.com",
  "status": "active"
}
```

### 4. Get the updated item

```bash
curl http://localhost:8080/v1/users/data/alice/_item
```

Response (`200 OK`):

```json
{
  "userId": "alice",
  "name": "Alice",
  "email": "newalice@example.com",
  "status": "active"
}
```

### 5. List items

```bash
curl http://localhost:8080/v1/users/_items?limit=10
```

Response (`200 OK`):

```json
{
  "items": [
    {
      "userId": "alice",
      "name": "Alice",
      "email": "newalice@example.com",
      "status": "active"
    }
  ]
}
```

### 6. Query an index

```bash
curl http://localhost:8080/v1/users/_index/by_status/active/_items
```

Response (`200 OK`):

```json
{
  "items": [
    {
      "userId": "alice",
      "name": "Alice",
      "email": "newalice@example.com",
      "status": "active"
    }
  ]
}
```

### 7. Delete the item

```bash
curl -X DELETE http://localhost:8080/v1/users/data/alice/_item
```

Response: `204 No Content` (empty body).
