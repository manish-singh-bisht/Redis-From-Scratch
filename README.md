# Redis-From-Scratch

Redis built from scratch.

## Features

### 1) Commands:

- GET
- SET (with TTL)
- PING
- ECHO
- KEYS (only \* pattern)

### 2) RESP Parser:

- Supports RESP

### 3) Concurrency:

- Supports multiple concurrent clients using go-routines
- Event loop(soon..)

### 4) RDB Parser:

- Parses and reads RDB files during server startup.
