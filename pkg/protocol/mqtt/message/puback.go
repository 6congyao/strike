//Created by zhbinary on 2018/12/14.
//Email: zhbinary@gmail.com
package message

import (
	"errors"
	"strike/pkg/buffer"
	"strike/pkg/protocol"
)

type PubAck struct {
	Header

	// Variable header
	PacketIdentifier uint16
}

func NewPubAck() *PubAck {
	return &PubAck{Header: Header{msgType: MsgTypePubAck}}
}

func (this *PubAck) DecodeFixedHeader(buf buffer.IoBuffer) bool {
	return false
}

func (this *PubAck) DecodeVariableHeader(buf buffer.IoBuffer) bool {
	before := buf.Len()
	this.PacketIdentifier = getUint16(buf)
	after := buf.Len()
	this.remainingLength -= RemainingLength(before - after)
	return true
}

func (this *PubAck) DecodePayload(buf buffer.IoBuffer) bool {
	panic("implement me")
}

func (this *PubAck) Encode() (buffer.IoBuffer, error) {
	buf := buffer.NewIoBuffer(0)
	putUint16(this.PacketIdentifier, buf)

	this.Header.remainingLength = RemainingLength(buf.Len())
	bufAll := this.Header.encode()

	_, err := bufAll.Write(buf.Bytes())
	if err != nil {
		return nil, errors.New(ErrorInvalidMessage)
	}

	return bufAll, nil
}

func (this *PubAck) GetHeader() (header map[string]string) {
	header = make(map[string]string, 1)
	header[protocol.StrikeHeaderMethod] = StrMsgTypePubAck
	return header
}

func (this *PubAck) GetPayload() (buf buffer.IoBuffer) {
	return nil
}
