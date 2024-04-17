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

	"github.com/fsindustry/kingshard/pkg/config"
	"github.com/fsindustry/kingshard/pkg/core/errors"
	"github.com/fsindustry/kingshard/pkg/core/golog"
)

const (
	Master      = "master"
	Slave       = "slave"
	SlaveSplit  = ","
	WeightSplit = "@"
)

type Group struct {
	Cfg config.GroupConfig

	sync.RWMutex
	Master *Node

	Slave          []*Node
	LastSlaveIndex int
	RoundRobinQ    []int
	SlaveWeights   []int

	DownAfterNoAlive time.Duration

	Online bool
}

func (g *Group) CheckNode() {
	//to do
	//1 check connection alive
	for g.Online {
		g.checkMaster()
		g.checkSlave()
		time.Sleep(16 * time.Second)
	}
}

func (g *Group) String() string {
	return g.Cfg.Name
}

func (g *Group) GetMasterConn() (*BackendConn, error) {
	db := g.Master
	if db == nil {
		return nil, errors.ErrNoMasterConn
	}
	if atomic.LoadInt32(&(db.state)) == Down {
		return nil, errors.ErrMasterDown
	}

	return db.GetConn()
}

func (g *Group) GetSlaveConn() (*BackendConn, error) {
	g.Lock()
	db, err := g.GetNextSlave()
	g.Unlock()
	if err != nil {
		return nil, err
	}

	if db == nil {
		return nil, errors.ErrNoSlaveDB
	}
	if atomic.LoadInt32(&(db.state)) == Down {
		return nil, errors.ErrSlaveDown
	}

	return db.GetConn()
}

func (g *Group) checkMaster() {
	master := g.Master
	if master == nil {
		golog.Error("Group", "checkMaster", "Master is no alive", 0)
		return
	}

	if err := master.Ping(); err != nil {
		golog.Error("Group", "checkMaster", "Ping", 0, "master.Addr", master.Addr(), "error", err.Error())
	} else {
		if atomic.LoadInt32(&(master.state)) == Down {
			golog.Info("Group", "checkMaster", "Master up", 0, "master.Addr", master.Addr())
			err := g.UpMaster(master.ExtractNodeConfig())
			if err != nil {
				golog.Error("Group", "checkMaster", "UpMaster", 0, "master.Addr", master.Addr(), "error", err.Error())
				return
			}
		}
		master.SetLastPing()
		if atomic.LoadInt32(&(master.state)) != ManualDown {
			atomic.StoreInt32(&(master.state), Up)
		}
		return
	}

	if int64(g.DownAfterNoAlive) > 0 && time.Now().Unix()-master.GetLastPing() > int64(g.DownAfterNoAlive/time.Second) {
		golog.Info("Group", "checkMaster", "Master down", 0,
			"master.Addr", master.Addr(),
			"Master_down_time", int64(g.DownAfterNoAlive/time.Second))
		g.DownMaster(master.addr, Down)
	}
}

func (g *Group) checkSlave() {
	g.RLock()
	if g.Slave == nil {
		g.RUnlock()
		return
	}
	slaves := make([]*Node, len(g.Slave))
	copy(slaves, g.Slave)
	g.RUnlock()

	for i := 0; i < len(slaves); i++ {
		if err := slaves[i].Ping(); err != nil {
			golog.Error("Group", "checkSlave", "Ping", 0, "node.Addr", slaves[i].Addr(), "error", err.Error())
		} else {
			if atomic.LoadInt32(&(slaves[i].state)) == Down {
				golog.Info("Group", "checkSlave", "Slave up", 0, "node.Addr", slaves[i].Addr())
				g.UpSlave(slaves[i].ExtractNodeConfig())
			}
			slaves[i].SetLastPing()
			if atomic.LoadInt32(&(slaves[i].state)) != ManualDown {
				atomic.StoreInt32(&(slaves[i].state), Up)
			}
			continue
		}

		if int64(g.DownAfterNoAlive) > 0 && time.Now().Unix()-slaves[i].GetLastPing() > int64(g.DownAfterNoAlive/time.Second) {
			golog.Info("Group", "checkSlave", "Slave down", 0,
				"node.Addr", slaves[i].Addr(),
				"slave_down_time", int64(g.DownAfterNoAlive/time.Second))
			//If can't ping slave after DownAfterNoAlive, set slave Down
			g.DownSlave(slaves[i].addr, Down)
		}
	}

}

