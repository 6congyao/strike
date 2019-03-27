//Created by zhbinary on 2018/12/14.
//Email: zhbinary@gmail.com
package message

import (
	"errors"
	"strconv"
	"strike/pkg/buffer"
	"strike/pkg/protocol"
)

type Publish struct {
	Header
	Dup    bool
	Qos    Qos
	Retain bool

	// Variable header
	TopicName        string
	PacketIdentifier uint16

	// Payload
	Payload []byte
}

func NewPublish() *Publish {
	return &Publish{Header: Header{msgType: MsgTypePublish}}
}

func (this *Publish) DecodeFixedHeader(buf buffer.IoBuffer) bool {
	this.Dup = this.msgFlag&0x08 > 0
	this.Qos = Qos(this.msgFlag & 0x06 >> 1)
	this.Retain = this.msgFlag&0x01 > 0
	return false
}

func (this *Publish) DecodeVariableHeader(buf buffer.IoBuffer) bool {
	before := buf.Len()
	this.TopicName = getString(buf)
	if this.Qos.HasId() {
		this.PacketIdentifier = getUint16(buf)
	}
	after := buf.Len()
	this.Header.remainingLength -= RemainingLength(before - after)
	if this.remainingLength == 0 {
		return true
	}
	return false
}

func (this *Publish) DecodePayload(buf buffer.IoBuffer) bool {
	before := buf.Len()
	if int(this.remainingLength) > buf.Len() {
		return false
	}
	this.Payload = make([]byte, this.Header.remainingLength)
	buf.Read(this.Payload)
	after := buf.Len()
	this.Header.remainingLength -= RemainingLength(before - after)
	return true
}

func (this *Publish) Encode() (buffer.IoBuffer, error) {
	buf := buffer.NewIoBuffer(0)
	putString(this.TopicName, buf)
	if this.Qos.HasId() {
		putUint16(this.PacketIdentifier, buf)
	}
	if this.Payload != nil {
		buf.Write(this.Payload)
	}

	this.Header.msgFlag = boolToByte(this.Dup)<<3 | byte(this.Qos)<<1 | boolToByte(this.Retain)
	this.Header.remainingLength = RemainingLength(buf.Len())
	bufAll := this.Header.encode()

	_, err := bufAll.Write(buf.Bytes())
	if err != nil {
		return nil, errors.New(ErrorInvalidMessage)
	}

	return bufAll, nil
}

func (this *Publish) GetHeader() (header map[string]string) {
	header = make(map[string]string, 6)
	header[protocol.StrikeHeaderMethod] = StrMsgTypePublish
	header[protocol.StrikeHeaderPacketID] = strconv.Itoa(int(this.PacketIdentifier))
	header[protocol.StrikeHeaderTopicName] = this.TopicName
	header[protocol.StrikeHeaderMessageDup] = strconv.FormatBool(this.Dup)
	header[protocol.StrikeHeaderMessageQos] = strconv.Itoa(int(this.Qos))
	header[protocol.StrikeHeaderMessageRetain] = strconv.FormatBool(this.Retain)
	return header
}

func (this *Publish) GetPayload() (buf buffer.IoBuffer) {
	buf = buffer.NewIoBuffer(0)
	_, err := buf.Write(this.Payload)
	if err != nil {
		return nil
	}
	return buf
}
