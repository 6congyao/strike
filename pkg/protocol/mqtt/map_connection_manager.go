//Created by zhbinary on 2018/11/13.
//Email: zhbinary@gmail.com
package mqtt

import "sync"

type MapConnectionManager struct {
	mutex *sync.RWMutex
	m     map[interface{}]*Connection
}

func NewMapConnectionManager() ConnectionManager {
	return &MapConnectionManager{m: make(map[interface{}]*Connection), mutex: &sync.RWMutex{}}
}

func (this *MapConnectionManager) PutConnection(id interface{}, Connection *Connection) {
	if id == "" || Connection == nil {
		return
	}
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.m[id] = Connection
}

func (this *MapConnectionManager) GetConnection(id interface{}) *Connection {
	if id == "" {
		return nil
	}
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return this.m[id]
}

func (this *MapConnectionManager) GetConnectionCount() int {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	return len(this.m)
}

func (this *MapConnectionManager) DeleteConnection(id interface{}) {
	if id == "" {
		return
	}
	this.mutex.Lock()
	defer this.mutex.Unlock()
	delete(this.m, id)
}

func (this *MapConnectionManager) DeleteAllConnection() {
	this.mutex.Lock()
	defer this.mutex.Unlock()
	this.m = nil
	this.m = make(map[interface{}]*Connection)
}

func (this *MapConnectionManager) CloseConnection(id interface{}) {
	if id == "" {
		return
	}
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	if this.m[id] != nil {
		this.m[id].Close()
	}
}

func (this *MapConnectionManager) CloseAllConnection() {
	this.mutex.RLock()
	defer this.mutex.RUnlock()
	for _, Connection := range this.m {
		Connection.Close()
	}
}
