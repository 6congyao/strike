//Created by zhbinary on 2018/12/14.
//Email: zhbinary@gmail.com
package message

import (
	"errors"
	"github.com/alipay/sofa-mosn/pkg/buffer"
)

type PubComp struct {
	Header

	// Variable header
	PacketIdentifier uint16
}

func NewPubComp() *PubComp {
	return &PubComp{Header: Header{msgType: MsgTypePubComp}}
}

func (this *PubComp) DecodeFixedHeader(buf *buffer.IoBuffer) bool {
	return false
}

func (this *PubComp) DecodeVariableHeader(buf *buffer.IoBuffer) bool {
	before := buf.Len()
	this.PacketIdentifier = getUint16(buf)
	after := buf.Len()
	this.remainingLength -= RemainingLength(before - after)
	return true
}

func (this *PubComp) DecodePayload(buf *buffer.IoBuffer) bool {
	panic("implement me")
}

func (this *PubComp) Encode() ([]byte, error) {
	buf := &buffer.IoBuffer{}
	putUint16(this.PacketIdentifier, buf)

	this.remainingLength = RemainingLength(buf.Len())
	bufAll := this.Header.encode()

	_, err := bufAll.Write(buf.Bytes())
	if err != nil {
		return nil, errors.New(ErrorInvalidMessage)
	}

	return bufAll.Bytes(), nil
}
