package codec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"

	"github.com/go-distributed/messenger/3rdparty/code.google.com/p/gogoprotobuf/proto"
	log "github.com/go-distributed/messenger/3rdparty/github.com/golang/glog"
)

// Only support 256 different kinds of messages for now.
type messageType uint8

// GoGoProtobufCodec implements the codec interface for Codec.
// We use reflect to make it a 'self-explained' codec.
type GoGoProtobufCodec struct {
	registeredMessages    map[reflect.Type]messageType
	reversedMap           map[messageType]reflect.Type
	registeredMessagePtrs map[reflect.Type]messageType
}

// NewGoGoProtobufCodec creates a new gogpprotobuf codec.
func NewGoGoProtobufCodec() *GoGoProtobufCodec {
	return &GoGoProtobufCodec{
		registeredMessages:    make(map[reflect.Type]messageType),
		reversedMap:           make(map[messageType]reflect.Type),
		registeredMessagePtrs: make(map[reflect.Type]messageType),
	}
}

// Initial the gogoprotobuf codec (no-op for now).
func (c *GoGoProtobufCodec) Initial() error {
	return nil
}

// Destroy the gogoprotobuf codec (no-op for now).
func (c *GoGoProtobufCodec) Destroy() error {
	return nil
}

// RegisterMessage regists a message type.
func (c *GoGoProtobufCodec) RegisterMessage(msg interface{}) error {
	var concreteType reflect.Type
	var ptrType reflect.Type

	msgTypeValue := reflect.ValueOf(msg)

	if msgTypeValue.Kind() == reflect.Ptr {
		concreteType = reflect.Indirect(msgTypeValue).Type()
		ptrType = msgTypeValue.Type()
	} else {
		concreteType = msgTypeValue.Type()
		ptrType = reflect.PtrTo(concreteType)
	}
	if _, ok := c.registeredMessages[concreteType]; ok {
		return fmt.Errorf("Message type %v is already registered", concreteType)
	}
	if _, ok := msg.(proto.Message); !ok {
		return fmt.Errorf("Not a protobuf message %v", concreteType)
	}
	// Store the message type.
	mtype := messageType(len(c.registeredMessages))
	c.registeredMessages[concreteType] = mtype
	c.reversedMap[mtype] = concreteType
	c.registeredMessagePtrs[ptrType] = mtype
	return nil
}

// Marshal a message into a byte slice.
// The msg must be a pointer type.
func (c *GoGoProtobufCodec) Marshal(msg interface{}) ([]byte, error) {
	var err error

	defer func() {
		if err != nil {
			log.Warningf("GoGoProtobufCodec: Failed to marshal: %v\n", err)
		}
	}()

	// Check if the message is registered.
	mtype, ok := c.registeredMessagePtrs[reflect.TypeOf(msg)]
	if !ok {
		return nil, fmt.Errorf("Unknown message type: %v", reflect.ValueOf(msg).Elem().Type())
	}

	b, err := proto.Marshal(msg.(proto.Message))
	if err != nil {
		return nil, err
	}

	// Write the type at the beginning.
	buf := new(bytes.Buffer)
	if err = binary.Write(buf, binary.LittleEndian, mtype); err != nil {
		return nil, err
	}

	// Write the data.
	n, err := buf.Write(b)
	if err != nil || n != len(b) {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal a message from a byte slice.
func (c *GoGoProtobufCodec) Unmarshal(data []byte) (interface{}, error) {
	var err error
	var mtype messageType

	defer func() {
		if err != nil {
			log.Warningf("GoGoProtobufCodec: Failed to unmarshal: %v\n", err)
		}
	}()

	buf := bytes.NewBuffer(data)
	if err = binary.Read(buf, binary.LittleEndian, &mtype); err != nil {
		return nil, err
	}

	msg := reflect.New(c.reversedMap[mtype]).Interface().(proto.Message)
	if err = proto.Unmarshal(buf.Bytes(), msg); err != nil {
		return nil, err
	}
	return msg, nil
}
