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
	"math/rand"
	"time"

	"github.com/fsindustry/kingshard/pkg/core/errors"
)

func Gcd(ary []int) int {
	var i int
	min := ary[0]
	length := len(ary)
	for i = 0; i < length; i++ {
		if ary[i] < min {
			min = ary[i]
		}
	}

	for {
		isCommon := true
		for i = 0; i < length; i++ {
			if ary[i]%min != 0 {
				isCommon = false
				break
			}
		}
		if isCommon {
			break
		}
		min--
		if min < 1 {
			break
		}
	}
	return min
}

func (g *Group) InitBalancer() {
	var sum int
	g.LastSlaveIndex = 0
	gcd := Gcd(g.SlaveWeights)

	for _, weight := range g.SlaveWeights {
		sum += weight / gcd
	}

	g.RoundRobinQ = make([]int, 0, sum)
	for index, weight := range g.SlaveWeights {
		for j := 0; j < weight/gcd; j++ {
			g.RoundRobinQ = append(g.RoundRobinQ, index)
		}
	}

	//random order
	if 1 < len(g.SlaveWeights) {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		for i := 0; i < sum; i++ {
			x := r.Intn(sum)
			temp := g.RoundRobinQ[x]
			other := sum % (x + 1)
			g.RoundRobinQ[x] = g.RoundRobinQ[other]
			g.RoundRobinQ[other] = temp
		}
	}
}

func (g *Group) GetNextSlave() (*Node, error) {
	var index int
	queueLen := len(g.RoundRobinQ)
	if queueLen == 0 {
		return nil, errors.ErrNoDatabase
	}
	if queueLen == 1 {
		index = g.RoundRobinQ[0]
		return g.Slave[index], nil
	}

	g.LastSlaveIndex = g.LastSlaveIndex % queueLen
	index = g.RoundRobinQ[g.LastSlaveIndex]
	if len(g.Slave) <= index {
		return nil, errors.ErrNoDatabase
	}
	node := g.Slave[index]
	g.LastSlaveIndex++
	g.LastSlaveIndex = g.LastSlaveIndex % queueLen
	return node, nil
}
