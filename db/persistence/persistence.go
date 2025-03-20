package persistence

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"os"
	"time"
)

var ErrInvalidHeader = errors.New("invalid RDB header")
var ErrInvalidMetadata = errors.New("invalid RDB metadata")
var ErrInvalidDatabase = errors.New("invalid RDB database")

const (
	RDB_METADATA   = 0xFA // Metadata
	RDB_DB_START   = 0xFE // Database selector
	RDB_DB_SIZE    = 0xFB // Hash table sizes
	RDB_STRING     = 0x00
	RDB_EXPIRES_MS = 0xFC // Expire time MS
	RDB_EXPIRES_S  = 0xFD // Expire time S
	RDB_EOF        = 0xFF // End of file
	RDB_MODULE_AUX = 0xF7 // Module auxiliary data
)

type rdbParser struct {
	rdbVersion      string
	metadata        map[string]string
	databaseIndex   uint64
	tableSize       uint64
	expireTableSize uint64
}
type ParsedKeyValue struct {
	Key       string
	Value     []byte
	ExpiresIn time.Duration
}

func newRDBParser() *rdbParser {
	return &rdbParser{
		metadata: make(map[string]string),
	}
}

var RDBInstance *rdbParser

/*
 	* GetRDBInstance returns the singleton instance of the rdbParser.
	* @return *rdbParser - the singleton instance of the rdbParser
*/
func GetRDBInstance() *rdbParser {
	if RDBInstance == nil {
		RDBInstance = newRDBParser()
	}
	return RDBInstance
}

/*
 	* Parse parses the RDB file at the given path.
	* @param path string - the path to the RDB file
	* @return []ParsedKeyValue - the parsed data
	* @return error - the error if there is one
*/
func (p *rdbParser) Parse(path string) ([]ParsedKeyValue, error) {
	var parsedData []ParsedKeyValue

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := bufio.NewReader(f)

	if err := p.parseHeader(r); err != nil {
		log.Println("Error parsing header", err)
		return nil, ErrInvalidHeader
	}

	if err := p.parseMetadata(r); err != nil {
		log.Println("Error parsing metadata", err)
		return nil, ErrInvalidMetadata
	}

	parsedData, err = p.parseDatabase(r)
	if err != nil {
		log.Println("Error parsing database", err)
		return nil, ErrInvalidDatabase
	}

	return parsedData, nil
}

/*
 	* parseDatabase parses the database section of the RDB file.
	* @param r *bufio.Reader - the reader to read the RDB file from
	* @return []ParsedKeyValue - the parsed data
	* @return error - the error if there is one
*/
func (p *rdbParser) parseDatabase(r *bufio.Reader) ([]ParsedKeyValue, error) {
	var parsedData []ParsedKeyValue

	// if there is anything between metadata and the database, skip it for now
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, err
		}
		if b == RDB_DB_START {
			break
		}
	}

	// read the database index
	l, _, err := p.readLength(r)
	if err != nil {
		return nil, err
	}
	p.databaseIndex = l

	// verify the database size delimiter
	b, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("expected DB size delimiter, got 0x%02X", b)
	}
	if b != RDB_DB_SIZE {
		return nil, fmt.Errorf("expected DB size delimiter, got 0x%02X", b)
	}

	// read the database size
	l, _, err = p.readLength(r)
	if err != nil {
		return nil, err
	}
	p.tableSize = l
	l, _, err = p.readLength(r)
	if err != nil {
		return nil, err
	}
	p.expireTableSize = l

	// read the key-value pairs
	for {
		b, err := r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("error reading key-value pair type: %w", err)
		}

		switch b {
		case RDB_STRING:
			log.Println("Adding new key-value pair")
			kv, err := p.addKeyValue(r)
			if err != nil {
				return nil, err
			}
			parsedData = append(parsedData, kv)
		case RDB_EXPIRES_MS, RDB_EXPIRES_S:
			log.Println("Adding new key-value pair with expire")
			kv, err := p.addKeyValueWithTTL(b, r)
			if err != nil {
				return nil, err
			}
			parsedData = append(parsedData, kv)
		case RDB_MODULE_AUX:
			// Skip the module auxiliary data
			log.Println("Skipping module auxiliary data")
			return parsedData, nil
		case RDB_EOF:
			log.Println("End of database section")
			return parsedData, nil
		default:
			return nil, fmt.Errorf("unknown key-value pair type: 0x%02X", b)
		}
	}
}

