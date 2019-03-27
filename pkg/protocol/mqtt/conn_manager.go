//Created by zhbinary on 2018/11/10.
//Email: zhbinary@gmail.com
package mqtt

type ConnectionManager interface {
	PutConnection(id interface{}, conn *Connection)
	GetConnection(id interface{}) *Connection
	GetConnectionCount() int
	DeleteConnection(id interface{})
	DeleteAllConnection()
	CloseConnection(id interface{})
	CloseAllConnection()
}
