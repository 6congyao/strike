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

type Subscribe struct {
	Header

	// Variable header
	PacketIdentifier uint16

	// Payload
	TopicFilters []TopicFilter
}

type TopicFilter struct {
	TopicName    string
	RequestedQos Qos
}

func NewSubscribe() *Subscribe {
	return &Subscribe{Header: Header{msgType: MsgTypeSubscribe}}
}

func (this *Subscribe) DecodeFixedHeader(buf buffer.IoBuffer) bool {
	return false
}

func (this *Subscribe) DecodeVariableHeader(buf buffer.IoBuffer) bool {
	before := buf.Len()
	this.PacketIdentifier = getUint16(buf)
	after := buf.Len()
	this.remainingLength -= RemainingLength(before - after)
	if this.remainingLength == 0 {
		return true
	}
	return false
}

func (this *Subscribe) DecodePayload(buf buffer.IoBuffer) bool {
	if int(this.remainingLength) > buf.Len() {
		return false
	}
	var topicFilters []TopicFilter
	if this.remainingLength > 0 {
		topicFilters = append(topicFilters, TopicFilter{TopicName: getString(buf), RequestedQos: Qos(getUint8(buf))})
	}
	this.TopicFilters = topicFilters
	return true
}

func (this *Subscribe) Encode() (buffer.IoBuffer, error) {
	buf := buffer.NewIoBuffer(0)
	putUint16(this.PacketIdentifier, buf)

	for _, tf := range this.TopicFilters {
		putString(tf.TopicName, buf)
		buf.WriteByte(byte(tf.RequestedQos))
	}

	this.remainingLength = RemainingLength(buf.Len())
	bufAll := this.Header.encode()

	_, err := bufAll.Write(buf.Bytes())
	if err != nil {
		return nil, errors.New(ErrorInvalidMessage)
	}

	return bufAll, nil
}
func (this *Subscribe) GetHeader() (header map[string]string) {
	header = make(map[string]string, 3)
	header[protocol.StrikeHeaderMethod] = StrMsgTypeSubscribe
	header[protocol.StrikeHeaderPacketID] = strconv.Itoa(int(this.PacketIdentifier))
	if b, err := json.Marshal(this.TopicFilters); err == nil {
		header[protocol.StrikeHeaderTopicFilter] = string(b)
	}
	return header
}

func (this *Subscribe) GetPayload() (buf buffer.IoBuffer) {
	return nil
}
