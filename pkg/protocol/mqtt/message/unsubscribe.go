//Created by zhbinary on 2018/12/14.
//Email: zhbinary@gmail.com
package message

import (
	"encoding/json"
	"errors"
	"strconv"
	"strike/pkg/buffer"
	"strike/pkg/protocol"
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

func (this *Unsubscribe) DecodeFixedHeader(buf buffer.IoBuffer) bool {
	return false
}

func (this *Unsubscribe) DecodeVariableHeader(buf buffer.IoBuffer) bool {
	before := buf.Len()
	this.PacketIdentifier = getUint16(buf)
	after := buf.Len()
	this.remainingLength -= RemainingLength(before - after)
	if this.remainingLength == 0 {
		return true
	}
	return false
}

func (this *Unsubscribe) DecodePayload(buf buffer.IoBuffer) bool {
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

func (this *Unsubscribe) Encode() (buffer.IoBuffer, error) {
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
func (this *Unsubscribe) GetHeader() (header map[string]string) {
	header = make(map[string]string, 3)
	header[protocol.StrikeHeaderMethod] = StrMsgTypeUnsubscribe
	header[protocol.StrikeHeaderPacketID] = strconv.Itoa(int(this.PacketIdentifier))
	if b, err := json.Marshal(this.TopicNames); err == nil {
		header[protocol.StrikeHeaderTopicFilter] = string(b)
	}
	return header
}

func (this *Unsubscribe) GetPayload() (buf buffer.IoBuffer) {
	return nil
}
