# Redis-From-Scratch

Redis built from scratch.

## Features

### 1) Basic Commands:

- GET
- SET (with TTL)
- PING
- ECHO
- KEYS (only \* pattern)

### 2) Concurrency:

- Supports multiple concurrent clients using go-routines
- Event loop(soon..)

### 3) RDB Parser:

- Parses and reads RDB files during server startup.
