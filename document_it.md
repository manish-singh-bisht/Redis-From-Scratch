## <ins>1. I/O multiplexing</ins>

I/O (Input/Output) operations refer to any action where a program reads data from or writes data to an external source, such as:

1. Reading from a file
2. Writing to a file
3. Reading data from a network socket (e.g., receiving HTTP requests)
4. Writing data to a network socket (e.g., sending HTTP responses)
5. Reading/writing from a database
6. Reading input from a keyboard or other devices

I/O multiplexing allows a program to monitor(actively checking for readiness of data) multiple I/O streams (e.g., sockets, files) simultaneously and take action only when data is available on one or more of them. This technique is especially useful for programs handling multiple I/O connections, such as servers managing multiple client connections.

It is more efficient because it avoids blocking a thread on a single I/O operation and reduces resource usage by eliminating the need to create multiple threads or processes for each I/O stream. Instead, a single thread can efficiently manage multiple I/O sources.

some blocking commands could be reading from a file, accept for a new socket connection, etc

Event loop is a program that executes tasks in a non blocking way for multiple clients. I/O multiplexing is the technique used to make the event loop work.

### Common system calls used for I/O multiplexing:

0.  select: Monitors multiple file descriptors for readiness.
1.  poll: Similar to select, but supports larger numbers of file descriptors.
2.  epoll (Linux-specific): More scalable and efficient than select or poll for large numbers of file descriptors.
3.  kqueue (BSD and macOS): Another efficient mechanism similar to epoll.

### Example Use Case of I/O multiplexing

A web server handling multiple client connections:

Each client connection is represented by a socket, which has an associated file descriptor.
Instead of dedicating a separate thread or process for each connection, the server uses I/O multiplexing to monitor all sockets simultaneously.
When data arrives on one socket, the server processes it while continuing to monitor the others.

## <ins>2. File Descriptor (FD)</ins>

A file descriptor is a non-negative integer that the operating system assigns to a file, socket, or any other I/O resource when it is opened by a program.
It serves as an index or handle that the program can use to refer to the open file/resource during subsequent operations (like reading, writing, or closing).

### Reserved File Descriptors:

0. stdin (Standard Input) - Typically connected to the keyboard or input source.
1. stdout (Standard Output) - Typically connected to the terminal or output destination.
2. stderr (Standard Error) - Typically used for error messages, separate from standard output.

### Sequential Allocation:

After the reserved FDs (0, 1, 2), additional FDs are allocated sequentially for each new file, socket, or resource that is opened.
Example: If you open three files in a program, their file descriptors might be 3, 4, and 5.

### Scope of File Descriptors:

File descriptors are local to the process that created them.
They are inherited by child processes created using system calls like fork (unless explicitly closed or marked as non-inheritable).

### Operations Using File Descriptors:

Read/Write: You can use system calls like read(fd, ...) or write(fd, ...).
Close: When done, you should release the file descriptor using close(fd) to free up resources.

### Special Case with Duplication:

You can duplicate file descriptors using system calls like dup or dup2, which create new FDs that refer to the same underlying resource.

## <ins>3. RESP</ins>

The RESP (Redis Serialization Protocol) is a simple protocol used by Redis to serialize data exchanged between clients and servers. RESP is used to encode and decode data such as strings, integers, arrays, and more. It is designed to be easy to parse and human-readable.

In RESP, the first byte of data determines its type.Subsequent bytes constitute the type's contents.
Redis generally uses RESP as a request-response protocol in the following way:

1. Clients send commands to a Redis server as an array of bulk strings. The first (and sometimes also the second) bulk string in the array is the command's name. Subsequent elements of the array are the arguments for the command.
2. The server replies with a RESP type. The reply's type is determined by the command's implementation and possibly by the client's protocol version.

### RESP Input Example for some types:

- Simple String==> +string\r\n
- Simple Error==> -ERR unknown command 'asdf'
- Bulk String==> $5\r\nhello\r\n
- Arrays==> \*2\r\n$5\r\nhello\r\n$5\r\nworld\r\n

## <ins>4. Persistence</ins>

Persistence refers to the writing of data to durable storage, such as a solid-state disk (SSD).

1. RDB (Redis Database): RDB persistence performs point-in-time snapshots of your dataset at specified intervals. A dataset refers to all the key-value pairs stored in memory at a given time

   An RDB file is a point-in-time snapshot of a Redis dataset. When RDB persistence is enabled, the Redis server syncs its in-memory state with an RDB file, by doing the following:

   1. On startup, the Redis server loads the data from the RDB file.
   2. While running, the Redis server periodically takes new snapshots of the dataset, in order to update the RDB file.
   3. Redis RDB snapshots can be triggered automatically (based on time + number of changes) , manually , or disabled them.

   RDB files are perfect for backups.

2) AOF (Append Only File): AOF persistence logs every write operation received by the server. These operations can then be replayed again at server startup, reconstructing the original dataset. Commands are logged using the same format as the Redis protocol itself.
3) No persistence: You can disable persistence completely. This is sometimes used when caching.
4) RDB + AOF: You can also combine both AOF and RDB in the same instance.

### Endianness

1.  <ins>BYTE</ins>

    Each bit in a byte is a placeholder that can be either 0 or 1, forming different combinations. Since there are 8 bits, the total number of unique combinations is:
    2^8=256

    Since counting starts from 0, the range of values that can be represented in an 8-bit system is:
    0 (00000000)to255 (11111111)
    How?? just go from 2^0 till 2^7 and add them , you will get 255, we cannot store 256 because it will require more than 8 bit ,thus we cannot store number above 255 in a single byte.

    1. Binary: 00000000 (0) to 11111111 (255)
    2. Decimal: 0 to 255
    3. Hexadecimal: 0x00 to 0xFF

