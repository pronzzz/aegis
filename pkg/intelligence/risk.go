package intelligence

import (
	"path/filepath"
	"strings"
)

type RiskLevel string

const (
	RiskCritical RiskLevel = "CRITICAL" // Keys, Secrets
	RiskHigh     RiskLevel = "HIGH"     // Source Code, Configs
	RiskMedium   RiskLevel = "MEDIUM"   // Documents
	RiskLow      RiskLevel = "LOW"      // Binaries, Archives, Media
)

type RiskAssessment struct {
	Level RiskLevel
	Score int
}

// AnalyzeFile determines the risk level of a file based on its extension/characteristics
func AnalyzeFile(path string) RiskAssessment {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	// Secrets / Critical Security
	case ".pem", ".key", ".kdbx", ".env", ".gpg", ".pfx", ".p12", ".ovpn", ".ssh":
		return RiskAssessment{Level: RiskCritical, Score: 100}

	// Source Code / Configs
	case ".go", ".rs", ".py", ".js", ".ts", ".c", ".cpp", ".h", ".java",
		".json", ".yaml", ".yml", ".toml", ".xml", ".conf", ".ini", ".sql", ".tf":
		return RiskAssessment{Level: RiskHigh, Score: 80}

	// Documents
	case ".pdf", ".docx", ".xlsx", ".pptx", ".md", ".txt", ".csv":
		return RiskAssessment{Level: RiskMedium, Score: 50}

	// Low Value / Heavy
	case ".mp4", ".mp3", ".mov", ".zip", ".tar", ".gz", ".iso", ".exe", ".bin", ".dll", ".so", ".dylib":
		return RiskAssessment{Level: RiskLow, Score: 10}

	default:
		return RiskAssessment{Level: RiskLow, Score: 10}
	}
}
