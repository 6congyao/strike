//Created by zhbinary on 2018/12/14.
//Email: zhbinary@gmail.com
package message

import "strike/pkg/buffer"

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

func (this *PingReq) Encode() ([]byte, error) {
	buf := this.Header.encode()
	return buf.Bytes(), nil
}