func (g *Group) AddSlave(nodeConfig *config.NodeConfig) error {
	var node *Node
	var err error
	if nodeConfig == nil || len(nodeConfig.Addr) == 0 {
		return errors.ErrAddressNull
	}
	g.Lock()
	defer g.Unlock()
	for _, v := range g.Slave {
		if v.addr == nodeConfig.Addr {
			return errors.ErrSlaveExist
		}
	}

	g.SlaveWeights = append(g.SlaveWeights, nodeConfig.Weight)
	if node, err = g.OpenNode(nodeConfig); err != nil {
		return err
	} else {
		g.Slave = append(g.Slave, node)
		g.InitBalancer()
		return nil
	}
}

func (g *Group) DeleteSlave(addr string) error {
	var i int
	g.Lock()
	defer g.Unlock()
	slaveCount := len(g.Slave)
	if slaveCount == 0 {
		return errors.ErrNoSlaveDB
	}
	for i = 0; i < slaveCount; i++ {
		if g.Slave[i].addr == addr {
			break
		}
	}
	if i == slaveCount {
		return errors.ErrSlaveNotExist
	}
	if slaveCount == 1 {
		g.Slave = nil
		g.SlaveWeights = nil
		g.RoundRobinQ = nil
		return nil
	}

	s := make([]*Node, 0, slaveCount-1)
	sw := make([]int, 0, slaveCount-1)
	for i = 0; i < slaveCount; i++ {
		if g.Slave[i].addr != addr {
			s = append(s, g.Slave[i])
			sw = append(sw, g.SlaveWeights[i])
		}
	}

	g.Slave = s
	g.SlaveWeights = sw
	g.InitBalancer()
	return nil
}

func (g *Group) OpenNode(nodeConfig *config.NodeConfig) (*Node, error) {
	node, err := Open(nodeConfig.Addr, nodeConfig.User, nodeConfig.Password, "", nodeConfig.MaxConnNum)
	return node, err
}

func (g *Group) UpNode(nodeConfig *config.NodeConfig) (*Node, error) {
	node, err := g.OpenNode(nodeConfig)

	if err != nil {
		return nil, err
	}

	if err := node.Ping(); err != nil {
		node.Close()
		atomic.StoreInt32(&(node.state), Down)
		return nil, err
	}
	atomic.StoreInt32(&(node.state), Up)
	return node, nil
}

func (g *Group) UpMaster(nodeConfig *config.NodeConfig) error {
	node, err := g.UpNode(nodeConfig)
	if err != nil {
		golog.Error("Group", "UpMaster", err.Error(), 0)
		return err
	}
	g.Master = node
	return err
}

func (g *Group) UpSlave(nodeConfig *config.NodeConfig) error {
	node, err := g.UpNode(nodeConfig)
	if err != nil {
		golog.Error("Group", "UpSlave", err.Error(), 0)
	}

	g.Lock()
	for k, slave := range g.Slave {
		if slave.addr == nodeConfig.Addr {
			g.Slave[k] = node
			g.Unlock()
			return nil
		}
	}
	g.Slave = append(g.Slave, node)
	g.Unlock()

	return err
}

func (g *Group) DownMaster(addr string, state int32) error {
	master := g.Master
	if master == nil || master.addr != addr {
		return errors.ErrNoMasterDB
	}

	master.Close()
	atomic.StoreInt32(&(master.state), state)
	return nil
}

func (g *Group) DownSlave(addr string, state int32) error {
	g.RLock()
	if g.Slave == nil {
		g.RUnlock()
		return errors.ErrNoSlaveDB
	}
	slaves := make([]*Node, len(g.Slave))
	copy(slaves, g.Slave)
	g.RUnlock()

	//slave is *Node
	for _, slave := range slaves {
		if slave.addr == addr {
			slave.Close()
			atomic.StoreInt32(&(slave.state), state)
			break
		}
	}
	return nil
}

func (g *Group) ParseNode(nodeConfig *config.NodeConfig) (*Node, error) {
	var err error
	if nodeConfig == nil {
		return nil, errors.ErrNoMasterDB
	}

	n, err := g.OpenNode(nodeConfig)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (g *Group) GetSlave(addr string) (*Node, error) {

	if len(addr) == 0 {
		return nil, errors.ErrAddressNull
	}

	for _, node := range g.Slave {
		if addr == node.addr {
			return node, nil
		}
	}

	return nil, errors.ErrSlaveNotExist
}
