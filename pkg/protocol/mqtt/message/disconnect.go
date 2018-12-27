//Created by zhbinary on 2018/12/14.
//Email: zhbinary@gmail.com
package message

import "github.com/alipay/sofa-mosn/pkg/buffer"

type Disconnect struct {
	Header
}

func NewDisconnect() *Disconnect {
	return &Disconnect{Header: Header{msgType: MsgTypeDisconnect}}
}

func (this *Disconnect) DecodeFixedHeader(buf *buffer.IoBuffer) bool {
	return true
}

func (this *Disconnect) DecodeVariableHeader(buf *buffer.IoBuffer) bool {
	panic("implement me")
}

func (this *Disconnect) DecodePayload(buf *buffer.IoBuffer) bool {
	panic("implement me")
}

func (this *Disconnect) Encode() ([]byte, error) {
	buf := this.Header.encode()
	return buf.Bytes(), nil
}
