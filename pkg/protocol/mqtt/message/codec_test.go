//Created by zhbinary on 2018/12/19.
//Email: zhbinary@gmail.com
package message

import (
	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/smartystreets/goconvey/convey"
	"reflect"
	"testing"
)

func TestCodec_Decode(t *testing.T) {
	convey.Convey("TestCodec_Decode", t, func() {
		codec := NewCodec()

		buf := &buffer.IoBuffer{}

		msgsLeft := []Connect{{Header: Header{msgType: MsgTypeConnect}, ProtocolName: "p0", ProtocolLevel: 3, UserNameFlag: true, passwordFlag: true},
			{Header: Header{msgType: MsgTypeConnect}, ProtocolName: "p1", ProtocolLevel: 0, UserNameFlag: false, passwordFlag: true},
			{Header: Header{msgType: MsgTypeConnect}, ProtocolName: "p2", ProtocolLevel: 1, UserNameFlag: true, passwordFlag: false},
			{Header: Header{msgType: MsgTypeConnect}, ProtocolName: "p3", ProtocolLevel: 4, UserNameFlag: true, passwordFlag: false},
			{Header: Header{msgType: MsgTypeConnect}, ProtocolName: "p4", ProtocolLevel: 7, UserNameFlag: false, passwordFlag: false},
		}

		for i := 0; i < len(msgsLeft); i++ {
			//msgLeft := msgsLeft[i]
			bytes, err := msgsLeft[i].Encode()
			convey.So(err, convey.ShouldBeNil)
			msgsLeft[i].remainingLength = 0

			_, err = buf.Write(bytes)
			convey.So(err, convey.ShouldBeNil)

			msgsRight, err := codec.Decode(buf)
			convey.So(err, convey.ShouldBeNil)

			if right, ok := msgsRight[0].(*Connect); ok {
				if msgsLeft[i] != *right {
					t.Errorf("left:%v right:%v\n", msgsLeft[i], right)
				}
			}
		}

		codec = NewCodec()
		buf.Reset()

		bytes, err := msgsLeft[0].Encode()
		convey.So(err, convey.ShouldBeNil)
		buf.Write(bytes)

		bytes, err = msgsLeft[1].Encode()
		convey.So(err, convey.ShouldBeNil)
		buf.Write(bytes)

		msgsRight, err := codec.Decode(buf)
		convey.So(err, convey.ShouldBeNil)

		convey.So(len(msgsRight), convey.ShouldEqual, 2)

		for i := 0; i < len(msgsRight); i++ {
			msgsLeft[i].remainingLength = 0
			if right, ok := msgsRight[i].(*Connect); ok {
				if msgsLeft[i] != *right {
					t.Errorf("left:%v right:%v\n", msgsLeft[i], right)
				}
			}
		}

		codec = NewCodec()
		buf.Reset()

		bytes, err = msgsLeft[0].Encode()
		convey.So(err, convey.ShouldBeNil)
		buf.Write(bytes)

		bytes, err = msgsLeft[1].Encode()
		convey.So(err, convey.ShouldBeNil)
		buf.Write(bytes)

		bytes, err = msgsLeft[2].Encode()
		convey.So(err, convey.ShouldBeNil)
		buf.Write(bytes[:3])

		msgsRight, err = codec.Decode(buf)
		convey.So(err, convey.ShouldBeNil)

		convey.So(len(msgsRight), convey.ShouldEqual, 2)

		buf.Write(bytes[3:])
		msgsRight1, err := codec.Decode(buf)
		convey.So(err, convey.ShouldBeNil)
		convey.So(len(msgsRight1), convey.ShouldEqual, 1)

		msgsRight = append(msgsRight, msgsRight1[0])

		for i := 0; i < len(msgsRight); i++ {
			msgsLeft[i].remainingLength = 0
			if right, ok := msgsRight[i].(*Connect); ok {
				if msgsLeft[i] != *right {
					t.Errorf("left:%v right:%v\n", msgsLeft[i], right)
				}
			}
		}

		msgsConnAckLeft := []ConnAck{{Header: Header{msgType: MsgTypeConnAck}, SessionPresentFlag: true, ReturnCode: RetCodeAccepted},
			{Header: Header{msgType: MsgTypeConnAck}, SessionPresentFlag: false, ReturnCode: RetCodeBadUsernameOrPassword},
			{Header: Header{msgType: MsgTypeConnAck}, SessionPresentFlag: true, ReturnCode: RetCodeIdentifierRejected},
			{Header: Header{msgType: MsgTypeConnAck}, SessionPresentFlag: false, ReturnCode: RetCodeNotAuthorized},
			{Header: Header{msgType: MsgTypeConnAck}, SessionPresentFlag: true, ReturnCode: RetCodeUnacceptableProtocolVersion},
		}

		for i := 0; i < len(msgsConnAckLeft); i++ {
			//msgLeft := msgsLeft[i]
			bytes, err := msgsConnAckLeft[i].Encode()
			convey.So(err, convey.ShouldBeNil)
			msgsConnAckLeft[i].remainingLength = 0

			_, err = buf.Write(bytes)
			convey.So(err, convey.ShouldBeNil)

			msgsRight, err := codec.Decode(buf)
			convey.So(err, convey.ShouldBeNil)

			if right, ok := msgsRight[0].(*ConnAck); ok {
				if msgsConnAckLeft[i] != *right {
					t.Errorf("left:%v right:%v\n", msgsConnAckLeft[i], right)
				}
			}
		}

		msgsDisconnectLeft := []Disconnect{{Header: Header{msgType: MsgTypeDisconnect}},
			{Header: Header{msgType: MsgTypeDisconnect}},
			{Header: Header{msgType: MsgTypeDisconnect}},
			{Header: Header{msgType: MsgTypeDisconnect}},
			{Header: Header{msgType: MsgTypeDisconnect}},
		}

		for i := 0; i < len(msgsDisconnectLeft); i++ {
			//msgLeft := msgsLeft[i]
			bytes, err := msgsDisconnectLeft[i].Encode()
			convey.So(err, convey.ShouldBeNil)
			msgsDisconnectLeft[i].remainingLength = 0

			_, err = buf.Write(bytes)
			convey.So(err, convey.ShouldBeNil)

			msgsRight, err := codec.Decode(buf)
			convey.So(err, convey.ShouldBeNil)

			if right, ok := msgsRight[0].(*Disconnect); ok {
				if msgsDisconnectLeft[i] != *right {
					t.Errorf("left:%v right:%v\n", msgsDisconnectLeft[i], right)
				}
			}
		}

		msgsPingReqLeft := []PingReq{{Header: Header{msgType: MsgTypePingReq}},
			{Header: Header{msgType: MsgTypePingReq}},
			{Header: Header{msgType: MsgTypePingReq}},
			{Header: Header{msgType: MsgTypePingReq}},
			{Header: Header{msgType: MsgTypePingReq}},
		}

		for i := 0; i < len(msgsPingReqLeft); i++ {
			//msgLeft := msgsLeft[i]
			bytes, err := msgsPingReqLeft[i].Encode()
			convey.So(err, convey.ShouldBeNil)
			msgsPingReqLeft[i].remainingLength = 0

			_, err = buf.Write(bytes)
			convey.So(err, convey.ShouldBeNil)

			msgsRight, err := codec.Decode(buf)
			convey.So(err, convey.ShouldBeNil)

			if right, ok := msgsRight[0].(*PingReq); ok {
				if msgsPingReqLeft[i] != *right {
					t.Errorf("left:%v right:%v\n", msgsPingReqLeft[i], right)
				}
			}
		}

		msgsPingRespLeft := []PingResp{{Header: Header{msgType: MsgTypePingResp}},
			{Header: Header{msgType: MsgTypePingResp}},
			{Header: Header{msgType: MsgTypePingResp}},
			{Header: Header{msgType: MsgTypePingResp}},
			{Header: Header{msgType: MsgTypePingResp}},
		}

		for i := 0; i < len(msgsPingRespLeft); i++ {
			//msgLeft := msgsLeft[i]
			bytes, err := msgsPingRespLeft[i].Encode()
			convey.So(err, convey.ShouldBeNil)
			msgsPingRespLeft[i].remainingLength = 0

			_, err = buf.Write(bytes)
			convey.So(err, convey.ShouldBeNil)

			msgsRight, err := codec.Decode(buf)
			convey.So(err, convey.ShouldBeNil)

			if right, ok := msgsRight[0].(*PingResp); ok {
				if msgsPingRespLeft[i] != *right {
					t.Errorf("left:%v right:%v\n", msgsPingRespLeft[i], right)
				}
			}
		}

		msgsPubAckLeft := []PubAck{{Header: Header{msgType: MsgTypePubAck}},
			{Header: Header{msgType: MsgTypePubAck}},
			{Header: Header{msgType: MsgTypePubAck}},
			{Header: Header{msgType: MsgTypePubAck}},
			{Header: Header{msgType: MsgTypePubAck}},
		}

		for i := 0; i < len(msgsPubAckLeft); i++ {
			//msgLeft := msgsLeft[i]
			bytes, err := msgsPubAckLeft[i].Encode()
			convey.So(err, convey.ShouldBeNil)
			msgsPubAckLeft[i].remainingLength = 0

			_, err = buf.Write(bytes)
			convey.So(err, convey.ShouldBeNil)

			msgsRight, err := codec.Decode(buf)
			convey.So(err, convey.ShouldBeNil)

			if right, ok := msgsRight[0].(*PubAck); ok {
				if msgsPubAckLeft[i] != *right {
					t.Errorf("left:%v right:%v\n", msgsPubAckLeft[i], right)
				}
			}
		}

		msgsPubCompLeft := []PubComp{{Header: Header{msgType: MsgTypePubComp}},
			{Header: Header{msgType: MsgTypePubComp}},
			{Header: Header{msgType: MsgTypePubComp}},
			{Header: Header{msgType: MsgTypePubComp}},
			{Header: Header{msgType: MsgTypePubComp}},
		}

		for i := 0; i < len(msgsPubCompLeft); i++ {
			//msgLeft := msgsLeft[i]
			bytes, err := msgsPubCompLeft[i].Encode()
			convey.So(err, convey.ShouldBeNil)
			msgsPubCompLeft[i].remainingLength = 0

			_, err = buf.Write(bytes)
			convey.So(err, convey.ShouldBeNil)

			msgsRight, err := codec.Decode(buf)
			convey.So(err, convey.ShouldBeNil)

			if right, ok := msgsRight[0].(*PubComp); ok {
				if msgsPubCompLeft[i] != *right {
					t.Errorf("left:%v right:%v\n", msgsPubCompLeft[i], right)
				}
			}
		}

		msgsPublishLeft := []Publish{{Header: Header{msgType: MsgTypePublish}, Dup: true, Qos: QosAtMostOnce, Retain: true, TopicName: "a", PacketIdentifier: 0, Payload: []byte("hehe")},
			{Header: Header{msgType: MsgTypePublish}, Dup: true, Qos: QosAtLeastOnce, Retain: true, TopicName: "arew", PacketIdentifier: 65535, Payload: []byte("hsdgfdehe")},
			{Header: Header{msgType: MsgTypePublish}, Dup: false, Qos: QosAtMostOnce, Retain: false, TopicName: "b", PacketIdentifier: 0, Payload: []byte("hehegdf")},
			{Header: Header{msgType: MsgTypePublish}, Dup: false, Qos: QosAtMostOnce, Retain: true, TopicName: "awefw", PacketIdentifier: 0, Payload: []byte("hehasdfsde")},
			{Header: Header{msgType: MsgTypePublish}},
		}

		for i := 0; i < len(msgsPublishLeft); i++ {
			//msgLeft := msgsLeft[i]
			bytes, err := msgsPublishLeft[i].Encode()
			convey.So(err, convey.ShouldBeNil)
			msgsPublishLeft[i].remainingLength = 0

			_, err = buf.Write(bytes)
			convey.So(err, convey.ShouldBeNil)

			msgsRight, err := codec.Decode(buf)
			convey.So(err, convey.ShouldBeNil)

			if right, ok := msgsRight[0].(*Publish); ok {
				if !reflect.DeepEqual(msgsPublishLeft[i], *right) {
					t.Errorf("left:%v right:%v\n", msgsPublishLeft[i], right)
				}
			}
		}

		msgsPubRecLeft := []PubRec{{Header: Header{msgType: MsgTypePubRec}},
			{Header: Header{msgType: MsgTypePubRec}},
			{Header: Header{msgType: MsgTypePubRec}},
			{Header: Header{msgType: MsgTypePubRec}},
			{Header: Header{msgType: MsgTypePubRec}},
		}

		for i := 0; i < len(msgsPubRecLeft); i++ {
			//msgLeft := msgsLeft[i]
			bytes, err := msgsPubRecLeft[i].Encode()
			convey.So(err, convey.ShouldBeNil)
			msgsPubRecLeft[i].remainingLength = 0

			_, err = buf.Write(bytes)
			convey.So(err, convey.ShouldBeNil)

			msgsRight, err := codec.Decode(buf)
			convey.So(err, convey.ShouldBeNil)

			if right, ok := msgsRight[0].(*PubRec); ok {
				if msgsPubRecLeft[i] != *right {
					t.Errorf("left:%v right:%v\n", msgsPubRecLeft[i], right)
				}
			}
		}

		msgsPubRelLeft := []PubRel{{Header: Header{msgType: MsgTypePubRel}},
			{Header: Header{msgType: MsgTypePubRel}},
			{Header: Header{msgType: MsgTypePubRel}},
			{Header: Header{msgType: MsgTypePubRel}},
			{Header: Header{msgType: MsgTypePubRel}},
		}

		for i := 0; i < len(msgsPubRelLeft); i++ {
			//msgLeft := msgsLeft[i]
			bytes, err := msgsPubRelLeft[i].Encode()
			convey.So(err, convey.ShouldBeNil)
			msgsPubRelLeft[i].remainingLength = 0

			_, err = buf.Write(bytes)
			convey.So(err, convey.ShouldBeNil)

			msgsRight, err := codec.Decode(buf)
			convey.So(err, convey.ShouldBeNil)

			if right, ok := msgsRight[0].(*PubRel); ok {
				if msgsPubRelLeft[i] != *right {
					t.Errorf("left:%v right:%v\n", msgsPubRelLeft[i], right)
				}
			}
		}

		msgsSubAckLeft := []SubAck{{Header: Header{msgType: MsgTypeSubAck}},
			{Header: Header{msgType: MsgTypeSubAck}},
			{Header: Header{msgType: MsgTypeSubAck}},
			{Header: Header{msgType: MsgTypeSubAck}},
			{Header: Header{msgType: MsgTypeSubAck}},
		}

		for i := 0; i < len(msgsSubAckLeft); i++ {
			//msgLeft := msgsLeft[i]
			bytes, err := msgsSubAckLeft[i].Encode()
			convey.So(err, convey.ShouldBeNil)
			msgsSubAckLeft[i].remainingLength = 0

			_, err = buf.Write(bytes)
			convey.So(err, convey.ShouldBeNil)

			msgsRight, err := codec.Decode(buf)
			convey.So(err, convey.ShouldBeNil)

			if right, ok := msgsRight[0].(*SubAck); ok {
				if reflect.DeepEqual(msgsPublishLeft[i], *right) {
					t.Errorf("left:%v right:%v\n", msgsSubAckLeft[i], right)
				}
			}
		}

		msgsSubscribeLeft := []Subscribe{{Header: Header{msgType: MsgTypeSubscribe}},
			{Header: Header{msgType: MsgTypeSubscribe}},
			{Header: Header{msgType: MsgTypeSubscribe}},
			{Header: Header{msgType: MsgTypeSubscribe}},
			{Header: Header{msgType: MsgTypeSubscribe}},
		}

		for i := 0; i < len(msgsSubscribeLeft); i++ {
			//msgLeft := msgsLeft[i]
			bytes, err := msgsSubscribeLeft[i].Encode()
			convey.So(err, convey.ShouldBeNil)
			msgsSubscribeLeft[i].remainingLength = 0

			_, err = buf.Write(bytes)
			convey.So(err, convey.ShouldBeNil)

			msgsRight, err := codec.Decode(buf)
			convey.So(err, convey.ShouldBeNil)

			if right, ok := msgsRight[0].(*Subscribe); ok {
				if reflect.DeepEqual(msgsPublishLeft[i], *right) {
					t.Errorf("left:%v right:%v\n", msgsSubscribeLeft[i], right)
				}
			}
		}

		msgsUnsubscribeLeft := []Unsubscribe{{Header: Header{msgType: MsgTypeUnsubscribe}},
			{Header: Header{msgType: MsgTypeUnsubscribe}},
			{Header: Header{msgType: MsgTypeUnsubscribe}},
			{Header: Header{msgType: MsgTypeUnsubscribe}},
			{Header: Header{msgType: MsgTypeUnsubscribe}},
		}

		for i := 0; i < len(msgsUnsubscribeLeft); i++ {
			//msgLeft := msgsLeft[i]
			bytes, err := msgsUnsubscribeLeft[i].Encode()
			convey.So(err, convey.ShouldBeNil)
			msgsUnsubscribeLeft[i].remainingLength = 0

			_, err = buf.Write(bytes)
			convey.So(err, convey.ShouldBeNil)

			msgsRight, err := codec.Decode(buf)
			convey.So(err, convey.ShouldBeNil)

			if right, ok := msgsRight[0].(*Unsubscribe); ok {
				if reflect.DeepEqual(msgsPublishLeft[i], *right) {
					t.Errorf("left:%v right:%v\n", msgsUnsubscribeLeft[i], right)
				}
			}
		}

		msgsUnsubAckLeft := []UnsubAck{{Header: Header{msgType: MsgTypeUnsubAck}},
			{Header: Header{msgType: MsgTypeUnsubAck}},
			{Header: Header{msgType: MsgTypeUnsubAck}},
			{Header: Header{msgType: MsgTypeUnsubAck}},
			{Header: Header{msgType: MsgTypeUnsubAck}},
		}

		for i := 0; i < len(msgsUnsubAckLeft); i++ {
			//msgLeft := msgsLeft[i]
			bytes, err := msgsUnsubAckLeft[i].Encode()
			convey.So(err, convey.ShouldBeNil)
			msgsUnsubAckLeft[i].remainingLength = 0

			_, err = buf.Write(bytes)
			convey.So(err, convey.ShouldBeNil)

			msgsRight, err := codec.Decode(buf)
			convey.So(err, convey.ShouldBeNil)

			if right, ok := msgsRight[0].(*UnsubAck); ok {
				if msgsUnsubAckLeft[i] != *right {
					t.Errorf("left:%v right:%v\n", msgsUnsubAckLeft[i], right)
				}
			}
		}
	})
}
