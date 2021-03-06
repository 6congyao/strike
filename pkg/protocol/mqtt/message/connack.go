//Created by zhbinary on 2018/12/14.
//Email: zhbinary@gmail.com
package message

import (
	"errors"
	"strike/pkg/buffer"
	"strike/pkg/protocol"
)

type ConnAck struct {
	Header

	// Variable header
	SessionPresentFlag bool
	ReturnCode         ReturnCode
}

func NewConnAck() *ConnAck {
	return &ConnAck{Header: Header{msgType: MsgTypeConnAck}}
}

func (this *ConnAck) DecodeFixedHeader(buf buffer.IoBuffer) bool {
	return false
}

func (this *ConnAck) DecodeVariableHeader(buf buffer.IoBuffer) bool {
	before := buf.Len()
	b1 := getUint8(buf)
	this.SessionPresentFlag = b1&0x01 > 0
	this.ReturnCode = ReturnCode(getUint8(buf))
	if !this.ReturnCode.IsValid() {
		panic(errors.New(ErrorInvalidMessage))
	}
	after := buf.Len()
	this.remainingLength -= RemainingLength(before - after)
	return true
}

func (this *ConnAck) DecodePayload(buf buffer.IoBuffer) bool {
	panic("implement me")
}

func (this *ConnAck) Encode() (buffer.IoBuffer, error) {
	buf := buffer.NewIoBuffer(0)
	err := buf.WriteByte(boolToByte(this.SessionPresentFlag))
	if err != nil {
		return nil, err
	}

	err = buf.WriteByte(byte(this.ReturnCode))
	if err != nil {
		return nil, err
	}

	this.Header.remainingLength = RemainingLength(buf.Len())
	if !this.Header.remainingLength.IsValid() {
		return nil, errors.New(ErrorInvalidMessage)
	}
	bufAll := this.Header.encode()
	_, err = bufAll.Write(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return bufAll, nil
}
func (this *ConnAck) GetHeader() (header map[string]string) {
	header = make(map[string]string, 1)
	header[protocol.StrikeHeaderMethod] = StrMsgTypeConnectAck
	return header
}

func (this *ConnAck) GetPayload() (buf buffer.IoBuffer) {
	return nil
}
