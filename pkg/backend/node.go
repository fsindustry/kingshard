// Copyright 2016 The kingshard Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package backend

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsindustry/kingshard/pkg/core/errors"
	"github.com/fsindustry/kingshard/pkg/mysql"
)

const (
	Up = iota
	Down
	ManualDown
	Unknown

	InitConnCount           = 16
	DefaultMaxConnNum       = 1024
	PingPeroid        int64 = 4
)

type Node struct {
	sync.RWMutex

	addr     string
	user     string
	password string
	db       string
	state    int32

	maxConnNum  int
	InitConnNum int
	idleConns   chan *Conn
	cacheConns  chan *Conn
	checkConn   *Conn
	lastPing    int64

	pushConnCount int64
	popConnCount  int64
}

func Open(addr string, user string, password string, dbName string, maxConnNum int) (*Node, error) {
	var err error
	node := new(Node)
	node.addr = addr
	node.user = user
	node.password = password
	node.db = dbName

	if 0 < maxConnNum {
		node.maxConnNum = maxConnNum
		if node.maxConnNum < 16 {
			node.InitConnNum = node.maxConnNum
		} else {
			node.InitConnNum = node.maxConnNum / 4
		}
	} else {
		node.maxConnNum = DefaultMaxConnNum
		node.InitConnNum = InitConnCount
	}
	//check connection
	node.checkConn, err = node.newConn()
	if err != nil {
		node.Close()
		return nil, err
	}

	node.idleConns = make(chan *Conn, node.maxConnNum)
	node.cacheConns = make(chan *Conn, node.maxConnNum)
	atomic.StoreInt32(&(node.state), Unknown)

	for i := 0; i < node.maxConnNum; i++ {
		if i < node.InitConnNum {
			conn, err := node.newConn()

			if err != nil {
				node.Close()
				return nil, err
			}

			node.cacheConns <- conn
			atomic.AddInt64(&node.pushConnCount, 1)
		} else {
			conn := new(Conn)
			node.idleConns <- conn
			atomic.AddInt64(&node.pushConnCount, 1)
		}
	}
	node.SetLastPing()

	return node, nil
}

func (node *Node) Addr() string {
	return node.addr
}

func (node *Node) State() string {
	var state string
	switch node.state {
	case Up:
		state = "up"
	case Down, ManualDown:
		state = "down"
	case Unknown:
		state = "unknow"
	}
	return state
}

func (node *Node) ConnCount() (int, int, int64, int64) {
	node.RLock()
	defer node.RUnlock()
	return len(node.idleConns), len(node.cacheConns), node.pushConnCount, node.popConnCount
}

func (node *Node) Close() error {
	node.Lock()
	idleChannel := node.idleConns
	cacheChannel := node.cacheConns
	node.cacheConns = nil
	node.idleConns = nil
	node.Unlock()
	if cacheChannel == nil || idleChannel == nil {
		return nil
	}

	close(cacheChannel)
	for conn := range cacheChannel {
		node.closeConn(conn)
	}
	close(idleChannel)

	return nil
}

func (node *Node) getConns() (chan *Conn, chan *Conn) {
	node.RLock()
	cacheConns := node.cacheConns
	idleConns := node.idleConns
	node.RUnlock()
	return cacheConns, idleConns
}

func (node *Node) getCacheConns() chan *Conn {
	node.RLock()
	conns := node.cacheConns
	node.RUnlock()
	return conns
}

func (node *Node) getIdleConns() chan *Conn {
	node.RLock()
	conns := node.idleConns
	node.RUnlock()
	return conns
}

func (node *Node) Ping() error {
	var err error
	if node.checkConn == nil {
		node.checkConn, err = node.newConn()
		if err != nil {
			if node.checkConn != nil {
				node.checkConn.Close()
				node.checkConn = nil
			}
			return err
		}
	}
	err = node.checkConn.Ping()
	if err != nil {
		if node.checkConn != nil {
			node.checkConn.Close()
			node.checkConn = nil
		}
		return err
	}
	return nil
}

func (node *Node) newConn() (*Conn, error) {
	co := new(Conn)

	if err := co.Connect(node.addr, node.user, node.password, node.db); err != nil {
		return nil, err
	}

	co.pushTimestamp = time.Now().Unix()

	return co, nil
}

func (node *Node) addIdleConn() {
	conn := new(Conn)
	select {
	case node.idleConns <- conn:
	default:
		break
	}
}

