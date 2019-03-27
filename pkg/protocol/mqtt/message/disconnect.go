//Created by zhbinary on 2018/12/14.
//Email: zhbinary@gmail.com
package message

import (
	"strike/pkg/buffer"
	"strike/pkg/protocol"
)

type Disconnect struct {
	Header
}

func NewDisconnect() *Disconnect {
	return &Disconnect{Header: Header{msgType: MsgTypeDisconnect}}
}

func (this *Disconnect) DecodeFixedHeader(buf buffer.IoBuffer) bool {
	return true
}

func (this *Disconnect) DecodeVariableHeader(buf buffer.IoBuffer) bool {
	panic("implement me")
}

func (this *Disconnect) DecodePayload(buf buffer.IoBuffer) bool {
	panic("implement me")
}

func (this *Disconnect) Encode() (buffer.IoBuffer, error) {
	buf := this.Header.encode()
	return buf, nil
}
func (this *Disconnect) GetHeader() (header map[string]string) {
	header = make(map[string]string, 1)
	header[protocol.StrikeHeaderMethod] = StrMsgTypeDisconnect
	return header
}

func (this *Disconnect) GetPayload() (buf buffer.IoBuffer) {
	return nil
}
