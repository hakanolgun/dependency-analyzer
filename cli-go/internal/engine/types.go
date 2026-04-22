package engine

type Weights struct {
	Native          float64
	Volume          float64
	APISurface      float64
	Entanglement    float64
	LogicComplexity float64
}

type Metrics struct {
	Native          float64 `json:"native"`
	Volume          float64 `json:"volume"`
	APISurface      float64 `json:"apiSurface"`
	Entanglement    float64 `json:"entanglement"`
	LogicComplexity float64 `json:"logicComplexity"`
}

type DependencyResult struct {
	Name       string  `json:"name"`
	Version    string  `json:"version"`
	Normalized float64 `json:"normalized"`
	Score      int     `json:"score"`
	Label      string  `json:"label"`
	Metrics    Metrics `json:"metrics"`
	Error      string  `json:"error,omitempty"`

	// Registry metadata (npmjs + downloads API); mirrors analyzer-js fetchPackageData.
	LatestVersion       string `json:"latestVersion,omitempty"`
	RepoURL             string `json:"repoUrl,omitempty"`
	WeeklyDownloads     *int   `json:"weeklyDownloads,omitempty"`
	LastUpdateDate      string `json:"lastUpdateDate,omitempty"`
	TimeSinceLastUpdate string `json:"timeSinceLastUpdate,omitempty"`
	IsMaintained        string `json:"isMaintained,omitempty"` // yes | unlikely | no

	// React Native directory (only populated when root has react-native).
	IsReactNativeLib bool  `json:"isReactNativeLib,omitempty"`
	NewArchitecture  *bool `json:"newArchitecture,omitempty"`
}

type ProjectReport struct {
	ProjectPath    string             `json:"projectPath"`
	Dependencies   []DependencyResult `json:"dependencies"`
	GeneratedAt    string             `json:"generatedAt"`
	ScannedCount   int                `json:"scannedCount"`
	FailedCount    int                `json:"failedCount"`
	ReportOutPath  string             `json:"reportOutPath"`
	HasReactNative bool               `json:"hasReactNative"`
}