func (node *Node) closeConn(co *Conn) error {
	atomic.AddInt64(&node.pushConnCount, 1)

	if co != nil {
		co.Close()
		conns := node.getIdleConns()
		if conns != nil {
			select {
			case conns <- co:
				return nil
			default:
				return nil
			}
		}
	} else {
		node.addIdleConn()
	}
	return nil
}

func (node *Node) closeConnNotAdd(co *Conn) error {
	if co != nil {
		co.Close()
		conns := node.getIdleConns()
		if conns != nil {
			select {
			case conns <- co:
				return nil
			default:
				return nil
			}
		}
	} else {
		node.addIdleConn()
	}
	return nil
}

func (node *Node) tryReuse(co *Conn) error {
	var err error
	//reuse Connection
	if co.IsInTransaction() {
		//we can not reuse a connection in transaction status
		err = co.Rollback()
		if err != nil {
			return err
		}
	}

	if !co.IsAutoCommit() {
		//we can not  reuse a connection not in autocomit
		_, err = co.exec("set autocommit = 1")
		if err != nil {
			return err
		}
	}

	//connection may be set names early
	//we must use default utf8
	if co.GetCharset() != mysql.DEFAULT_CHARSET {
		err = co.SetCharset(mysql.DEFAULT_CHARSET, mysql.DEFAULT_COLLATION_ID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (node *Node) PopConn() (*Conn, error) {
	var co *Conn
	var err error

	cacheConns, idleConns := node.getConns()
	if cacheConns == nil || idleConns == nil {
		return nil, errors.ErrDatabaseClose
	}
	co = node.GetConnFromCache(cacheConns)
	if co == nil {
		co, err = node.GetConnFromIdle(cacheConns, idleConns)
		if err != nil {
			return nil, err
		}
	}

	err = node.tryReuse(co)
	if err != nil {
		node.closeConn(co)
		return nil, err
	}

	return co, nil
}

func (node *Node) GetConnFromCache(cacheConns chan *Conn) *Conn {
	var co *Conn
	var err error
	for 0 < len(cacheConns) {
		co = <-cacheConns
		atomic.AddInt64(&node.popConnCount, 1)
		if co != nil && PingPeroid < time.Now().Unix()-co.pushTimestamp {
			err = co.Ping()
			if err != nil {
				node.closeConn(co)
				co = nil
			}
		}
		if co != nil {
			break
		}
	}
	return co
}

func (node *Node) GetConnFromIdle(cacheConns, idleConns chan *Conn) (*Conn, error) {
	var co *Conn
	var err error
	select {
	case co = <-idleConns:
		atomic.AddInt64(&node.popConnCount, 1)
		co, err := node.newConn()
		if err != nil {
			node.closeConn(co)
			return nil, err
		}
		err = co.Ping()
		if err != nil {
			node.closeConn(co)
			return nil, errors.ErrBadConn
		}
		return co, nil
	case co = <-cacheConns:
		atomic.AddInt64(&node.popConnCount, 1)
		if co == nil {
			return nil, errors.ErrConnIsNil
		}
		if co != nil && PingPeroid < time.Now().Unix()-co.pushTimestamp {
			err = co.Ping()
			if err != nil {
				node.closeConn(co)
				return nil, errors.ErrBadConn
			}
		}
	}
	return co, nil
}

func (node *Node) PushConn(co *Conn, err error) {
	atomic.AddInt64(&node.pushConnCount, 1)
	if co == nil {
		node.addIdleConn()
		return
	}
	conns := node.getCacheConns()
	if conns == nil {
		co.Close()
		return
	}
	if err != nil {
		node.closeConnNotAdd(co)
		return
	}
	co.pushTimestamp = time.Now().Unix()
	select {
	case conns <- co:
		return
	default:
		node.closeConnNotAdd(co)
		return
	}
}

type BackendConn struct {
	*Conn
	node *Node
}

func (p *BackendConn) Close() {
	if p != nil && p.Conn != nil {
		if p.Conn.pkgErr != nil {
			p.node.closeConn(p.Conn)
		} else {
			p.node.PushConn(p.Conn, nil)
		}
		p.Conn = nil
	}
}

func (node *Node) GetConn() (*BackendConn, error) {
	c, err := node.PopConn()
	if err != nil {
		return nil, err
	}
	return &BackendConn{c, node}, nil
}

func (node *Node) SetLastPing() {
	node.lastPing = time.Now().Unix()
}

func (node *Node) GetLastPing() int64 {
	return node.lastPing
}
