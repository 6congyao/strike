//Created by zhbinary on 2018/12/14.
//Email: zhbinary@gmail.com
package message

import (
	"errors"
	"strconv"
	"strike/pkg/buffer"
	"strike/pkg/protocol"
)

type UnsubAck struct {
	Header

	// Variable header
	PacketIdentifier uint16
}

func NewUnsubAck() *UnsubAck {
	return &UnsubAck{Header: Header{msgType: MsgTypeUnsubAck}}
}

func (this *UnsubAck) DecodeFixedHeader(buf buffer.IoBuffer) bool {
	return false
}

func (this *UnsubAck) DecodeVariableHeader(buf buffer.IoBuffer) bool {
	before := buf.Len()
	this.PacketIdentifier = getUint16(buf)
	after := buf.Len()
	this.remainingLength -= RemainingLength(before - after)
	return true
}

func (this *UnsubAck) DecodePayload(buf buffer.IoBuffer) bool {
	panic("implement me")
}

func (this *UnsubAck) Encode() (buffer.IoBuffer, error) {
	buf := buffer.NewIoBuffer(0)
	putUint16(this.PacketIdentifier, buf)

	this.remainingLength = RemainingLength(buf.Len())
	bufAll := this.Header.encode()

	_, err := bufAll.Write(buf.Bytes())
	if err != nil {
		return nil, errors.New(ErrorInvalidMessage)
	}

	return bufAll, nil
}

func (this *UnsubAck) GetHeader() (header map[string]string) {
	header = make(map[string]string, 2)
	header[protocol.StrikeHeaderMethod] = StrMsgTypeUnsubAck
	header[protocol.StrikeHeaderPacketID] = strconv.Itoa(int(this.PacketIdentifier))
	return header
}

func (this *UnsubAck) GetPayload() (buf buffer.IoBuffer) {
	return nil
}
