package message

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/aliyun/aliyun_assist_client/agent/log"
)

const (
	InputStreamDataMessage  = 0 // string = "input_stream_data"
	OutputStreamDataMessage = 1 // string = "output_stream_data"
	SetSizeDataMessage      = 2 //string = "set_size"
	CloseDataChannel        = 3
	StatusDataMessage       = 5
)

const (
	AgentMessage_MessageTypeLength    = 4
	AgentMessage_SchemaVersionLength  = 4
	AgentMessage_SessionIdLength      = 32
	AgentMessage_CreatedDateLength    = 8
	AgentMessage_SequenceNumberLength = 8
	AgentMessage_PayloadLength        = 4
)

const (
	AgentMessage_MessageTypeOffset    = 0
	AgentMessage_SchemaVersionOffset  = AgentMessage_MessageTypeOffset + AgentMessage_MessageTypeLength
	AgentMessage_SessionIdOffset      = AgentMessage_SchemaVersionOffset + AgentMessage_SchemaVersionLength
	AgentMessage_CreatedDateOffset    = AgentMessage_SessionIdOffset + AgentMessage_SessionIdLength
	AgentMessage_SequenceNumberOffset = AgentMessage_CreatedDateOffset + AgentMessage_CreatedDateLength
	AgentMessage_PayloadLengthOffset  = AgentMessage_SequenceNumberOffset + AgentMessage_SequenceNumberLength
	AgentMessage_PayloadOffset        = AgentMessage_PayloadLengthOffset + AgentMessage_PayloadLength
)

type Message struct {
	MessageType    uint32
	SchemaVersion  string
	SessionId      string
	CreatedDate    uint64
	SequenceNumber int64
	// MessageId      string
	PayloadLength uint32
	Payload       []byte
}

func (message *Message) Deserialize(input []byte) (err error) {
	message.MessageType, err = getUInteger(input, AgentMessage_MessageTypeOffset)
	if err != nil {
		log.GetLogger().Errorf("Could not deserialize field MessageType with error: %v", err)
		return err
	}
	message.SchemaVersion, err = getString(input, AgentMessage_SchemaVersionOffset, AgentMessage_SchemaVersionLength)
	if err != nil {
		log.GetLogger().Errorf("Could not deserialize field SchemaVersion with error: %v", err)
		return err
	}

	message.SessionId, err = getString(input, AgentMessage_SessionIdOffset, AgentMessage_SessionIdLength)
	if err != nil {
		log.GetLogger().Errorf("Could not deserialize field SessonId with error: %v", err)
		return err
	}
	message.CreatedDate, err = getULong(input, AgentMessage_CreatedDateOffset)
	if err != nil {
		log.GetLogger().Errorf("Could not deserialize field CreatedDate with error: %v", err)
		return err
	}

	message.SequenceNumber, err = getLong(input, AgentMessage_SequenceNumberOffset)
	if err != nil {
		log.GetLogger().Errorf("Could not deserialize field SequenceNumber with error: %v", err)
		return err
	}

	message.PayloadLength, err = getUInteger(input, AgentMessage_PayloadLengthOffset)
	message.Payload = input[AgentMessage_PayloadOffset:]

	return nil
}

func getUInteger(byteArray []byte, offset int) (result uint32, err error) {
	var temp int32
	temp, err = getInteger(byteArray, offset)
	return uint32(temp), err
}

// getULong gets an unsigned long integer
func getULong(byteArray []byte, offset int) (result uint64, err error) {
	var temp int64
	temp, err = getLong(byteArray, offset)
	return uint64(temp), err
}

func getLong(byteArray []byte, offset int) (result int64, err error) {
	byteArrayLength := len(byteArray)
	if offset > byteArrayLength-1 || offset+8 > byteArrayLength-1 || offset < 0 {
		log.GetLogger().Error("getLong failed: Offset is invalid.")
		return 0, errors.New("Offset is outside the byte array.")
	}
	return bytesToLong(byteArray[offset : offset+8])
}

func getInteger(byteArray []byte, offset int) (result int32, err error) {
	byteArrayLength := len(byteArray)
	if offset > byteArrayLength-1 || offset+4 > byteArrayLength-1 || offset < 0 {
		log.GetLogger().Error("getInteger failed: Offset is invalid.")
		return 0, errors.New("Offset is bigger than the byte array.")
	}
	return bytesToInteger(byteArray[offset : offset+4])
}

