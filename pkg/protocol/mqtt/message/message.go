//Created by zhbinary on 2018/12/14.
//Email: zhbinary@gmail.com
package message

import (
	"encoding/binary"
	"errors"
	"strike/pkg/buffer"
)

type Type uint8

type Message interface {
	DecodeFixedHeader(buf buffer.IoBuffer) bool
	DecodeVariableHeader(buf buffer.IoBuffer) bool
	DecodePayload(buf buffer.IoBuffer) bool
	Encode() (buffer.IoBuffer, error)
	GetHeader() (map[string]string)
	GetPayload() (buffer.IoBuffer)
	//Equal(message Message) bool
}

const (
	MsgTypeConnect = Type(iota + 1)
	MsgTypeConnAck
	MsgTypePublish
	MsgTypePubAck
	MsgTypePubRec
	MsgTypePubRel
	MsgTypePubComp
	MsgTypeSubscribe
	MsgTypeSubAck
	MsgTypeUnsubscribe
	MsgTypeUnsubAck
	MsgTypePingReq
	MsgTypePingResp
	MsgTypeDisconnect
	MsgTypeInvalid

	ErrorInvalidRemainingLength = "Invalid remaining length "
	ErrorInvalidMessage         = "Invalid message "
)

const (
	StrMsgTypeConnect     = "Connect"
	StrMsgTypeConnectAck  = "ConnAck"
	StrMsgTypePublish     = "Publish"
	StrMsgTypeSubscribe   = "Subscribe"
	StrMsgTypePing        = "Ping"
	StrMsgTypeDisconnect  = "Disconnect"
	StrMsgTypePingResp    = "PingResp"
	StrMsgTypePubAck      = "PubAck"
	StrMsgTypePubRec      = "PubRec"
	StrMsgTypePubRel      = "PubRel"
	StrMsgTypePubComp     = "PubComp"
	StrMsgTypeSubAck      = "SubAck"
	StrMsgTypeUnsubscribe = "Unsubscribe"
	StrMsgTypeUnsubAck    = "UnsubAck"
)

const (
	RetCodeAccepted = ReturnCode(iota)
	RetCodeUnacceptableProtocolVersion
	RetCodeIdentifierRejected
	RetCodeServerUnavailable
	RetCodeBadUsernameOrPassword
	RetCodeNotAuthorized
	RetCodeInvalid
)

func (this Type) string() (str string) {
	name := func(t Type, name string) {
		if this&t == 0 {
			return
		}
		str += "|"
		str += name
		return
	}

	name(MsgTypeConnect, "Connect")
	name(MsgTypeConnAck, "ConnectAck")
	name(MsgTypePublish, "Publish")
	name(MsgTypePubAck, "PubAck")
	name(MsgTypePubRec, "PubRec")
	name(MsgTypePubRel, "PubRel")
	name(MsgTypePubComp, "PubComp")
	name(MsgTypeSubscribe, "Subscribe")
	name(MsgTypeSubAck, "SubAck")
	name(MsgTypeUnsubscribe, "Unsubscribe")
	name(MsgTypeUnsubAck, "UnsubAck")
	name(MsgTypePingReq, "PingReq")
	name(MsgTypePingResp, "PingResp")
	name(MsgTypeDisconnect, "Disconnect")

	return
}

func (mt Type) IsValid() bool {
	return mt >= MsgTypeConnect && mt < MsgTypeInvalid
}

const (
	QosAtMostOnce  = Qos(iota) // Qos 1
	QosAtLeastOnce             // Qos 2
	QosExactlyOnce             // Qos 3
	QosTypeInvalid
)

type Qos uint8

func (qos Qos) IsValid() bool {
	return qos < QosTypeInvalid
}

func (qos Qos) HasId() bool {
	return qos == QosAtLeastOnce || qos == QosExactlyOnce
}

type ReturnCode uint8

