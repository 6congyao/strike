//Created by zhbinary on 2018/12/17.
//Email: zhbinary@gmail.com
package mqtt

import (
	//"qingcloud.com/gateway/mqtt/message"
	//"qingcloud.com/qing-cloud-mq/common"
	//"qingcloud.com/qing-cloud-mq/common/proto"
	//"qingcloud.com/qing-cloud-mq/consumer"
	//"qingcloud.com/qing-cloud-mq/network"
	//"qingcloud.com/qing-cloud-mq/producer"
	//"time"
	"context"
	"fmt"
	"strike/pkg/network"
	"strike/pkg/protocol/mqtt/message"
	"strike/pkg/types"
)

const (
	TopicMqtt = "TopicMqtt"
)

type Processor struct {
	connManager ConnectionManager
	//producer     *producer.Producer
	//retryHandler *network.RetryHandler
}

//TODO: singleton
func NewProcessor() *Processor {
	// Create producer
	//opts := producer.NewOptions()
	//opts.NsServerList = "127.0.0.1:2379"
	//opts.BatchSize = 10
	//opts.BatchInterval = 10 * time.Microsecond
	//producer, err := producer.NewProducer(opts)
	//if err != nil {
	//	panic(err)
	//}
	//
	//retryHandler := network.NewRetryHandler(-1, 3*time.Second)
	//
	//p := &Processor{producer: producer, retryHandler: retryHandler}
	//return p
	return &Processor{}
}

func (this *Processor) Process(context context.Context, m message.Message) {
	if m == nil {
		return
	}
	// got network.connection from context
	conn, _ := context.Value(types.ContextKeyConnectionRef).(network.Connection)

	var resp message.Message
	switch msg := m.(type) {
	case *message.Connect:
		// Check fields base on protocol

		// Check client id or fix it

		// Authentication

		// Retrieve conn or create
		//conn := this.connManager.GetConnection(msg.ClientId)
		//if conn == nil {
		//	conn = &Connection{}
		//	this.connManager.PutConnection(msg.ClientId, conn)
		//}

		fmt.Println("got msg:", msg)
		// Send ack to client
		ack := message.NewConnAck()
		ack.ReturnCode = message.RetCodeAccepted
		resp = ack
		break
	case *message.ConnAck:
		break
	//case *message.Publish:
	//	if msg.Dup {
	//		// This is a resent message
	//	}
	//
	//	if msg.Retain {
	//		// Store it
	//	} else {
	//		// Send it immediately
	//	}
	//
	//	switch msg.Qos {
	//	case message.QosAtMostOnce:
	//		mqMsg := proto.NewMessage()
	//		mqMsg.Headers.Destination = TopicMqtt
	//		mqMsg.Headers.MessageKey = msg.TopicName
	//		mqMsg.Data = msg.Payload
	//		this.producer.SendOneWay(mqMsg)
	//		break
	//	case message.QosAtLeastOnce:
	//		mqMsg := proto.NewMessage()
	//		mqMsg.Headers.Destination = TopicMqtt
	//		mqMsg.Headers.MessageKey = msg.TopicName
	//		mqMsg.Data = msg.Payload
	//
	//		result, err := this.producer.Send(mqMsg)
	//		if err != nil || result.Code != common.CodeOk {
	//			this.retryHandler.Retry(func(count int) bool {
	//				result, err := this.producer.Send(mqMsg)
	//				if err != nil || result.Code != common.CodeOk {
	//					return false
	//				}
	//				return true
	//			})
	//		}
	//
	//		// Send ack
	//		ack := message.NewPubAck()
	//		ack.PacketIdentifier = msg.PacketIdentifier
	//		break
	//	case message.QosExactlyOnce:
	//		// todo
	//		break
	//	}
	//	break
	//case *message.PubAck:
	//	break
	//case *message.PubRec:
	//	break
	//case *message.PubRel:
	//	break
	//case *message.PubComp:
	//	break
	//case *message.Subscribe:
	//	// Create consumer per client
	//	opt := consumer.NewOptions()
	//	opt.GroupID = "group1"
	//	opt.EtcdEndpoints = "127.0.0.1:2379"
	//	opt.OffsetMode = consumer.CONSUME_FROM_HEAD
	//	opt.MsgCnt = 10
	//	c, err := consumer.NewConsumer(opt)
	//	if err != nil {
	//		panic(err)
	//	}
	//	c.Start()
	//
	//	for _, topicFilter := range msg.TopicFilters {
	//		topic := topicFilter.TopicName
	//		c.BindKeyWithListener(TopicMqtt, topic, &ReceiveCallback{})
	//	}
	//	// sub 3,4
	//	//this.consumer.BindKeyWithListener(TopicMqtt, []topic, &ReceiveCallback{})
	//	//this.consumer.BindQueue(TopicMqtt, msg.TopicFilters[0].TopicName)
	//	break
	//case *message.SubAck:
	//	break
	//case *message.Unsubscribe:
	//	//keys := msg.TopicNames // unsub 3,4
	//	//this.consumer.UnbindQueue(keys)
	//	break
	//case *message.UnsubAck:
	//	break
	case *message.PingReq:
		// Send pong to client, close conn if did not receive ping 3 times
		resp = message.NewPingResp()
		break
	case *message.PingResp:
		break
	//case *message.Disconnect:
	//	// Close conn and clean resources
	//
	//	break
	default:
		break
	}

	if resp != nil {
		buf, err := resp.Encode()
		if err != nil {
			fmt.Println(err)
			return
		}
		conn.Write(buf)
	}
}

type ReceiveCallback struct {
	ch chan []byte
	// rawConn  write(msg)
}

//func (t *ReceiveCallback) OnReceived(msg *proto.Message, ctx consumer.Context) {
//	t.ch <- msg.Data
//	// this.rawConn.write(msg)
//	ctx.Ack()
//}