func getString(byteArray []byte, offset int, stringLength int) (result string, err error) {
	byteArrayLength := len(byteArray)
	if offset > byteArrayLength-1 || offset+stringLength-1 > byteArrayLength-1 || offset < 0 {
		log.GetLogger().Error("getString failed: Offset is invalid.")
		return "", errors.New("Offset is outside the byte array.")
	}

	//remove nulls from the bytes array
	b := bytes.Trim(byteArray[offset:offset+stringLength], "\x00")

	return strings.TrimSpace(string(b)), nil
}

// Validate returns error if the message is invalid
func (message *Message) Validate() error {
	if message.CreatedDate == 0 {
		return errors.New("CreatedDate is missing")
	}
	return nil
}

func (message *Message) Serialize() (result []byte, err error) {
	payloadLength := uint32(len(message.Payload))
	headerLength := uint32(AgentMessage_PayloadLengthOffset)
	// If the payloadinfo length is incorrect, fix it.
	if payloadLength != message.PayloadLength {
		log.GetLogger().Debugf("Payload length will be adjusted: %v", message.PayloadLength)
		message.PayloadLength = payloadLength
	}

	totalMessageLength := headerLength + 4 + payloadLength
	result = make([]byte, totalMessageLength)

	if err = putUInteger(result, AgentMessage_MessageTypeOffset, message.MessageType); err != nil {
		log.GetLogger().Errorf("Could not serialize MessageType with error: %v", err)
		return make([]byte, 1), err
	}

	startPosition := AgentMessage_SchemaVersionOffset
	endPosition := AgentMessage_SchemaVersionOffset + AgentMessage_SchemaVersionLength - 1
	if err = putString(result, startPosition, endPosition, message.SchemaVersion); err != nil {
		log.GetLogger().Errorf("Could not serialize version with error: %v", err)
		return make([]byte, 1), err
	}

	startPosition = AgentMessage_SessionIdOffset
	endPosition = AgentMessage_SessionIdOffset + AgentMessage_SessionIdLength - 1
	if err = putString(result, startPosition, endPosition, message.SessionId); err != nil {
		log.GetLogger().Errorf("Could not serialize SessionId with error: %v", err)
		return make([]byte, 1), err
	}

	if err = putULong(result, AgentMessage_CreatedDateOffset, message.CreatedDate); err != nil {
		log.GetLogger().Errorf("Could not serialize CreatedDate with error: %v", err)
		return make([]byte, 1), err
	}

	if err = putLong(result, AgentMessage_SequenceNumberOffset, message.SequenceNumber); err != nil {
		log.GetLogger().Errorf("Could not serialize SequenceNumber with error: %v", err)
		return make([]byte, 1), err
	}

	if err = putUInteger(result, AgentMessage_PayloadLengthOffset, message.PayloadLength); err != nil {
		log.GetLogger().Errorf("Could not serialize PayloadLength with error: %v", err)
		return make([]byte, 1), err
	}

	startPosition = AgentMessage_PayloadOffset
	endPosition = AgentMessage_PayloadOffset + int(payloadLength) - 1
	if err = putBytes(result, startPosition, endPosition, message.Payload); err != nil {
		log.GetLogger().Errorf("Could not serialize Payload with error: %v", err)
		return make([]byte, 1), err
	}

	return result, nil
}

func putBytes(byteArray []byte, offsetStart int, offsetEnd int, inputBytes []byte) (err error) {
	byteArrayLength := len(byteArray)
	if offsetStart > byteArrayLength-1 || offsetEnd > byteArrayLength-1 || offsetStart > offsetEnd || offsetStart < 0 {
		log.GetLogger().Error("putBytes failed: Offset is invalid.")
		return errors.New("Offset is outside the byte array.")
	}

	if offsetEnd-offsetStart+1 != len(inputBytes) {
		log.GetLogger().Error("putBytes failed: Not enough space to save the bytes.")
		return errors.New("Not enough space to save the bytes.")
	}

	copy(byteArray[offsetStart:offsetEnd+1], inputBytes)
	return nil
}

