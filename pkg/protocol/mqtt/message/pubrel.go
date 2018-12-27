//Created by zhbinary on 2018/12/14.
//Email: zhbinary@gmail.com
package message

import (
	"errors"
	"strike/pkg/buffer"
)

type PubRel struct {
	Header

	// Variable header
	PacketIdentifier uint16
}

func NewPubRel() *PubRel {
	return &PubRel{Header: Header{msgType: MsgTypePubRel}}
}

func (this *PubRel) DecodeFixedHeader(buf buffer.IoBuffer) bool {
	return false
}

func (this *PubRel) DecodeVariableHeader(buf buffer.IoBuffer) bool {
	before := buf.Len()
	this.PacketIdentifier = getUint16(buf)
	after := buf.Len()
	this.remainingLength -= RemainingLength(before - after)
	return true
}

func (this *PubRel) DecodePayload(buf buffer.IoBuffer) bool {
	panic("implement me")
}

func (this *PubRel) Encode() ([]byte, error) {
	buf := buffer.NewIoBuffer(0)
	putUint16(this.PacketIdentifier, buf)

	this.remainingLength = RemainingLength(buf.Len())
	bufAll := this.Header.encode()

	_, err := bufAll.Write(buf.Bytes())
	if err != nil {
		return nil, errors.New(ErrorInvalidMessage)
	}

	return bufAll.Bytes(), nil
}
