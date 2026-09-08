package index

import (
	"fishyAHP/LogParser.git/internal/features/index/component"
	"fishyAHP/LogParser.git/internal/features/index/level"
	"fishyAHP/LogParser.git/internal/features/index/pid"
	"fishyAHP/LogParser.git/internal/features/index/timestamp"
)

type Indexer struct {
	level     *level.Index
	component *component.Index
	pid       *pid.Index
	timestamp *timestamp.Index
}
