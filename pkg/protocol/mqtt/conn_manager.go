//Created by zhbinary on 2018/11/10.
//Email: zhbinary@gmail.com
package mqtt

type ConnectionManager interface {
	PutConnection(id string, conn *Connection)
	GetConnection(id string) *Connection
	GetConnectionCount() int
	DeleteConnection(id string)
	DeleteAllConnection()
	CloseConnection(id string)
	CloseAllConnection()
}
