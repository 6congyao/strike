//Created by zhbinary on 2018/12/15.
//Email: zhbinary@gmail.com
package message

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strike/pkg/buffer"
	"strike/pkg/network"
	"strike/pkg/protocol"
)

type Codec struct {
	header       *Header
	decoderState DecoderState
	msg          Message
}

type DecoderState uint8

const (
	DecoderStateReadFixedHeader = DecoderState(iota)
	DecoderStateReadVariableHeader
	DecoderStateReadPayload
)

func NewCodec() *Codec {
	return &Codec{decoderState: DecoderStateReadFixedHeader}
}

func (this *Codec) doDecode(buf buffer.IoBuffer) (msgs []Message, e error) {
	defer func() {
		if err := recover(); err != nil {
			if msgs != nil {
				return
			}
			buf.Restore()
			if re, ok := err.(error); ok {
				e = re
				fmt.Printf("Error:%v", err)
				//debug.PrintStack()
			} else {
				fmt.Printf("Unknown error ")
			}
		}
	}()

	if buf == nil || buf.Len() == 0 {
		return nil, errors.New("Invalid buffer ")
	}

	for buf.Len() > 0 {
		switch this.decoderState {
		case DecoderStateReadFixedHeader:
			buf.Mark()
			this.header = &Header{}
			this.header.decode(buf)
			this.msg = NewMessage(*this.header)
			if this.msg == nil {
				return nil, errors.New("No matched message ")
			}
			if this.msg.DecodeFixedHeader(buf) {
				msgs = append(msgs, this.msg)
				this.msg = nil
				this.decoderState = DecoderStateReadFixedHeader
			} else {
				this.decoderState = DecoderStateReadVariableHeader
			}
			break
		case DecoderStateReadVariableHeader:
			buf.Mark()
			//wait for remaining bytes here
			if RemainingLength(buf.Len()) < this.header.remainingLength {
				panic(errors.New(ErrorInvalidRemainingLength))
			}
			if this.msg.DecodeVariableHeader(buf) {
				msgs = append(msgs, this.msg)
				this.msg = nil
				this.decoderState = DecoderStateReadFixedHeader
			} else {
				this.decoderState = DecoderStateReadPayload
			}
			break
		case DecoderStateReadPayload:
			buf.Mark()
			if this.msg.DecodePayload(buf) {
				msgs = append(msgs, this.msg)
				this.msg = nil
				this.decoderState = DecoderStateReadFixedHeader
			} else {
				this.decoderState = DecoderStateReadPayload
			}
			break
		default:
			break
		}
		buf.Mark()
	}
	return
}

func doEncode(m Message) (buffer.IoBuffer, error) {
	return m.Encode()
}

func (c *Codec) Encode(ctx context.Context, model interface{}) (buffer.IoBuffer, error) {
	panic("implement me")
}

func (c *Codec) Decode(ctx context.Context, data buffer.IoBuffer, filter protocol.DecodeFilter) {
	msgs, err := c.doDecode(data)

	if err != nil {
		log.Println("mqtt decode error:", err)
		return
	}
	for _, msg := range msgs {
		streamID := protocol.GenerateID()
		// notify
		status := filter.OnDecodeDone(streamID, msg)
		if status == network.Stop {
			return
		}
	}
}