/*
 	* addKeyValue adds a key-value pair to the global key-value store.
	* @param r *bufio.Reader - the reader to read the RDB file from
	* @return ParsedKeyValue - the parsed key-value pair
	* @return error - the error if there is one
*/
func (p *rdbParser) addKeyValue(r *bufio.Reader) (ParsedKeyValue, error) {
	// key
	key, err := p.readNextString(r)
	if err != nil {
		return ParsedKeyValue{}, fmt.Errorf("error reading db key: %w", err)
	}

	// value
	value, err := p.readNextString(r)
	if err != nil {
		return ParsedKeyValue{}, fmt.Errorf("error reading db value: %w", err)
	}

	return ParsedKeyValue{
		Key:   key,
		Value: []byte(value),
	}, nil
}

/*
 	* addKeyValueWithTTL adds a key-value with expiration time.
	* @param kv_type byte - the type of the key-value pair
	* @param r *bufio.Reader - the reader to read the RDB file from
	* @return ParsedKeyValue - the parsed key-value pair
	* @return error - the error if there is one
*/
func (p *rdbParser) addKeyValueWithTTL(kv_type byte, r *bufio.Reader) (ParsedKeyValue, error) {
	var expireIn time.Duration
	switch kv_type {
	case RDB_EXPIRES_MS:
		bytes := make([]byte, 8)
		_, err := r.Read(bytes)
		if err != nil {
			return ParsedKeyValue{}, fmt.Errorf("error reading expire time MS: %w", err)
		}
		expireAt := time.UnixMilli(int64(binary.LittleEndian.Uint64(bytes)))
		expireIn = time.Until(expireAt)

	case RDB_EXPIRES_S:
		bytes := make([]byte, 4)
		_, err := r.Read(bytes)
		if err != nil {
			return ParsedKeyValue{}, fmt.Errorf("error reading expire time S: %w", err)
		}
		expireAt := time.Unix(int64(binary.LittleEndian.Uint32(bytes)), 0)
		expireIn = time.Until(expireAt)
	default:
		return ParsedKeyValue{}, fmt.Errorf("unknown expire time type: 0x%02X", kv_type)
	}

	// Verify value type
	b, err := r.ReadByte()
	if err != nil {
		return ParsedKeyValue{}, fmt.Errorf("error reading value type: %w", err)
	}
	if b != RDB_STRING {
		return ParsedKeyValue{}, fmt.Errorf("expected string value type, got 0x%02X", b)
	}

	// key
	key, err := p.readNextString(r)
	if err != nil {
		return ParsedKeyValue{}, fmt.Errorf("error reading db key: %w", err)
	}

	// value
	value, err := p.readNextString(r)
	if err != nil {
		return ParsedKeyValue{}, fmt.Errorf("error reading db value: %w", err)
	}

	if expireIn < 0 {
		log.Println("Key ", key, " expired")
		return ParsedKeyValue{}, nil
	}

	return ParsedKeyValue{
		Key:       key,
		Value:     []byte(value),
		ExpiresIn: expireIn,
	}, nil
}

/*
 	* parseHeader parses the header of the RDB file.
	* @param r *bufio.Reader - the reader to read the RDB file from
	* @return error - the error if there is one
*/
func (p *rdbParser) parseHeader(r *bufio.Reader) error {
	header := make([]byte, 9)
	_, err := r.Read(header)
	if err != nil {
		return err
	}

	if string(header)[:5] != "REDIS" {
		return ErrInvalidHeader
	}

	p.rdbVersion = string(header)[5:]

	return nil
}

