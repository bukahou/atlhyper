package commonapi

import (
	"NeuroController/internal/types"
	"NeuroController/model"
	"NeuroController/sync/center/http"
	"log"
	"sync"
)

// ===============================
// ✅ 获取 Cleaned Events
// ===============================
// func GetCleanedEventsFromAgents() [][]types.LogEvent {
// 	var wg sync.WaitGroup
// 	var mu sync.Mutex
// 	results := make([][]types.LogEvent, 0)

// 	for _, base := range http.AgentEndpoints {
// 		wg.Add(1)
// 		go func(endpoint string) {
// 			defer wg.Done()
// 			url := endpoint + "/agent/commonapi/cleaned-events"
// 			var events []types.LogEvent
// 			if err := http.GetFromAgent(url, &events); err != nil {
// 				log.Printf("⚠️ 获取 %s 失败: %v", url, err)
// 				return
// 			}
// 			mu.Lock()
// 			results = append(results, events)
// 			mu.Unlock()
// 		}(base)
// 	}

// 	wg.Wait()
// 	return results
// }



func GetCleanedEventsFromAgents() [][]model.LogEvent {
	var result [][]model.LogEvent
	var events []model.LogEvent

	// 仅请求第一个 Agent，直接传相对路径
	err := http.GetFromAgent("/agent/commonapi/cleaned-events", &events)
	if err != nil {
		log.Printf("⚠️ 获取清洗事件失败: %v", err)
		return result
	}

	result = append(result, events)
	return result
}

type AlertResponse struct {
	Alert bool
	Title string
	Data  types.AlertGroupData
}

func GetAlertGroupFromAgents() []AlertResponse {
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]AlertResponse, 0)

	for range http.AgentEndpoints {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var resp AlertResponse
			if err := http.GetFromAgent("/agent/commonapi/alert-group", &resp); err != nil {
				log.Printf("⚠️ 获取 alert-group 失败: %v", err)
				return
			}
			if resp.Alert {
				mu.Lock()
				results = append(results, resp)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return results
}

type LightAlertResponse struct {
	Display bool
	Title   string
	Data    types.AlertGroupData
}

func GetLightweightAlertsFromAgents() []LightAlertResponse {
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]LightAlertResponse, 0)

	for range http.AgentEndpoints {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var resp LightAlertResponse
			if err := http.GetFromAgent("/agent/commonapi/alert-group-light", &resp); err != nil {
				log.Printf("⚠️ 获取 alert-group-light 失败: %v", err)
				return
			}
			if resp.Display {
				mu.Lock()
				results = append(results, resp)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return results
}



// ------------------------------------------------------------------------------------------------------




// func GetCleanedEventsFromAgents() [][]model.LogEvent {
// 	var result [][]model.LogEvent
// 	var events []model.LogEvent

// 	// 仅请求第一个 Agent
// 	err := http.GetFromAgent("/agent/commonapi/cleaned-events", &events)
// 	if err != nil {
// 		log.Printf("⚠️ 获取清洗事件失败: %v", err)
// 		return result
// 	}

// 	result = append(result, events)
// 	return result
// }


// // ===============================
// // ✅ 获取策略告警组
// // ===============================

// type AlertResponse struct {
// 	Alert bool
// 	Title string
// 	Data  types.AlertGroupData
// }

// func GetAlertGroupFromAgents() []AlertResponse {
// 	var wg sync.WaitGroup
// 	var mu sync.Mutex
// 	results := make([]AlertResponse, 0)

// 	for _, base := range http.AgentEndpoints {
// 		wg.Add(1)
// 		go func(endpoint string) {
// 			defer wg.Done()
// 			url := endpoint + "/agent/commonapi/alert-group"
// 			var resp AlertResponse
// 			if err := http.GetFromAgent(url, &resp); err != nil {
// 				log.Printf("⚠️ 获取 %s 失败: %v", url, err)
// 				return
// 			}
// 			if resp.Alert {
// 				mu.Lock()
// 				results = append(results, resp)
// 				mu.Unlock()
// 			}
// 		}(base)
// 	}

// 	wg.Wait()
// 	return results
// }



// // ===============================
// // ✅ 获取轻量告警组
// // ===============================
// type LightAlertResponse struct {
// 	Display bool
// 	Title   string
// 	Data    types.AlertGroupData
// }

// func GetLightweightAlertsFromAgents() []LightAlertResponse {
// 	var wg sync.WaitGroup
// 	var mu sync.Mutex
// 	results := make([]LightAlertResponse, 0)

// 	for _, base := range http.AgentEndpoints {
// 		wg.Add(1)
// 		go func(endpoint string) {
// 			defer wg.Done()
// 			url := endpoint + "/agent/commonapi/alert-group-light"
// 			var resp LightAlertResponse
// 			if err := http.GetFromAgent(url, &resp); err != nil {
// 				log.Printf("⚠️ 获取 %s 失败: %v", url, err)
// 				return
// 			}
// 			if resp.Display {
// 				mu.Lock()
// 				results = append(results, resp)
// 				mu.Unlock()
// 			}
// 		}(base)
// 	}

// 	wg.Wait()
// 	return results
// }
