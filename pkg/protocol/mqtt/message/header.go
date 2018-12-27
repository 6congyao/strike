//Created by zhbinary on 2018/12/14.
//Email: zhbinary@gmail.com
package message

import (
	"errors"
	"github.com/alipay/sofa-mosn/pkg/buffer"
)

type Header struct {
	msgType         Type
	msgFlag         byte
	remainingLength RemainingLength
}

func (this *Header) decode(buf *buffer.IoBuffer) {
	byte1 := getUint8(buf)
	this.msgType = Type(byte1 & 0xf0 >> 4)
	if !this.msgType.IsValid() {
		panic(errors.New("Invalid message type "))
	}
	this.msgFlag = byte1 & 0x0f
	this.remainingLength = decodeLength(buf)
}

func (this *Header) encode() *buffer.IoBuffer {
	buf := &buffer.IoBuffer{}
	buf.WriteByte(byte(this.msgType)<<4 | this.msgFlag)
	encodeLength(this.remainingLength, buf)
	return buf
}
