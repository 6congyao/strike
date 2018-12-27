//Created by zhbinary on 2018/12/14.
//Email: zhbinary@gmail.com
package message

import "github.com/alipay/sofa-mosn/pkg/buffer"

type PingResp struct {
	Header
}

func NewPingResp() *PingResp {
	return &PingResp{Header: Header{msgType: MsgTypePingResp}}
}

func (this *PingResp) DecodeFixedHeader(buf *buffer.IoBuffer) bool {
	return true
}

func (this *PingResp) DecodeVariableHeader(buf *buffer.IoBuffer) bool {
	panic("implement me")
}

func (this *PingResp) DecodePayload(buf *buffer.IoBuffer) bool {
	panic("implement me")
}

func (this *PingResp) Encode() ([]byte, error) {
	buf := this.Header.encode()
	return buf.Bytes(), nil
}
