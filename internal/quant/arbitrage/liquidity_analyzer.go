package arbitrage

import (
	"github.com/wyfcoding/pkg/algorithm"
)

// MarketLiquidity 代表交易对的深度
type MarketLiquidity struct {
	FromAsset string
	ToAsset   string
	MaxAmount int64 // 该路径最大承载资金量
}

// LiquidityAnalyzer 深度分析器
type LiquidityAnalyzer struct {
	assets map[string]int
}

// NewLiquidityAnalyzer 创建分析器
func NewLiquidityAnalyzer() *LiquidityAnalyzer {
	return &LiquidityAnalyzer{
		assets: make(map[string]int),
	}
}

// CalculateMaxRouteAmount 计算从 source 到 sink 路径上能通过的最大资金量
func (a *LiquidityAnalyzer) CalculateMaxRouteAmount(source, sink string, markets []MarketLiquidity) int64 {
	// 1. 映射资产名称到节点 ID
	nodeID := 0
	for _, m := range markets {
		if _, ok := a.assets[m.FromAsset]; !ok {
			a.assets[m.FromAsset] = nodeID
			nodeID++
		}
		if _, ok := a.assets[m.ToAsset]; !ok {
			a.assets[m.ToAsset] = nodeID
			nodeID++
		}
	}

	// 2. 构建 Dinic 图
	graph := algorithm.NewDinicGraph(nodeID)
	for _, m := range markets {
		graph.AddEdge(a.assets[m.FromAsset], a.assets[m.ToAsset], m.MaxAmount)
	}

	// 3. 计算最大流
	sID, okS := a.assets[source]
	tID, okT := a.assets[sink]
	if !okS || !okT {
		return 0
	}

	return graph.MaxFlow(sID, tID)
}