2.  <ins>Endianness</ins>

    1.  Endianness is the order in which multi-byte numbers are stored in computer memory.

        Since a single byte (8 bits) can only store values from 0 to 255, larger numbers (like 16-bit, 32-bit, or 64-bit values) require multiple bytes. The way these bytes are arranged in memory is called endianness.

        <ins>Types of Endianness</ins>

        1. Little-Endian (Used by Intel CPUs)

           1. The least significant byte (LSB) is stored at the lowest memory address.
           2. Bytes are arranged in order of least to most significant.

        2. Big-Endian (Used in Networks, Older Macs, Some CPUs)

           1. The most significant byte (MSB) is stored at the lowest memory address.
           2. Bytes are arranged in order of most to least significant.

    2.  LSB,MSB

        1. LSB (Least Significant Bit): Smallest impact(2^0), determines odd/even.
        2. MSB (Most Significant Bit): Largest impact(2^7), can indicate sign (positive/negative).
        3. Endianness is about how bytes are stored, not individual bits!
        4. Example:
           1. 01111111 = 127 (Positive, MSB = 0, odd, LSB = 1)
           2. 10000000 = -128 (Negative, MSB = 1, even , LSB = 0)

3.  RDB file format - https://rdb.fnordig.de/file_format.html

## <ins> 5. Redis Commands </ins>

1. "PING": responds with "PONG"
2. "ECHO": echoes a message, that is return what is passed in
3. "SET": sets a key to a value, with optional expiration time, updates the value if the key already exists
4. "GET": gets a value from a key
5. "CONFIG": gets the configuration of the server
6. "KEYS": returns all the keys that match the pattern
7. "TYPE": returns the type of the key,
   1. return stream if the key is a stream key
8. "XADD": adds a new entry to a stream, creates a stream if it doesn't exist
9. "XRANGE":

   1. gets a range of entries from a stream,
   2. inclusive of the start and end IDs,
   3. takes in start and end IDs as arguments,
   4. cannot read from multiple streams
   5. "-" as the start id, means bring all the entries from the start of the stream
   6. "+" as the end id, means bring all the entries till the end of the stream

10. "XREAD":

    1. gets a range of entries from a stream
    2. that are strictly greater than the start id,
    3. exclusive of start id, takes in start id as argument,
    4. can also read from multiple streams(this is good when we want to read from multiple streams using just one command)
    5. also has blocking options(that is the command is blocked until the given time, in ms, specified in command and during that time if entries come they will be listened nearly instantly.)>
       1. block ms, blocks util that msTime and any new entries will be listened nearly instantly.
       2. block with 0ms, will be blocked forever and any new entries will be listened nearly instantly.
       3. $ as the id tells redis to read from the new entries after this xread command has been executed.

11. "INCR": increments the value of a key by 1, if the key doesn't exist, it will be set to 1

### <ins>6. Redis</ins>

Redis is designed for both speed and durability, combining the best of in-memory data storage with optional persistence.

1. In-Memory for Speed – All data is stored in RAM, making read and write operations extremely fast.

2. Single-Threaded for Efficiency – Avoids context switching overhead of multi-threaded architectures, relying on efficient I/O multiplexing.

3. Optimized Data Structures – Uses lists, sets, hash tables, and sorted sets, ensuring O(1) or O(log N) time complexity for most operations.

4. Persistence Mechanisms for Durability:

   1. RDB (Redis Database Backup): Takes snapshots at specified intervals and saves them to disk.
   2. AOF (Append-Only File): Logs every write operation, replaying them at startup to restore data.
   3. Hybrid Approach: You can use both RDB and AOF for better reliability.

   4. Restores Data on Restart – If persistence is enabled, Redis loads data from RDB or AOF on startup.

5. Non-Blocking Replication – Redis supports asynchronous replication, ensuring that read operations are not slowed down by replication processes.

### <ins>7. Redis Transactions</ins>

Redis Transactions execute a group of commands in a single step using MULTI, EXEC, DISCARD, and WATCH.

1. Key Guarantees:

   1. Isolation: All commands in a transaction run sequentially, without interference from other clients.
   2. Execution Control: If a client disconnects before EXEC, no operations are performed. Once EXEC is called, all commands execute.
   3. For AOF persistence, Redis writes transactions in a single syscall. If a crash causes partial writes, redis-check-aof can repair the log to restore consistency.

2. Redis transactions are executed serially, one at a time, ensuring that no two transactions interfere with each other. A request sent by another client will never be served in the middle of the execution of a Redis Transaction.

3. Transactions in Redis are initiated with the MULTI command, followed by a series of operations, and concluded with EXEC to execute all queued commands.

4. If an error occurs in one of the commands inside a transaction, Redis does not roll back the already executed commands; all valid commands still execute.

5. The DISCARD command can be used to abort a transaction before execution, clearing all queued commands.

6. The WATCH command can be used for optimistic locking, allowing conditional execution of a transaction only if a watched key remains unchanged.

   1. WATCH is used to provide a check-and-set (CAS) behavior to Redis transactions. CAS is, read the value, perform calculations, and before writing the value back, check if the value has changed, if it has, then the transaction is aborted, else the transaction is executed.

   2. WATCHed keys are monitored in order to detect changes against them. If at least one watched key is modified before the EXEC command, the whole transaction aborts, and EXEC returns a Null reply to notify that the transaction failed.

   3. When EXEC is called, all keys are UNWATCHed, regardless of whether the transaction was aborted or not. Also when a client connection is closed, everything gets UNWATCHed.

7. Redis transactions do not support rollback like SQL databases, so careful validation of commands is necessary before execution.