func (rc ReturnCode) IsValid() bool {
	return rc >= RetCodeAccepted && rc < RetCodeInvalid
}

type RemainingLength int32

func (this RemainingLength) IsValid() bool {
	return this >= 0 && this <= 268435455
}

func NewMessage(header Header) (msg Message) {
	switch header.msgType {
	case MsgTypeConnect:
		msg = &Connect{Header: header}
		break
	case MsgTypeConnAck:
		msg = &ConnAck{Header: header}
		break
	case MsgTypePublish:
		msg = &Publish{Header: header}
		break
	case MsgTypePubAck:
		msg = &PubAck{Header: header}
		break
	case MsgTypePubRec:
		msg = &PubRec{Header: header}
		break
	case MsgTypePubRel:
		msg = &PubRel{Header: header}
		break
	case MsgTypePubComp:
		msg = &PubComp{Header: header}
		break
	case MsgTypeSubscribe:
		msg = &Subscribe{Header: header}
		break
	case MsgTypeSubAck:
		msg = &SubAck{Header: header}
		break
	case MsgTypeUnsubscribe:
		msg = &Unsubscribe{Header: header}
		break
	case MsgTypeUnsubAck:
		msg = &UnsubAck{Header: header}
		break
	case MsgTypePingReq:
		msg = &PingReq{Header: header}
		break
	case MsgTypePingResp:
		msg = &PingResp{Header: header}
		break
	case MsgTypeDisconnect:
		msg = &Disconnect{Header: header}
		break
	default:
		msg = nil
		break
	}
	return
}

func decodeLength(buf buffer.IoBuffer) (v RemainingLength) {
	var shift uint
	for i := 0; i < 4; i++ {
		if b, err := buf.ReadByte(); err == nil {
			v |= RemainingLength(b&0x7f) << shift
			if b&0x80 == 0 {
				return
			}
			shift += 7
		} else {
			panic(err)
		}
	}
	panic(errors.New(ErrorInvalidRemainingLength))
}

func encodeLength(len RemainingLength, buf buffer.IoBuffer) {
	if len == 0 {
		buf.WriteByte(0)
		return
	}
	for len > 0 {
		digit := len & 0x7f
		len = len >> 7
		if len > 0 {
			digit = digit | 0x80
		}
		buf.WriteByte(byte(digit))
	}
}

func getString(buf buffer.IoBuffer) (s string) {
	strLen := getUint16(buf)

	if buf.Len() < int(strLen) {
		panic(errors.New(ErrorInvalidRemainingLength))
	}

	bytes := make([]byte, strLen)
	if n, err := buf.Read(bytes); err == nil {
		if n != int(strLen) {
			panic(errors.New(ErrorInvalidRemainingLength))
		} else {
			return string(bytes)
		}
	} else {
		panic(err)
	}
}

func getUint16(buf buffer.IoBuffer) (i uint16) {
	if buf.Len() < 2 {
		panic(errors.New(ErrorInvalidRemainingLength))
	}

	bytes := make([]byte, 2)
	if n, err := buf.Read(bytes); err == nil {
		if n != 2 {
			panic(errors.New(ErrorInvalidRemainingLength))
		} else {
			return binary.BigEndian.Uint16(bytes[:2])
		}
	} else {
		panic(err)
	}
}

func getUint8(buf buffer.IoBuffer) (i uint8) {
	if b, err := buf.ReadByte(); err == nil {
		return b
	} else {
		panic(err)
	}
}

func putString(s string, buf buffer.IoBuffer) {
	len := uint16(len(s))
	putUint16(len, buf)
	if s == "" {
		panic(errors.New("Nil utf8 string "))
	}
	buf.Write([]byte(s))
}

func putUint16(i uint16, buf buffer.IoBuffer) {
	buf.WriteByte(byte(i >> 8))
	buf.WriteByte(byte(i))
}

func boolToByte(val bool) byte {
	if val {
		return byte(1)
	}
	return byte(0)
}
