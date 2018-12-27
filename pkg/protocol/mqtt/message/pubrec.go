//Created by zhbinary on 2018/12/14.
//Email: zhbinary@gmail.com
package message

import (
	"errors"
	"github.com/alipay/sofa-mosn/pkg/buffer"
)

type PubRec struct {
	Header

	// Variable header
	PacketIdentifier uint16
}

func NewPubRec() *PubRec {
	return &PubRec{Header: Header{msgType: MsgTypePubRec}}
}

func (this *PubRec) DecodeFixedHeader(buf *buffer.IoBuffer) bool {
	return false
}

func (this *PubRec) DecodeVariableHeader(buf *buffer.IoBuffer) bool {
	before := buf.Len()
	this.PacketIdentifier = getUint16(buf)
	after := buf.Len()
	this.remainingLength -= RemainingLength(before - after)
	return true
}

func (this *PubRec) DecodePayload(buf *buffer.IoBuffer) bool {
	panic("implement me")
}

func (this *PubRec) Encode() ([]byte, error) {
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
