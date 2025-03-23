# Redis-From-Scratch

Redis built from scratch in Go.

## Features

### 1) Basic Commands:

- GET
- SET (with TTL)
- PING
- ECHO
- KEYS (only \* pattern)
- TYPE

### 2) Stream Commands:

- XADD - Add entries to a stream
- XRANGE - Get range of entries (inclusive of start/end IDs)
- XREAD - Read entries newer than given ID, blocking available.

### 3) Transaction Commands:

- MULTI - Start a transaction
- EXEC - Execute a transaction
- DISCARD - Discard a transaction
- WATCH - Watch keys for changes (optimistic locking) (CAS)
- UNWATCH - Stop watching keys

### 4) RESP Protocol:

- Full RESP V2 (Redis Serialization Protocol) support
- Handles RESP data types:
  - Simple Strings
  - Errors
  - Integers
  - Bulk Strings
  - Arrays

### 5) Concurrency:

- Supports multiple concurrent clients using go-routines
- Thread-safe operations with mutex locks

### 6) Persistence:

- RDB file support (read-only, no RDB file saving/creation)
- Automatic loading of RDB files on startup

## How to setup locally

To clone and run locally, follow these steps:

1. **Clone the repository**:
   ```bash
   git clone https://github.com/yourusername/Redis-From-Scratch.git
   cd Redis-From-Scratch
   ```
2. **Initialize Go Modules:**
   ```bash
   go mod init github.com/manish-singh-bisht/Redis-From-Scratch
   go mod tidy
   ```
3. **Build and run**
   ```bash
   go build -o rds
   ./rds
   ```

## Example

1. Use any Redis client to connect to the server on the port 9379.
2. Using redis-cli, connect using `redis-cli -p 9379`
3. Run the supported commands mentioned above.

### Basic Commands

```bash
# Start redis-cli
redis-cli

# Basic key-value operations
1. SET mykey "Hello World"
2. GET mykey
3. SET mykey-with-ttl "I will expire" EX 10  # Expires in 10 seconds
4. KEYS *  # List all keys
5. TYPE mykey  # Get type of key

# Simple commands
6. PING  # Returns PONG
7. ECHO "Hello"  # Returns Hello
```

### Streams

```bash
# Add entries to a stream
1. XADD mystream * name "John" age "25"  # Auto-generated ID

# Read from stream
2. XRANGE mystream - +  # Read all entries

# Read new entries
3. XREAD STREAMS mystream $  # Read entries newer than last seen
4. XREAD BLOCK 3000 STREAMS mystream $  # Block for 3 seconds waiting for new entries
```

### Transactions

```bash
# Basic transaction
1. MULTI  # Start transaction
SET user:1 "John"
SET user:2 "Jane"
2. EXEC  # Execute transaction

# Transaction with optimistic locking
3. WATCH user:1  # Watch for changes
MULTI
SET user:1 "John Doe"
EXEC  # Will fail if user:1 was modified by another client

```

## Contributing

We'd love your help! Here's a simple guide to contributing:

1. Fork & Clone
2. Create a Branch
3. Make Your Changes
4. Open a Pull Request
