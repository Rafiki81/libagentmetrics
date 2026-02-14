package monitor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Rafiki81/libagentmetrics/agent"
	"github.com/Rafiki81/libagentmetrics/config"
)

// LocalModelMonitor checks for locally running model servers.
type LocalModelMonitor struct {
	mu     sync.Mutex
	config config.LocalModelsConfig
	client *http.Client
	models []agent.LocalModelInfo

	prevRequests map[string]int64
	prevTokens   map[string]int64
	prevTime     map[string]time.Time
}

// NewLocalModelMonitor creates a new local model monitor.
func NewLocalModelMonitor(cfg config.LocalModelsConfig) *LocalModelMonitor {
	return &LocalModelMonitor{
		config: cfg,
		client: &http.Client{
			Timeout: 2 * time.Second,
		},
		models:       make([]agent.LocalModelInfo, 0),
		prevRequests: make(map[string]int64),
		prevTokens:   make(map[string]int64),
		prevTime:     make(map[string]time.Time),
	}
}

type serverDef struct {
	Name         string
	ID           string
	DefaultPort  int
	ProcessNames []string
}

func knownServers() []serverDef {
	return []serverDef{
		{Name: "Ollama", ID: "ollama", DefaultPort: 11434, ProcessNames: []string{"ollama"}},
		{Name: "LM Studio", ID: "lm-studio", DefaultPort: 1234, ProcessNames: []string{"lms", "LM Studio", "lmstudio"}},
		{Name: "llama.cpp", ID: "llama-cpp", DefaultPort: 8080, ProcessNames: []string{"llama-server", "llama-cli", "server"}},
		{Name: "vLLM", ID: "vllm", DefaultPort: 8000, ProcessNames: []string{"vllm"}},
		{Name: "LocalAI", ID: "localai", DefaultPort: 8080, ProcessNames: []string{"local-ai"}},
		{Name: "text-generation-webui", ID: "text-gen-webui", DefaultPort: 5000, ProcessNames: []string{"text-generation"}},
		{Name: "GPT4All", ID: "gpt4all", DefaultPort: 4891, ProcessNames: []string{"gpt4all", "chat"}},
	}
}

// Collect scans for all local model servers and updates their status.
func (lm *LocalModelMonitor) Collect() []agent.LocalModelInfo {
	if !lm.config.Enabled {
		return nil
	}

	lm.mu.Lock()
	defer lm.mu.Unlock()

	var results []agent.LocalModelInfo

	for _, ep := range lm.config.Endpoints {
		info := lm.probeOpenAICompatible(ep.Name, ep.ID, ep.URL)
		if info != nil {
			info.PID = lm.findProcessPID([]string{ep.ID, ep.Name})
			lm.calculateRates(info)
			results = append(results, *info)
		}
	}

	for _, srv := range knownServers() {
		alreadyFound := false
		for _, r := range results {
			if r.ServerID == srv.ID {
				alreadyFound = true
				break
			}
		}
		if alreadyFound {
			continue
		}

		endpoint := fmt.Sprintf("http://localhost:%d", srv.DefaultPort)

		var info *agent.LocalModelInfo
		if srv.ID == "ollama" {
			info = lm.probeOllama(endpoint)
		} else {
			info = lm.probeOpenAICompatible(srv.Name, srv.ID, endpoint)
		}

		if info != nil {
			info.PID = lm.findProcessPID(srv.ProcessNames)
			if info.PID > 0 {
				info.CPU, info.MemoryMB = lm.getProcessStats(info.PID)
			}
			lm.calculateRates(info)
			results = append(results, *info)
		}
	}

	lm.models = results
	return results
}

// GetModels returns the last collected model info.
func (lm *LocalModelMonitor) GetModels() []agent.LocalModelInfo {
	lm.mu.Lock()
	defer lm.mu.Unlock()
	result := make([]agent.LocalModelInfo, len(lm.models))
	copy(result, lm.models)
	return result
}

// --- Ollama-specific probing ---

type ollamaTagsResponse struct {
	Models []ollamaModel `json:"models"`
}

type ollamaModel struct {
	Name       string `json:"name"`
	Model      string `json:"model"`
	Size       int64  `json:"size"`
	Digest     string `json:"digest"`
	ModifiedAt string `json:"modified_at"`
	Details    struct {
		Family            string `json:"family"`
		ParameterSize     string `json:"parameter_size"`
		QuantizationLevel string `json:"quantization_level"`
	} `json:"details"`
}

type ollamaPSResponse struct {
	Models []ollamaPSModel `json:"models"`
}

