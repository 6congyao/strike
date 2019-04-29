//Created by zhbinary on 2018/12/14.
//Email: zhbinary@gmail.com
package message

import (
	"errors"
	"fmt"
	"strike/pkg/buffer"
	"strike/pkg/protocol"
)

type Connect struct {
	Header

	// Variable header
	ProtocolName  string
	ProtocolLevel uint8
	UserNameFlag  bool
	passwordFlag  bool
	WillRetain    bool
	WillQos       Qos
	WillFlag      bool
	CleanSession  bool
	KeepAlive     uint16

	// Payload
	ClientId    string
	WillTopic   string
	WillMessage string
	Username    string
	Password    string
}

func NewConnect() *Connect {
	return &Connect{Header: Header{msgType: MsgTypeConnect}}
}

func (this *Connect) DecodeFixedHeader(buf buffer.IoBuffer) bool {
	return false
}

func (this *Connect) DecodeVariableHeader(buf buffer.IoBuffer) bool {
	before := buf.Len()
	this.ProtocolName = getString(buf)
	this.ProtocolLevel = getUint8(buf)
	flags := getUint8(buf)
	this.UserNameFlag = flags&0x80 > 0
	this.passwordFlag = flags&0x40 > 0
	this.WillRetain = flags&0x20 > 0
	this.WillQos = Qos(flags & 0x18 >> 3)
	this.WillFlag = flags&0x04 > 0
	this.CleanSession = flags&0x02 > 0
	this.KeepAlive = getUint16(buf)
	after := buf.Len()
	this.remainingLength -= RemainingLength(before - after)
	return false
}

func (this *Connect) DecodePayload(buf buffer.IoBuffer) bool {
	if int(this.remainingLength) > buf.Len() {
		return false
	}
	before := buf.Len()
	this.ClientId = getString(buf)

	if this.WillFlag {
		this.WillTopic = getString(buf)
		this.WillMessage = getString(buf)
	}

	if this.UserNameFlag {
		this.Username = getString(buf)
	}

	if this.passwordFlag {
		this.Password = getString(buf)
	}
	after := buf.Len()

	this.remainingLength -= RemainingLength(before - after)
	return true
}

func (this *Connect) Encode() (buffer.IoBuffer, error) {
	if !this.WillQos.IsValid() {
		return nil, errors.New(ErrorInvalidMessage)
	}

	buf := buffer.NewIoBuffer(0)
	// Variable header
	flags := boolToByte(this.UserNameFlag) << 7
	flags |= boolToByte(this.passwordFlag) << 6
	flags |= boolToByte(this.WillRetain) << 5
	flags |= byte(this.WillQos) << 3
	flags |= boolToByte(this.WillFlag) << 2
	flags |= boolToByte(this.CleanSession) << 1
	putString(this.ProtocolName, buf)
	buf.WriteByte(this.ProtocolLevel)
	buf.WriteByte(flags)
	putUint16(this.KeepAlive, buf)

	// Payload
	putString(this.ClientId, buf)
	if this.WillFlag {
		putString(this.WillTopic, buf)
		putString(this.WillMessage, buf)
	}

	if this.UserNameFlag {
		putString(this.Username, buf)
	}

	if this.passwordFlag {
		putString(this.Password, buf)
	}

	// Fixed header
	this.Header.remainingLength = RemainingLength(buf.Len())
	if !this.Header.remainingLength.IsValid() {
		return nil, errors.New(ErrorInvalidMessage)
	}
	bufAll := this.Header.encode()
	_, err := bufAll.Write(buf.Bytes())
	if err != nil {
		return nil, err
	}

	return bufAll, nil
}

func (this *Connect) GetHeader() (header map[string]string) {
	header = make(map[string]string, 6)
	header[protocol.StrikeHeaderMethod] = StrMsgTypeConnect
	header[protocol.StrikeHeaderCredential] = this.Password
	header[protocol.StrikeHeaderClientID] = this.ClientId
	header[protocol.StrikeHeaderWillTopic] = this.WillTopic
	header[protocol.StrikeHeaderWillMessage] = this.WillMessage
	header[protocol.StrikeHeaderUsername] = this.Username
	header[protocol.StrikeHeaderKeepAlive] = fmt.Sprintf("%d", this.KeepAlive)
	return header
}

func (this *Connect) GetPayload() (buf buffer.IoBuffer) {
	return nil
}
