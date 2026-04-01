package providers

const (
	RunningStatus = "running"
	Two           = 2
)

type runOptions struct {
	DryRun      bool
	Interactive bool
	TailLogs    bool
}

type inspectOutput struct {
	ID    string `json:"Id"`
	State struct {
		Status string `json:"Status"`
	} `json:"State"`
	Config struct {
		Labels map[string]string `json:"Labels"`
		Env    []string          `json:"Env"`
	} `json:"Config"`
}
