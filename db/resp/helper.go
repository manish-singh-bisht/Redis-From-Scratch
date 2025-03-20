package RESP

import (
	"strconv"
)

/*
 	* IsArray checks if the message is an array
	* @return bool - true if the message is an array
*/
func (r *RESPMessage) IsArray() bool {
	return r.RESPType == Array
}

/*
 	* IsBulkString checks if the message is a bulk string
	* @return bool - true if the message is a bulk string
*/
func (r *RESPMessage) IsBulkString() bool {
	return r.RESPType == BulkString
}

/*
 	* IsInteger checks if the message is an integer
	* @return bool - true if the message is an integer
*/
func (r *RESPMessage) IsInteger() bool {
	return r.RESPType == Integer
}

/*
 	* IsError checks if the message is an error
	* @return bool - true if the message is an error
*/
func (r *RESPMessage) IsError() bool {
	return r.RESPType == Error
}

/*
 	* IsSimpleString checks if the message is a simple string
	* @return bool - true if the message is a simple string
*/
func (r *RESPMessage) IsSimpleString() bool {
	return r.RESPType == SimpleString
}

/*
 	* readLine reads data until CRLF is encountered
	* @return line []byte - the line read
	* @return length int - the length of the line
	* @return err error - the error if there is one
*/
func (r *Reader) readLine() (line []byte, length int, err error) {
	for {
		b, err := r.reader.ReadByte()
		if err != nil {
			return nil, 0, err
		}
		length += 1
		line = append(line, b)

		// break the loop
		if len(line) >= 2 && line[len(line)-2] == '\r' {
			break
		}
	}
	return line[:len(line)-2], length, nil
}

// // checks for presence of CRLF and it's correct order
// // also moves the reader ahead as reader.ReadByte() moves it ahead by one.
// func (r *Reader) readCRLF() error {
// 	cr, err := r.reader.ReadByte()
// 	if err != nil {
// 		return fmt.Errorf("failed to read \\r: %v", err)
// 	}
// 	if cr != '\r' {
// 		return fmt.Errorf("expected \\r, got %v", cr)
// 	}

// 	lf, err := r.reader.ReadByte()
// 	if err != nil {
// 		return fmt.Errorf("failed to read \\n: %v", err)
// 	}
// 	if lf != '\n' {
// 		return fmt.Errorf("expected \\n, got %v", lf)
// 	}

// 	return nil
// }

/*
 	* readLength reads the length specified for the incoming type of data
	* @return l int - the length of the data
	* @return err error - the error if there is one
*/
func (r *Reader) readLength() (l int, err error) {
	lengthLine, _, err := r.readLine()
	if err != nil {
		return 0, err
	}

	length, err := strconv.Atoi(string(lengthLine))
	if err != nil {
		return 0, err
	}

	return length, nil
}

/*
 	* encodeNil encodes a nil value
	* @return error - the error if there is one
*/
func (w *Writer) EncodeNil() error {
	if _, err := w.writer.Write([]byte("$-1\r\n")); err != nil {
		return err
	}
	return w.writer.Flush()
}
