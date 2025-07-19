// =======================================================================================
// 📄 interface/cleaned_event_api.go
//
// 📦 Description:
//     Interface bridge for the Diagnosis module that exposes access to the cleaned event pool.
//     Wraps the call to diagnosis.GetCleanedEvents and abstracts internal implementation details.
//
// 🔌 Responsibilities:
//     - Provide a unified access interface for cleaned event data
//     - Prevent external layer from directly depending on internal.diagnosis package
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package interfaces

import (
	"NeuroController/internal/diagnosis"
	"NeuroController/model"
)

func GetCleanedEventLogs() []model.LogEvent {
	events := diagnosis.GetCleanedEvents()

	return events
}
