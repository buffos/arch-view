package export

import (
	"sort"

	"github.com/buffo/arch-view/internal/viewer"
)

const (
	deterministicNodeWidth  = 190
	deterministicNodeHeight = 82
	deterministicColumnGap  = 84
	deterministicRowGap     = 35
)

type deterministicLayout struct {
	Engine    string                             `json:"engine"`
	Key       string                             `json:"key"`
	Width     float64                            `json:"width"`
	Height    float64                            `json:"height"`
	Positions map[string]deterministicLayoutNode `json:"positions"`
	Edges     map[string]deterministicLayoutEdge `json:"edges"`
}

type deterministicLayoutNode struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

type deterministicLayoutPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type deterministicLayoutEdge struct {
	Points []deterministicLayoutPoint `json:"points"`
	LabelX float64                    `json:"label_x"`
	LabelY float64                    `json:"label_y"`
}

func buildDeterministicLayout(scene viewer.SceneSnapshot) deterministicLayout {
	byLayer := make(map[int][]viewer.VisibleNode)
	for _, node := range scene.VisibleNodes {
		layer := 0
		if node.Layer != nil {
			layer = *node.Layer
		} else if len(node.Layers) > 0 {
			layer = node.Layers[0]
		}
		byLayer[layer] = append(byLayer[layer], node)
	}
	layers := make([]int, 0, len(byLayer))
	for layer := range byLayer {
		layers = append(layers, layer)
	}
	sort.Ints(layers)
	maxRows := 1
	for _, layer := range layers {
		sort.SliceStable(byLayer[layer], func(left, right int) bool {
			if byLayer[layer][left].Label == byLayer[layer][right].Label {
				return byLayer[layer][left].ID < byLayer[layer][right].ID
			}
			return byLayer[layer][left].Label < byLayer[layer][right].Label
		})
		if len(byLayer[layer]) > maxRows {
			maxRows = len(byLayer[layer])
		}
	}

	width := float64(maxInt(760, len(layers)*(deterministicNodeWidth+deterministicColumnGap)+80))
	height := float64(maxInt(430, maxRows*(deterministicNodeHeight+deterministicRowGap)+100))
	positions := make(map[string]deterministicLayoutNode, len(scene.VisibleNodes))
	for layerIndex, layer := range layers {
		for rowIndex, node := range byLayer[layer] {
			positions[node.ID] = deterministicLayoutNode{
				X:      float64(40 + layerIndex*(deterministicNodeWidth+deterministicColumnGap)),
				Y:      float64(42 + rowIndex*(deterministicNodeHeight+deterministicRowGap)),
				Width:  deterministicNodeWidth,
				Height: deterministicNodeHeight,
			}
		}
	}

	edges := make(map[string]deterministicLayoutEdge, len(scene.VisibleRelationships))
	for _, relationship := range scene.VisibleRelationships {
		from, fromOK := positions[relationship.FromVisibleID]
		to, toOK := positions[relationship.ToVisibleID]
		if !fromOK || !toOK || relationship.FromVisibleID == relationship.ToVisibleID {
			continue
		}
		edges[relationship.ID] = orthogonalEdge(from, to)
	}
	return deterministicLayout{
		Engine:    "deterministic-export",
		Key:       sceneLayoutKey(scene),
		Width:     width,
		Height:    height,
		Positions: positions,
		Edges:     edges,
	}
}

func orthogonalEdge(from, to deterministicLayoutNode) deterministicLayoutEdge {
	fromCenterX := from.X + from.Width/2
	fromCenterY := from.Y + from.Height/2
	toCenterX := to.X + to.Width/2
	toCenterY := to.Y + to.Height/2
	points := []deterministicLayoutPoint{}
	if absFloat(toCenterX-fromCenterX) >= absFloat(toCenterY-fromCenterY) {
		forward := toCenterX >= fromCenterX
		sourceX := from.X
		targetX := to.X + to.Width
		if forward {
			sourceX = from.X + from.Width
			targetX = to.X
		}
		middleX := (sourceX + targetX) / 2
		points = append(points,
			deterministicLayoutPoint{X: sourceX, Y: fromCenterY},
			deterministicLayoutPoint{X: middleX, Y: fromCenterY},
			deterministicLayoutPoint{X: middleX, Y: toCenterY},
			deterministicLayoutPoint{X: targetX, Y: toCenterY},
		)
	} else {
		forward := toCenterY >= fromCenterY
		sourceY := from.Y
		targetY := to.Y + to.Height
		if forward {
			sourceY = from.Y + from.Height
			targetY = to.Y
		}
		middleY := (sourceY + targetY) / 2
		points = append(points,
			deterministicLayoutPoint{X: fromCenterX, Y: sourceY},
			deterministicLayoutPoint{X: fromCenterX, Y: middleY},
			deterministicLayoutPoint{X: toCenterX, Y: middleY},
			deterministicLayoutPoint{X: toCenterX, Y: targetY},
		)
	}
	middle := points[len(points)/2]
	return deterministicLayoutEdge{Points: points, LabelX: middle.X, LabelY: middle.Y - 7}
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}
