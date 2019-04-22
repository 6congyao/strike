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
	"strike/pkg/protocol/mqtt/message"
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
	return
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