type ollamaPSModel struct {
	Name      string `json:"name"`
	Model     string `json:"model"`
	Size      int64  `json:"size"`
	SizeVRAM  int64  `json:"size_vram"`
	ExpiresAt string `json:"expires_at"`
	Details   struct {
		Family            string `json:"family"`
		ParameterSize     string `json:"parameter_size"`
		QuantizationLevel string `json:"quantization_level"`
	} `json:"details"`
}

func (lm *LocalModelMonitor) probeOllama(endpoint string) *agent.LocalModelInfo {
	resp, err := lm.client.Get(endpoint + "/api/tags")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var tagsResp ollamaTagsResponse
	if err := json.Unmarshal(body, &tagsResp); err != nil {
		return nil
	}

	info := &agent.LocalModelInfo{
		ServerName: "Ollama",
		ServerID:   "ollama",
		Endpoint:   endpoint,
		Status:     agent.LocalModelIdle,
		LastSeen:   time.Now(),
	}

	for _, m := range tagsResp.Models {
		info.Models = append(info.Models, agent.LocalModel{
			Name:       m.Name,
			SizeBytes:  m.Size,
			Size:       formatBytes(m.Size),
			QuantLevel: m.Details.QuantizationLevel,
			Family:     m.Details.Family,
			Parameters: m.Details.ParameterSize,
		})
	}

	psResp, err := lm.client.Get(endpoint + "/api/ps")
	if err == nil {
		defer psResp.Body.Close()
		if psResp.StatusCode == http.StatusOK {
			psBody, _ := io.ReadAll(psResp.Body)
			var ps ollamaPSResponse
			if json.Unmarshal(psBody, &ps) == nil {
				for _, running := range ps.Models {
					info.Status = agent.LocalModelLoaded
					info.ActiveModel = running.Name
					info.VRAM_MB = float64(running.SizeVRAM) / (1024 * 1024)

					for i := range info.Models {
						if info.Models[i].Name == running.Name {
							info.Models[i].Running = true
							info.Models[i].VRAM_MB = float64(running.SizeVRAM) / (1024 * 1024)
						}
					}
				}
			}
		}
	}

	return info
}

// --- OpenAI-compatible probing ---

type openAIModelsResponse struct {
	Data []openAIModel `json:"data"`
}

type openAIModel struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	OwnedBy string `json:"owned_by"`
}

func (lm *LocalModelMonitor) probeOpenAICompatible(name, id, endpoint string) *agent.LocalModelInfo {
	resp, err := lm.client.Get(endpoint + "/v1/models")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var modelsResp openAIModelsResponse
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return nil
	}

	info := &agent.LocalModelInfo{
		ServerName: name,
		ServerID:   id,
		Endpoint:   endpoint,
		Status:     agent.LocalModelRunning,
		LastSeen:   time.Now(),
	}

	for _, m := range modelsResp.Data {
		model := agent.LocalModel{
			Name:    m.ID,
			Running: true,
		}
		info.Models = append(info.Models, model)
		if info.ActiveModel == "" {
			info.ActiveModel = m.ID
		}
	}

	if len(info.Models) == 0 {
		info.Status = agent.LocalModelIdle
	}

	return info
}

// --- Helper functions ---

func (lm *LocalModelMonitor) findProcessPID(processNames []string) int {
	out, err := exec.Command("ps", "aux").Output()
	if err != nil {
		return 0
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		lineLower := strings.ToLower(line)
		for _, name := range processNames {
			if strings.Contains(lineLower, strings.ToLower(name)) {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if pid, err := strconv.Atoi(fields[1]); err == nil {
						return pid
					}
				}
			}
		}
	}
	return 0
}

func (lm *LocalModelMonitor) getProcessStats(pid int) (cpu float64, memMB float64) {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "%cpu=,%mem=,rss=").Output()
	if err != nil {
		return 0, 0
	}

	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) >= 3 {
		cpu, _ = strconv.ParseFloat(fields[0], 64)
		rssKB, _ := strconv.ParseFloat(fields[2], 64)
		memMB = rssKB / 1024
	}
	return
}

func (lm *LocalModelMonitor) calculateRates(info *agent.LocalModelInfo) {
	now := time.Now()
	key := info.ServerID

	if prevTime, ok := lm.prevTime[key]; ok {
		elapsed := now.Sub(prevTime).Seconds()
		if elapsed > 0 {
			prevTok := lm.prevTokens[key]
			if info.TokensGenerated > prevTok {
				info.TokensPerSec = float64(info.TokensGenerated-prevTok) / elapsed
			}
		}
	}

	lm.prevTokens[key] = info.TokensGenerated
	lm.prevRequests[key] = info.TotalRequests
	lm.prevTime[key] = now
}

func formatBytes(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
