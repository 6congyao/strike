//Created by zhbinary on 2018-12-25.
//Email: zhbinary@gmail.com
package mqtt

import "strike/pkg/protocol/mqtt/message"

type Connection struct {
	// raw conn
	ProtocolName  string
	ProtocolLevel uint8
	UserNameFlag  bool
	passwordFlag  bool
	WillRetain    bool
	WillQos       message.Qos
	WillFlag      bool
	CleanSession  bool
	KeepAlive     uint16

	ClientId    string
	WillTopic   string
	WillMessage string
	Username    string
	Password    string
}

func NewConnecion() *Connection {
	return &Connection{}
}

func (this *Connection) Close() {

}
