// Package network 提供了更上层的网络接口
//
// Example Usage
//
//	id, err := network.Listen(network.TcpGNet, ":6699", agent.GetSingleAgent)
package network

import (
	"github.com/finishy1995/go-library/network/agent"
	"github.com/finishy1995/go-library/network/core"
)

func test() {
	Listen(
		TcpGNet,
		":6699",
		agent.GetSingleAgent,
		core.WithMaxConnNum(1000),
		core.WithServerContext(map[string]interface{}{
			"test": "hello world",
		}))
}
