package main

import (
	"sync"
	"time"
)

const creditIntelligenceProgressTotal = 11

type CreditIntelligenceProgress struct {
	Running   bool   `json:"running"`
	Done      bool   `json:"done"`
	Failed    bool   `json:"failed"`
	Step      int    `json:"step"`
	Total     int    `json:"total"`
	Key       string `json:"key"`
	Label     string `json:"label"`
	Detail    string `json:"detail"`
	StartedAt string `json:"started_at"`
	UpdatedAt string `json:"updated_at"`
	Error     string `json:"error"`
}

var creditIntelligenceProgressState struct {
	sync.RWMutex
	Value CreditIntelligenceProgress
}

func startCreditIntelligenceProgress() {
	now := time.Now().Format(time.RFC3339)
	creditIntelligenceProgressState.Lock()
	creditIntelligenceProgressState.Value = CreditIntelligenceProgress{
		Running:   true,
		Step:      1,
		Total:     creditIntelligenceProgressTotal,
		Key:       "operations",
		Label:     "Operações SICOR",
		Detail:    "Localizando as operações públicas vinculadas ao CAR.",
		StartedAt: now,
		UpdatedAt: now,
	}
	creditIntelligenceProgressState.Unlock()
}

func updateCreditIntelligenceProgress(step int, key, label, detail string) {
	now := time.Now().Format(time.RFC3339)
	creditIntelligenceProgressState.Lock()
	p := creditIntelligenceProgressState.Value
	if p.StartedAt == "" {
		p.StartedAt = now
	}
	p.Running = true
	p.Done = false
	p.Failed = false
	p.Step = step
	p.Total = creditIntelligenceProgressTotal
	p.Key = key
	p.Label = label
	p.Detail = detail
	p.UpdatedAt = now
	p.Error = ""
	creditIntelligenceProgressState.Value = p
	creditIntelligenceProgressState.Unlock()
}

func completeCreditIntelligenceProgress(detail string) {
	now := time.Now().Format(time.RFC3339)
	creditIntelligenceProgressState.Lock()
	p := creditIntelligenceProgressState.Value
	p.Running = false
	p.Done = true
	p.Failed = false
	p.Step = creditIntelligenceProgressTotal
	p.Total = creditIntelligenceProgressTotal
	p.Key = "done"
	p.Label = "Consulta concluída"
	p.Detail = detail
	p.UpdatedAt = now
	p.Error = ""
	creditIntelligenceProgressState.Value = p
	creditIntelligenceProgressState.Unlock()
}

func failCreditIntelligenceProgress(err error) {
	now := time.Now().Format(time.RFC3339)
	creditIntelligenceProgressState.Lock()
	p := creditIntelligenceProgressState.Value
	p.Running = false
	p.Done = false
	p.Failed = true
	if p.Total == 0 {
		p.Total = creditIntelligenceProgressTotal
	}
	p.UpdatedAt = now
	if err != nil {
		p.Error = err.Error()
		p.Detail = err.Error()
	}
	creditIntelligenceProgressState.Value = p
	creditIntelligenceProgressState.Unlock()
}

func (a *App) GetCreditIntelligenceProgress() CreditIntelligenceProgress {
	creditIntelligenceProgressState.RLock()
	defer creditIntelligenceProgressState.RUnlock()
	return creditIntelligenceProgressState.Value
}
