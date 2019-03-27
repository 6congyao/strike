/*
 * Copyright (c) 2018. LuCongyao <6congyao@gmail.com> .
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this work except in compliance with the License.
 * You may obtain a copy of the License in the LICENSE file, or at:
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mqtt

import (
	"context"
	"log"
	"strconv"
	"strike/pkg/buffer"
	"strike/pkg/network"
	"strike/pkg/protocol"
	"strike/pkg/protocol/mqtt/message"
	"strike/pkg/stream"
	"strike/pkg/types"
)

func init() {
	stream.Register(protocol.MQTT, &streamConnFactory{})
}

type streamConnFactory struct{}

func (f *streamConnFactory) CreateClientStreamConnection(context context.Context, connection network.ClientConnection,
	streamConnCallbacks stream.StreamConnectionEventListener,
	callbacks network.ConnectionEventListener) stream.ClientStreamConnection {
	return nil
}

func (f *streamConnFactory) CreateServerStreamConnection(context context.Context, connection network.Connection,
	callbacks stream.ServerStreamConnectionEventListener) stream.ServerStreamConnection {
	return newStreamConnection(context, connection, nil, callbacks)
}

func (f *streamConnFactory) CreateBiDirectStreamConnection(context context.Context, connection network.ClientConnection,
	clientCallbacks stream.StreamConnectionEventListener,
	serverCallbacks stream.ServerStreamConnectionEventListener) stream.ClientStreamConnection {
	return nil
}

func newStreamConnection(context context.Context, connection network.Connection, clientCallbacks stream.StreamConnectionEventListener,
	serverCallbacks stream.ServerStreamConnectionEventListener) stream.ServerStreamConnection {
	sc := &streamConnection{
		context:      context,
		connection:   connection,
		protocol:     protocol.MQTT,
		codec:        message.NewCodec(),
		cscCallbacks: clientCallbacks,
		sscCallbacks: serverCallbacks,
	}

	connection.AddConnectionEventListener(sc)
	return sc
}

// protocol.DecodeFilter
// stream.ServerStreamConnection
// stream.StreamConnection
type streamConnection struct {
	context       context.Context
	protocol      protocol.Protocol
	codec         protocol.Codec
	connection    network.Connection
	connCallbacks network.ConnectionEventListener
	// Client Stream Conn Callbacks
	cscCallbacks stream.StreamConnectionEventListener
	// Server Stream Conn Callbacks
	sscCallbacks stream.ServerStreamConnectionEventListener
}

func (sc *streamConnection) OnDecodeHeader(streamID uint64, headers protocol.HeaderMap, endStream bool) network.FilterStatus {
	return network.Continue
}

func (sc *streamConnection) OnDecodeData(streamID uint64, data buffer.IoBuffer, endStream bool) network.FilterStatus {
	return network.Continue
}

func (sc *streamConnection) OnDecodeTrailer(streamID uint64, trailers protocol.HeaderMap) network.FilterStatus {
	return network.Continue
}

func (sc *streamConnection) OnDecodeDone(streamID uint64, result interface{}) network.FilterStatus {
	if msg, ok := result.(message.Message); ok {
		srvStream := &serverStream{
			streamBase: streamBase{
				id:      streamID,
				req:     msg,
				context: context.WithValue(sc.context, types.ContextKeyStreamID, streamID),
			},
			connection: sc,
		}

		srvStream.receiver = sc.sscCallbacks.NewStreamDetect(sc.context, streamID, srvStream)
		srvStream.handleMessage()
	}

	return network.Continue
}

func (sc *streamConnection) OnDecodeError(err error, headers protocol.HeaderMap) {
	log.Println("decode error:", err)
}

func (sc *streamConnection) Dispatch(buf buffer.IoBuffer) {
	sc.codec.Decode(sc.context, buf, sc)
}

func (sc *streamConnection) Protocol() protocol.Protocol {
	return sc.protocol
}

func (sc *streamConnection) GoAway() {
	panic("implement me")
}

func (sc *streamConnection) OnEvent(event network.ConnectionEvent) {
	if event.IsClose() || event.ConnectFailure() {
		// clear
	}
}

type streamBase struct {
	id      uint64
	req     message.Message
	res     message.Message
	context context.Context

	receiver  stream.StreamReceiver
	streamCbs []stream.StreamEventListener
}

func (sb *streamBase) ID() uint64 {
	return sb.id
}

func (sb *streamBase) AddEventListener(streamCb stream.StreamEventListener) {
	sb.streamCbs = append(sb.streamCbs, streamCb)
}

func (sb *streamBase) RemoveEventListener(streamCb stream.StreamEventListener) {
	cbIdx := -1

	for i, streamCb := range sb.streamCbs {
		if streamCb == streamCb {
			cbIdx = i
			break
		}
	}

	if cbIdx > -1 {
		sb.streamCbs = append(sb.streamCbs[:cbIdx], sb.streamCbs[cbIdx+1:]...)
	}
}

func (sb *streamBase) ResetStream(reason stream.StreamResetReason) {
	for _, cb := range sb.streamCbs {
		cb.OnResetStream(reason)
	}
}

// stream.StreamSender
// stream.Stream
type serverStream struct {
	streamBase
	connection *streamConnection
}

func (ss *serverStream) AppendHeaders(ctx context.Context, headerIn protocol.HeaderMap, endStream bool) error {
	var msgType string
	var status string
	var ok bool
	var strPacketId string

	if status, ok = headerIn.Get(types.HeaderStatus); ok {
		headerIn.Del(types.HeaderStatus)
	}

	if msgType, ok = headerIn.Get(types.HeaderMethod); ok {
		headerIn.Del(types.HeaderMethod)
	} else {
		status = ""
	}

	// todo: handle other responses here
	// @liuzhen
	switch msgType {
	case message.StrMsgTypeConnect:
		ack := message.NewConnAck()
		// todo:
		if status == "200" {
			ack.ReturnCode = message.RetCodeAccepted
		} else {
			ack.ReturnCode = message.RetCodeNotAuthorized
		}
		ss.res = ack

		break
	case message.StrMsgTypeConnectAck:
		break
	case message.StrMsgTypePublish:
		ack := message.NewPubAck()
		if strPacketId, ok = headerIn.Get(types.HeaderPacketID); ok {
			headerIn.Del(types.HeaderPacketID)
		}
		if packetId, err := strconv.Atoi(strPacketId); err == nil {
			ack.PacketIdentifier = uint16(packetId)
		}

		ss.res = ack
		break
	case message.StrMsgTypePubAck:
		break
	case message.StrMsgTypePubRec:
		break
	case message.StrMsgTypePubRel:
		break
	case message.StrMsgTypePubComp:
		break
	case message.StrMsgTypeSubscribe:
		ack := message.NewSubAck()
		if strPacketId, ok = headerIn.Get(types.HeaderPacketID); ok {
			headerIn.Del(types.HeaderPacketID)
		}
		if packetId, err := strconv.Atoi(strPacketId); err == nil {
			ack.PacketIdentifier = uint16(packetId)
		}

		ss.res = ack
	case message.StrMsgTypeSubAck:
		break
	case message.StrMsgTypeUnsubscribe:
		ack := message.NewUnsubAck()
		if strPacketId, ok = headerIn.Get(types.HeaderPacketID); ok {
			headerIn.Del(types.HeaderPacketID)
		}
		if packetId, err := strconv.Atoi(strPacketId); err == nil {
			ack.PacketIdentifier = uint16(packetId)
		}
		ss.res = ack
		break
	case message.StrMsgTypeUnsubAck:
		break
	case message.StrMsgTypePing:
		ss.res = message.NewPingResp()
	case message.StrMsgTypePingResp:
		break
	case message.StrMsgTypeDisconnect:
		break

	default:
		break
	}

	if endStream {
		ss.endStream()
	}
	return nil
}

func (ss *serverStream) AppendData(ctx context.Context, data buffer.IoBuffer, endStream bool) error {
	panic("implement me")
}

func (ss *serverStream) AppendTrailers(ctx context.Context, trailers protocol.HeaderMap) error {
	ss.endStream()
	return nil
}

func (ss *serverStream) GetStream() stream.Stream {
	return ss
}

func (ss *serverStream) ReadDisable(disable bool) {
	panic("unsupported")
}

func (ss *serverStream) handleMessage() {
	if ss.req == nil {
		return
	}

	// todo: + msg.Header() & msg.Payload()
	// @liuzhen
	header := make(map[string]string, 2)
	var payload buffer.IoBuffer

	header = ss.req.GetHeader()
	payload = ss.req.GetPayload()

	ss.receiver.OnReceiveHeaders(ss.context, protocol.CommonHeader(header), payload == nil)

	if payload != nil {
		ss.receiver.OnReceiveHeaders(ss.context, protocol.CommonHeader(header), true)
	}
}

func (ss *serverStream) endStream() {
	ss.doSend()
}

func (ss *serverStream) doSend() {
	// todo: remove after mqtt codec updated
	// @liuzhen
	buf, _ := ss.res.Encode()

	ss.connection.connection.Write(buf)

}
