# WebSocket Protocol

## Connection

- **Endpoint**: `ws://127.0.0.1:8080`
- **Protocol**: JSON messages over WebSocket
- **Allowed Origins**:
  - `http://127.0.0.1:3000`
  - `http://localhost:3000`

## Message Format

All messages are JSON objects with a `type` field that identifies the message type.

## Client → Server Messages

Messages sent from the client to the server.

### `join`

Join a room. If the client is already in a room, they will leave the previous room first.

**Request:**

```json
{
  "type": "join",
  "name": "string",
  "room": "string"
}
```

**Fields:**

- `type` (string, required): Must be `"join"`
- `name` (string, required): User's display name (non-empty)
- `room` (string, required): Room identifier to join (non-empty)

**Response:**

- Server sends `joined` message to the sender
- Server broadcasts `user_joined` message to all other clients in the room

**Example:**

```json
{
  "type": "join",
  "name": "Alice",
  "room": "general"
}
```

---

### `leave`

Leave the current room.

**Request:**

```json
{
  "type": "leave"
}
```

**Fields:**

- `type` (string, required): Must be `"leave"`

**Response:**

- Server sends `left` message to the sender
- Server broadcasts `user_left` message to all other clients in the room

**Example:**

```json
{
  "type": "leave"
}
```

---

## Server → Client Messages

Messages sent from the server to the client.

### `joined`

Confirmation that the client successfully joined a room. Sent only to the client that joined.

**Message:**

```json
{
  "type": "joined",
  "room": "string"
}
```

**Fields:**

- `type` (string): Always `"joined"`
- `room` (string): Room identifier that was joined

**Example:**

```json
{
  "type": "joined",
  "room": "general"
}
```

---

### `left`

Confirmation that the client left a room. Sent only to the client that left.

**Message:**

```json
{
  "type": "left",
  "room": "string"
}
```

**Fields:**

- `type` (string): Always `"left"`
- `room` (string): Room identifier that was left

**Example:**

```json
{
  "type": "left",
  "room": "general"
}
```

---

### `user_joined`

Broadcast to all clients in a room when a user joins. Not sent to the user who joined.

**Message:**

```json
{
  "type": "user_joined",
  "name": "string"
}
```

**Fields:**

- `type` (string): Always `"user_joined"`
- `name` (string): Display name of the user who joined

**Example:**

```json
{
  "type": "user_joined",
  "name": "Alice"
}
```

---

### `user_left`

Broadcast to all clients in a room when a user leaves. Not sent to the user who left.

**Message:**

```json
{
  "type": "user_left",
  "name": "string"
}
```

**Fields:**

- `type` (string): Always `"user_left"`
- `name` (string): Display name of the user who left

**Example:**

```json
{
  "type": "user_left",
  "name": "Alice"
}
```

---

## Message Flow Examples

### Joining a Room

1. Client sends `join` message:

   ```json
   {"type": "join", "name": "Alice", "room": "general"}
   ```

2. Server responds to sender with `joined`:

   ```json
   {"type": "joined", "room": "general"}
   ```

3. Server broadcasts `user_joined` to other clients in room:

   ```json
   {"type": "user_joined", "name": "Alice"}
   ```

### Leaving a Room

1. Client sends `leave` message:

   ```json
   {"type": "leave"}
   ```

2. Server responds to sender with `left`:

   ```json
   {"type": "left", "room": "general"}
   ```

3. Server broadcasts `user_left` to other clients in room:

   ```json
   {"type": "user_left", "name": "Alice"}
   ```

---

## Error Handling

- Invalid JSON messages will result in connection errors
- Messages with unknown `type` will result in an error response
- Missing required fields in `join` messages will result in an error
- If a client disconnects unexpectedly, the server will automatically broadcast `user_left` to other clients in the room
