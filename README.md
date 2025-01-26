# Redis-From-Scratch

Redis built from scratch.

# <ins>Learnings</ins>

## <ins>I/O multiplexing</ins>

I/O multiplexing allows a program to monitor(actively checking for readiness of data) multiple I/O streams (e.g., sockets, files) simultaneously and take action only when data is available on one or more of them. This technique is especially useful for programs handling multiple I/O connections, such as servers managing multiple client connections.

It is more efficient because it avoids blocking a thread on a single I/O operation and reduces resource usage by eliminating the need to create multiple threads or processes for each I/O stream. Instead, a single thread can efficiently manage multiple I/O sources.

some blocking commands could be reading from a file, accept for a new socket connection, etc

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

## <ins>File Descriptor (FD)</ins>

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

## <ins>RESP</ins>

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
