package RESP

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
)

const (
	Integer      = ':'
	SimpleString = '+'
	BulkString   = '$'
	Array        = '*'
	Error        = '-'
)

type RESPMessage struct {
	Type      byte
	Len       int
	Value     []byte
	ArrayElem []RESPMessage // storing for array type separately as it prevents double decoding of array during encoding.
}

type Reader struct {
	reader *bufio.Reader // helps track the state of reader.
}

func NewReader(rd io.Reader) *Reader {
	return &Reader{reader: bufio.NewReader(rd)}
}

func (r *Reader) Decode() (*RESPMessage, error) {
	_type, err := r.reader.ReadByte() // the first byte always represents the type of data coming in.
	if err != nil {
		return nil, err
	}

	switch _type {

	case SimpleString:
		return r.decodeSimpleString()

	case Error:
		return r.decodeError()

	case Integer:
		return r.decodeInteger()

	case BulkString:
		return r.decodeBulkString()

	case Array:
		return r.decodeArray()

	default:
		return nil, fmt.Errorf("unknown RESP type: %v", _type)
	}
}

func (r *Reader) decodeSimpleString() (*RESPMessage, error) {
	line, length, err := r.readLine()
	if err != nil {
		return nil, err
	}

	return &RESPMessage{Type: SimpleString, Len: length, Value: line}, nil
}

func (r *Reader) decodeError() (*RESPMessage, error) {
	line, length, err := r.readLine()
	if err != nil {
		return nil, err
	}

	return &RESPMessage{Type: Error, Len: length, Value: line}, nil
}

func (r *Reader) decodeInteger() (*RESPMessage, error) {
	line, length, err := r.readLine()
	if err != nil {
		return nil, err
	}

	return &RESPMessage{Type: Integer, Len: length, Value: line}, nil
}

func (r *Reader) decodeBulkString() (*RESPMessage, error) {
	length, err := r.readLength()
	if err != nil {
		return nil, fmt.Errorf("invalid bulk string length: %v", err)
	}

	// Redis limits to 512 MB
	if length > 512*1024*1024 {
		return nil, fmt.Errorf("bulk string length exceeds limit")
	}

	if length <= 0 {
		return &RESPMessage{Type: BulkString, Len: 0, Value: nil}, nil
	}

	content := make([]byte, length)
	if _, err := io.ReadFull(r.reader, content); err != nil {
		return nil, err
	}

	r.readLine() // Consume trailing \r\n

	return &RESPMessage{Type: BulkString, Len: length, Value: content}, nil
}

func (r *Reader) decodeArray() (*RESPMessage, error) {
	length, err := r.readLength()
	if err != nil {
		return nil, fmt.Errorf("invalid array length: %v", err)
	}

	// limit to 1MB
	if length > 1024*1024 {
		return nil, fmt.Errorf("array length exceeds limit")
	}

	arrayElements := make([]RESPMessage, 0, length)
	for i := 0; i < length; i++ {
		// re-read the inner types
		element, err := r.Decode()
		if err != nil {
			return nil, fmt.Errorf("error decoding array element %d: %v", i, err)
		}
		arrayElements = append(arrayElements, *element)
	}

	return &RESPMessage{Type: Array, Len: length, ArrayElem: arrayElements}, nil
}

type Writer struct {
	writer *bufio.Writer
}

func NewWriter(w io.Writer) *Writer {
	return &Writer{writer: bufio.NewWriter(w)}
}

func (w *Writer) Encode(msg *RESPMessage) error {

	switch msg.Type {

	case SimpleString:
		return w.encodeSimpleString(msg)

	case Error:
		return w.encodeError(msg)

	case Integer:
		return w.encodeInteger(msg)

	case BulkString:
		return w.encodeBulkString(msg)

	case Array:
		return w.encodeArray(msg)

	default:
		return fmt.Errorf("unsupported RESP type for encoding: %c", msg.Type)
	}
}

func (w *Writer) encodeSimpleString(msg *RESPMessage) error {
	if err := w.writer.WriteByte(SimpleString); err != nil {
		return err
	}

	if _, err := w.writer.Write(msg.Value); err != nil {
		return err
	}

	if _, err := w.writer.Write([]byte("\r\n")); err != nil {
		return err
	}

	return w.writer.Flush()
}

func (w *Writer) encodeError(msg *RESPMessage) error {
	if err := w.writer.WriteByte(Error); err != nil {
		return err
	}

	if _, err := w.writer.Write(msg.Value); err != nil {
		return err
	}

	if _, err := w.writer.Write([]byte("\r\n")); err != nil {
		return err
	}

	return w.writer.Flush()
}

func (w *Writer) encodeInteger(msg *RESPMessage) error {
	if err := w.writer.WriteByte(Integer); err != nil {
		return err
	}

	if _, err := w.writer.Write(msg.Value); err != nil {
		return err
	}

	if _, err := w.writer.Write([]byte("\r\n")); err != nil {
		return err
	}

	return w.writer.Flush()
}

func (w *Writer) encodeBulkString(msg *RESPMessage) error {
	if msg.Value == nil {
		return w.encodeNil()
	}

	if err := w.writer.WriteByte(BulkString); err != nil {
		return err
	}

	if _, err := w.writer.Write([]byte(strconv.Itoa(msg.Len))); err != nil {
		return err
	}

	if _, err := w.writer.Write([]byte("\r\n")); err != nil {
		return err
	}

	if _, err := w.writer.Write(msg.Value); err != nil {
		return err
	}

	if _, err := w.writer.Write([]byte("\r\n")); err != nil {
		return err
	}

	return w.writer.Flush()
}

func (w *Writer) encodeArray(msg *RESPMessage) error {
	if err := w.writer.WriteByte(Array); err != nil {
		return err
	}

	if _, err := w.writer.Write([]byte(strconv.Itoa(msg.Len))); err != nil {
		return err
	}

	if _, err := w.writer.Write([]byte("\r\n")); err != nil {
		return err
	}

	for _, element := range msg.ArrayElem {
		// this is why a separate storage was done for array elements
		// no extra decoding for array elements
		if err := w.Encode(&element); err != nil {
			return fmt.Errorf("error encoding array element: %v", err)
		}
	}

	return w.writer.Flush()
}

func (w *Writer) encodeNil() error {
	if _, err := w.writer.Write([]byte("$-1\r\n")); err != nil {
		return err
	}
	return w.writer.Flush()
}