/*
 	* parseMetadata parses the metadata section of the RDB file.
	* @param r *bufio.Reader - the reader to read the RDB file from
	* @return error - the error if there is one
*/
func (p *rdbParser) parseMetadata(r *bufio.Reader) error {
	for {
		// check if we are in the metadata section
		b, err := r.Peek(1)
		if err != nil {
			return err
		}

		if b[0] != RDB_METADATA {
			break
		}

		// ignore the metadata byte
		_, err = r.ReadByte()
		if err != nil {
			return err
		}

		key, err := p.readNextString(r)
		if err != nil {
			return err
		}

		value, err := p.readNextString(r)
		if err != nil {
			return err
		}

		p.metadata[key] = value
	}

	return nil
}

/*
 	* readNextString reads the next string from the reader.
	* @param r *bufio.Reader - the reader to read the RDB file from
	* @return string - the next string
	* @return error - the error if there is one
*/
// It's expected that the next byte of the reader is the length of the string.
func (p *rdbParser) readNextString(r *bufio.Reader) (string, error) {
	l, stringEncoded, err := p.readLength(r)
	if err != nil {
		return "", err
	}

	if stringEncoded {
		return p.readStringEncoded(r)
	}

	buf := make([]byte, l)
	_, err = r.Read(buf)
	if err != nil {
		return "", err
	}

	return string(buf), nil
}

/*
 	* readStringEncoded reads a string encoded in a special way.
	* @param r *bufio.Reader - the reader to read the RDB file from
	* @return string - the next string
	* @return error - the error if there is one
*/
func (p *rdbParser) readStringEncoded(r *bufio.Reader) (string, error) {
	b, err := r.ReadByte()
	if err != nil {
		return "", fmt.Errorf("error reading special encoding: %w", err)
	}

	switch b {
	case 0xC0:
		b, err := r.ReadByte()
		if err != nil {
			return "", fmt.Errorf("error reading special encoding 0xC0: %w", err)
		}
		return fmt.Sprintf("%d", int(b)), nil
	case 0xC1:
		b := make([]byte, 2)
		_, err := r.Read(b)
		if err != nil {
			return "", fmt.Errorf("error reading special encoding 0xC1: %w", err)
		}
		return fmt.Sprintf("%d", binary.LittleEndian.Uint16(b)), nil
	case 0xC2:
		b := make([]byte, 4)
		_, err := r.Read(b)
		if err != nil {
			return "", fmt.Errorf("error reading special encoding 0xC1: %w", err)
		}
		return fmt.Sprintf("%d", binary.LittleEndian.Uint16(b)), nil
	}

	return "", fmt.Errorf("unknown special encoding: 0x%02X", b)
}

/*
 	* readLength reads the length of the next string from the reader.
	* @param r *bufio.Reader - the reader to read the RDB file from
	* @return uint64 - the length of the next string
	* @return bool - true if the length is encoded, false otherwise
	* @return error - the error if there is one
*/
func (p *rdbParser) readLength(r *bufio.Reader) (uint64, bool, error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, false, fmt.Errorf("error reading length: %w", err)
	}

	lengthType := (b & 0xC0) >> 6

	switch lengthType {
	case 0x00: // 6 bits string length
		return uint64(b & 0x3F), false, nil
	case 0x01: // 14 bits string length
		b2, err := r.ReadByte()
		if err != nil {
			return 0, false, fmt.Errorf("error reading length in 14bit encoded: %w", err)
		}
		// TODO: Check if the length is correct
		return (uint64(b&0x3F) << 8) | uint64(b2), false, nil
	case 0x02: // 32 bits string length
		buf := make([]byte, 4)
		_, err := r.Read(buf)
		if err != nil {
			return 0, false, fmt.Errorf("error reading length in 32bit encoded: %w", err)
		}
		return uint64(binary.BigEndian.Uint32(buf)), false, nil
	case 0x03: // special encoding
		// unread the special byte, so we can read it again in the special encoding function
		err := r.UnreadByte()
		if err != nil {
			return 0, false, fmt.Errorf("error unread byte: %w", err)
		}
		return 0, true, nil
	}

	return 0, false, fmt.Errorf("unknown length type: 0x%02X", lengthType)
}
