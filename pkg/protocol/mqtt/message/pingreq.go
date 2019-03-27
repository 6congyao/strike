//Created by zhbinary on 2018/12/14.
//Email: zhbinary@gmail.com
package message

import (
	"strike/pkg/buffer"
	"strike/pkg/protocol"
)

type PingReq struct {
	Header
}

func NewPingReq() *PingReq {
	return &PingReq{Header: Header{msgType: MsgTypePingReq}}
}

func (this *PingReq) DecodeFixedHeader(buf buffer.IoBuffer) bool {
	return true
}

func (this *PingReq) DecodeVariableHeader(buf buffer.IoBuffer) bool {
	panic("implement me")
}

func (this *PingReq) DecodePayload(buf buffer.IoBuffer) bool {
	panic("implement me")
}

func (this *PingReq) Encode() (buffer.IoBuffer, error) {
	buf := this.Header.encode()
	return buf, nil
}
func (this *PingReq) GetHeader() (header map[string]string) {
	header = make(map[string]string, 1)
	header[protocol.StrikeHeaderMethod] = StrMsgTypePing
	return header
}

func (this *PingReq) GetPayload() (buf buffer.IoBuffer) {
	return nil
}
