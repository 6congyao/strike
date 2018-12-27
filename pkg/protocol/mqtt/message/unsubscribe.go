//Created by zhbinary on 2018/12/14.
//Email: zhbinary@gmail.com
package message

import (
	"errors"
	"github.com/alipay/sofa-mosn/pkg/buffer"
)

type Unsubscribe struct {
	Header

	// Variable header
	PacketIdentifier uint16

	// Payload
	TopicNames []string
}

func NewUnsubscribe() *Unsubscribe {
	return &Unsubscribe{Header: Header{msgType: MsgTypeUnsubscribe}}
}

func (this *Unsubscribe) DecodeFixedHeader(buf *buffer.IoBuffer) bool {
	return false
}

func (this *Unsubscribe) DecodeVariableHeader(buf *buffer.IoBuffer) bool {
	before := buf.Len()
	this.PacketIdentifier = getUint16(buf)
	after := buf.Len()
	this.remainingLength -= RemainingLength(before - after)
	if this.remainingLength == 0 {
		return true
	}
	return false
}

func (this *Unsubscribe) DecodePayload(buf *buffer.IoBuffer) bool {
	if int(this.remainingLength) > buf.Len() {
		return false
	}
	var topicNames []string
	if this.remainingLength > 0 {
		topicNames = append(topicNames, getString(buf))
	}
	this.TopicNames = topicNames
	return true
}

func (this *Unsubscribe) Encode() ([]byte, error) {
	buf := &buffer.IoBuffer{}
	putUint16(this.PacketIdentifier, buf)

	this.Header.remainingLength = RemainingLength(buf.Len())
	bufAll := this.Header.encode()

	_, err := bufAll.Write(buf.Bytes())
	if err != nil {
		return nil, errors.New(ErrorInvalidMessage)
	}

	return bufAll.Bytes(), nil
}
