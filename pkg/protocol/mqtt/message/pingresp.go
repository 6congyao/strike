//Created by zhbinary on 2018/12/14.
//Email: zhbinary@gmail.com
package message

import (
	"strike/pkg/buffer"
	"strike/pkg/protocol"
)

type PingResp struct {
	Header
}

func NewPingResp() *PingResp {
	return &PingResp{Header: Header{msgType: MsgTypePingResp}}
}

func (this *PingResp) DecodeFixedHeader(buf buffer.IoBuffer) bool {
	return true
}

func (this *PingResp) DecodeVariableHeader(buf buffer.IoBuffer) bool {
	panic("implement me")
}

func (this *PingResp) DecodePayload(buf buffer.IoBuffer) bool {
	panic("implement me")
}

func (this *PingResp) Encode() (buffer.IoBuffer, error) {
	buf := this.Header.encode()
	return buf, nil
}
func (this *PingResp) GetHeader() (header map[string]string) {
	header = make(map[string]string, 1)
	header[protocol.StrikeHeaderMethod] = StrMsgTypePingResp
	return header
}

func (this *PingResp) GetPayload() (buf buffer.IoBuffer) {
	return nil
}