// putString puts a string value to a byte array starting from the specified offset.
func putString(byteArray []byte, offsetStart int, offsetEnd int, inputString string) (err error) {
	byteArrayLength := len(byteArray)
	if offsetStart > byteArrayLength-1 || offsetEnd > byteArrayLength-1 || offsetStart > offsetEnd || offsetStart < 0 {
		log.GetLogger().Error("putString failed: Offset is invalid.")
		return errors.New("Offset is outside the byte array.")
	}

	if offsetEnd-offsetStart+1 < len(inputString) {
		log.GetLogger().Error("putString failed: Not enough space to save the string.")
		return errors.New("Not enough space to save the string.")
	}

	// wipe out the array location first and then insert the new value.
	for i := offsetStart; i <= offsetEnd; i++ {
		byteArray[i] = ' '
	}

	copy(byteArray[offsetStart:offsetEnd+1], inputString)
	return nil
}

// putUInteger puts an unsigned integer
func putUInteger(byteArray []byte, offset int, value uint32) (err error) {
	return putInteger(byteArray, offset, int32(value))
}

// putULong puts an unsigned long integer.
func putULong(byteArray []byte, offset int, value uint64) (err error) {
	return putLong(byteArray, offset, int64(value))
}

func putInteger(byteArray []byte, offset int, value int32) (err error) {
	byteArrayLength := len(byteArray)
	if offset > byteArrayLength-1 || offset+4 > byteArrayLength-1 || offset < 0 {
		log.GetLogger().Error("putInteger failed: Offset is invalid.")
		return errors.New("Offset is outside the byte array.")
	}

	bytes, err := integerToBytes(value)
	if err != nil {
		log.GetLogger().Error("putInteger failed: getBytesFromInteger Failed.")
		return err
	}

	copy(byteArray[offset:offset+4], bytes)
	return nil
}

// putLong puts a long integer value to a byte array starting from the specified offset.
func putLong(byteArray []byte, offset int, value int64) (err error) {
	byteArrayLength := len(byteArray)
	if offset > byteArrayLength-1 || offset+8 > byteArrayLength-1 || offset < 0 {
		log.GetLogger().Error("putLong failed: Offset is invalid.")
		return errors.New("Offset is outside the byte array.")
	}

	mbytes, err := longToBytes(value)
	if err != nil {
		log.GetLogger().Error("putLong failed: getBytesFromInteger Failed.")
		return err
	}

	copy(byteArray[offset:offset+8], mbytes)
	return nil
}

// bytesToLong gets a Long integer from a byte array.
func bytesToLong(input []byte) (result int64, err error) {
	var res int64
	inputLength := len(input)
	if inputLength != 8 {
		log.GetLogger().Error("bytesToLong failed: input array size is not equal to 8.")
		return 0, errors.New("Input array size is not equal to 8.")
	}
	buf := bytes.NewBuffer(input)
	binary.Read(buf, binary.LittleEndian, &res)
	return res, nil
}

// longToBytes gets bytes array from a long integer.
func longToBytes(input int64) (result []byte, err error) {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, input)
	if buf.Len() != 8 {
		log.GetLogger().Error("longToBytes failed: buffer output length is not equal to 8.")
		return make([]byte, 8), errors.New("Input array size is not equal to 8.")
	}

	return buf.Bytes(), nil
}

// integerToBytes gets bytes array from an integer.
func integerToBytes(input int32) (result []byte, err error) {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, input)
	if buf.Len() != 4 {
		log.GetLogger().Error("integerToBytes failed: buffer output length is not equal to 4.")
		return make([]byte, 4), errors.New("Input array size is not equal to 4.")
	}

	return buf.Bytes(), nil
}

// bytesToInteger gets an integer from a byte array.
func bytesToInteger(input []byte) (result int32, err error) {
	var res int32
	inputLength := len(input)
	if inputLength != 4 {
		log.GetLogger().Error("bytesToInteger failed: input array size is not equal to 4.")
		return 0, errors.New("Input array size is not equal to 4.")
	}
	buf := bytes.NewBuffer(input)
	binary.Read(buf, binary.LittleEndian, &res)
	return res, nil
}

func BytesToIntU(b []byte) (int, error) {
	if len(b) == 3 {
		b = append([]byte{0}, b...)
	}
	bytesBuffer := bytes.NewBuffer(b)
	switch len(b) {
	case 1:
		var tmp uint8
		err := binary.Read(bytesBuffer, binary.BigEndian, &tmp)
		return int(tmp), err
	case 2:
		var tmp uint16
		err := binary.Read(bytesBuffer, binary.BigEndian, &tmp)
		return int(tmp), err
	case 4:
		var tmp uint32
		err := binary.Read(bytesBuffer, binary.BigEndian, &tmp)
		return int(tmp), err
	default:
		return 0, fmt.Errorf("%s", "BytesToInt bytes lenth is invalid!")
	}
}
