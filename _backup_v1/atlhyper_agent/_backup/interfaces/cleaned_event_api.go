package interfaces

import (
	"AtlHyper/atlhyper_agent/internal/diagnosis"
	model "AtlHyper/model/transport"
)

func GetCleanedEventLogs() []model.LogEvent {
	events := diagnosis.GetCleanedEvents()

	return events
}
