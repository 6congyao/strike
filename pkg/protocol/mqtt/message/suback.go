//Created by zhbinary on 2018/12/14.
//Email: zhbinary@gmail.com
package message

import (
	"errors"
	"strike/pkg/buffer"
	"strike/pkg/protocol"
)

type SubAck struct {
	Header

	// Variable header
	PacketIdentifier uint16

	// Payload
	acks []SubAckCode
}

type SubAckCode uint8

const (
	SubAckCodeQos0 = SubAckCode(iota)
	SubAckCodeQos1
	SubAckCodeQos2
	SubAckCodeFailure
)

func NewSubAck() *SubAck {
	return &SubAck{Header: Header{msgType: MsgTypeSubAck}}
}

func (this *SubAck) DecodeFixedHeader(buf buffer.IoBuffer) bool {
	return false
}

func (this *SubAck) DecodeVariableHeader(buf buffer.IoBuffer) bool {
	before := buf.Len()
	this.PacketIdentifier = getUint16(buf)
	after := buf.Len()
	this.remainingLength -= RemainingLength(before - after)
	if this.remainingLength == 0 {
		return true
	}
	return false
}

func (this *SubAck) DecodePayload(buf buffer.IoBuffer) bool {
	if int(this.remainingLength) > buf.Len() {
		return false
	}
	var acks []SubAckCode
	if this.remainingLength > 0 {
		acks = append(acks, SubAckCode(getUint8(buf)))
	}
	this.acks = acks
	return true
}

func (this *SubAck) Encode() (buffer.IoBuffer, error) {
	buf := buffer.NewIoBuffer(0)
	putUint16(this.PacketIdentifier, buf)

	for _, ack := range this.acks {
		buf.WriteByte(byte(ack))
	}

	this.remainingLength = RemainingLength(buf.Len())
	bufAll := this.Header.encode()

	_, err := bufAll.Write(buf.Bytes())
	if err != nil {
		return nil, errors.New(ErrorInvalidMessage)
	}

	return bufAll, nil
}
func (this *SubAck) GetHeader() (header map[string]string) {
	header = make(map[string]string, 1)
	header[protocol.StrikeHeaderMethod] = StrMsgTypeSubAck
	return header
}

func (this *SubAck) GetPayload() (buf buffer.IoBuffer) {
	return nil
}
