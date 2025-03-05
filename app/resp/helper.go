package RESP

import (
	"fmt"
	"strconv"
)

/*
 	* asString converts to string
	* @return string - the string
	* @return error - the error if there is one
*/
func (msg *RESPMessage) asString() (string, error) {
	if msg.Value == nil {
		return "", fmt.Errorf("cannot parse nil value to string")
	}
	return string(msg.Value), nil
}

/*
* asInteger converts to integer
	* @return int64 - the integer
	* @return error - the error if there is one
*/
func (msg *RESPMessage) asInteger() (int64, error) {
	if msg.Value == nil {
		return 0, fmt.Errorf("cannot parse nil value to integer")
	}
	return strconv.ParseInt(string(msg.Value), 10, 64)
}

/*
 	* asError converts to an error
	* @return error - the error if there is one
*/
func (msg *RESPMessage) asError() error {
	if msg.Value == nil {
		return fmt.Errorf("%s", "error!!")
	}
	return fmt.Errorf("%s", string(msg.Value))
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
 	* convertArrayToBytesArray converts an array to an array of bytes
	* @param elements []RESPMessage - the array to convert
	* @return []byte - the array of bytes
	* @return error - the error if there is one
*/
func (r *Reader) convertArrayToBytesArray(elements []RESPMessage) ([]byte, error) {
	var result []byte
	for _, elem := range elements {
		result = append(result, elem.Value...)
	}
	return result, nil
}
