// =======================================================================================
// 📄 cmd/controller/main.go
//
// ✨ Description:
//     Entry point of NeuroController. This is a Kubernetes controller plugin designed
//     to run persistently inside the cluster. It dynamically enables modules such as
//     Watcher, Webhook, Scaler, Reporter, and NeuroAI based on the config.yaml file.
//
// 🧠 Startup Logic:
//     1. Initialize the logging system (zap)
//     2. Load configuration from config.yaml
//     3. Initialize Kubernetes client (controller-runtime)
//     4. Start modules in parallel as defined in the configuration
//     5. Enter the main event loop to monitor and respond to cluster events
//
// 📍 Deployment Recommendation:
//     - Deploy as a Kubernetes Deployment or DaemonSet
//     - Supports per-module enable/disable to fit different environments
//
// ✍️ Author: bukahou (@ZGMF-X10A)
// 📅 Created: June 2025
// =======================================================================================

package main

import (
	"NeuroController/internal/bootstrap"
	"NeuroController/internal/diagnosis"
	"NeuroController/internal/utils"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	// ✅ Set controller-runtime logging system (should be called first)
	ctrl.SetLogger(zap.New(zap.UseDevMode(false))) // (true): Development mode / (false): Production mode
	utils.InitLogger()

	cfg := utils.InitK8sClient()
	// ✅ Automatically select the best available API server endpoint (inside or outside the cluster)
	// api := utils.ChooseBestK8sAPI(cfg.Host)
	utils.StartK8sHealthChecker(cfg)

	// ✅ Start the periodic cleaner for the log event pool (runs every 30 seconds)
	diagnosis.StartDiagnosisSystem()

	// ✅ Register modules and start the controller manager
	bootstrap.StartManager()
}
