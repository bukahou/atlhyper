// =======================================================================================
// 📄 interface/alert_group_builder_api.go
//
// 📦 Description:
//     Interface bridge for the Alerter module, exposing formatted alert construction logic.
//     Wraps EvaluateAlertsFromCleanedEvents and provides a unified entry point for
//     dispatchers and external modules.
//
// 🔌 Responsibilities:
//     - Evaluate whether an alert should be triggered from a set of events
//     - Construct AlertGroupData for use in email or alert display
//
// 🧩 Internal Dependency:
//     - alerter.EvaluateAlertsFromCleanedEvents
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// =======================================================================================

package alert

import (
	"NeuroController/internal/alerter"
	"NeuroController/internal/types"
	"NeuroController/model"
)

func ComposeAlertGroupIfNecessary(events []model.LogEvent) (bool, string, types.AlertGroupData) {
	shouldAlert, subject, data := alerter.EvaluateAlertsFromCleanedEvents(events)

	return shouldAlert, subject, data
}
