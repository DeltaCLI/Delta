package validation

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// ObfuscationDetector detects and analyzes obfuscated commands
type ObfuscationDetector struct {
	patterns       []ObfuscationPattern
	enableAI       bool
	confidenceThreshold float64
}

// ObfuscationPattern represents a pattern for detecting obfuscation
type ObfuscationPattern struct {
	Name        string
	Description string
	Detector    func(string) (bool, string, float64) // Returns: detected, deobfuscated, confidence
	RiskLevel   RiskLevel
}

// ObfuscationResult contains the analysis results
type ObfuscationResult struct {
	IsObfuscated    bool
	Confidence      float64
	Techniques      []string
	Deobfuscated    string
	RiskLevel       RiskLevel
	Explanation     string
}

// NewObfuscationDetector creates a new obfuscation detector
func NewObfuscationDetector() *ObfuscationDetector {
	detector := &ObfuscationDetector{
		patterns:            []ObfuscationPattern{},
		confidenceThreshold: 0.7,
	}
	
	// Initialize patterns
	detector.initializePatterns()
	
	return detector
}

// initializePatterns sets up detection patterns
func (o *ObfuscationDetector) initializePatterns() {
	o.patterns = []ObfuscationPattern{
		{
			Name:        "Base64 Encoding",
			Description: "Detects base64 encoded commands",
			Detector:    detectBase64,
			RiskLevel:   RiskLevelHigh,
		},
		{
			Name:        "Hex Encoding",
			Description: "Detects hex encoded commands",
			Detector:    detectHexEncoding,
			RiskLevel:   RiskLevelHigh,
		},
		{
			Name:        "Unicode Escapes",
			Description: "Detects unicode escape sequences",
			Detector:    detectUnicodeEscapes,
			RiskLevel:   RiskLevelMedium,
		},
		{
			Name:        "Variable Substitution",
			Description: "Detects obfuscation through variable substitution",
			Detector:    detectVariableSubstitution,
			RiskLevel:   RiskLevelMedium,
		},
		{
			Name:        "Character Substitution",
			Description: "Detects character substitution like ${IFS}",
			Detector:    detectCharacterSubstitution,
			RiskLevel:   RiskLevelMedium,
		},
		{
			Name:        "Command Substitution Abuse",
			Description: "Detects nested command substitution",
			Detector:    detectCommandSubstitution,
			RiskLevel:   RiskLevelHigh,
		},
		{
			Name:        "Eval Chains",
			Description: "Detects eval command chains",
			Detector:    detectEvalChains,
			RiskLevel:   RiskLevelCritical,
		},
	}
}

// DetectObfuscation analyzes a command for obfuscation
func (o *ObfuscationDetector) DetectObfuscation(command string) ObfuscationResult {
	result := ObfuscationResult{
		IsObfuscated: false,
		Confidence:   0.0,
		Techniques:   []string{},
		Deobfuscated: command,
		RiskLevel:    RiskLevelLow,
	}
	
	maxConfidence := 0.0
	deobfuscatedCommand := command
	
	// Check each pattern
	for _, pattern := range o.patterns {
		detected, deobfuscated, confidence := pattern.Detector(command)
		if detected && confidence > 0.5 {
			result.IsObfuscated = true
			result.Techniques = append(result.Techniques, pattern.Name)
			
			// Track highest confidence and risk
			if confidence > maxConfidence {
				maxConfidence = confidence
				deobfuscatedCommand = deobfuscated
			}
			
			// Escalate risk level
			if isHigherRisk(pattern.RiskLevel, result.RiskLevel) {
				result.RiskLevel = pattern.RiskLevel
			}
		}
	}
	
	result.Confidence = maxConfidence
	result.Deobfuscated = deobfuscatedCommand
	
	// Generate explanation
	if result.IsObfuscated {
		result.Explanation = o.generateExplanation(result)
	}
	
	return result
}

// detectBase64 detects base64 encoded commands
func detectBase64(command string) (bool, string, float64) {
	// Pattern for base64 piped to base64 -d or similar
	base64Pattern := regexp.MustCompile(`echo\s+["']?([A-Za-z0-9+/=]+)["']?\s*\|\s*base64\s+-d`)
	matches := base64Pattern.FindStringSubmatch(command)
	
	if len(matches) > 1 {
		encoded := matches[1]
		decoded, err := base64.StdEncoding.DecodeString(encoded)
		if err == nil {
			decodedStr := string(decoded)
			// Check if decoded string looks like a command
			if looksLikeCommand(decodedStr) {
				return true, decodedStr, 0.9
			}
		}
	}
	
	// Check for direct base64 strings
	if containsBase64String(command) {
		return true, command, 0.7
	}
	
	return false, command, 0.0
}

// detectHexEncoding detects hex encoded commands
func detectHexEncoding(command string) (bool, string, float64) {
	// Pattern for hex encoding with echo -e
	hexPattern := regexp.MustCompile(`echo\s+-e\s+["']?((?:\\x[0-9a-fA-F]{2})+)["']?`)
	matches := hexPattern.FindStringSubmatch(command)
	
	if len(matches) > 1 {
		hexStr := matches[1]
		decoded := decodeHexString(hexStr)
		if looksLikeCommand(decoded) {
			return true, decoded, 0.9
		}
	}
	
	// Check for $'\x...' syntax
	bashHexPattern := regexp.MustCompile(`\$'((?:\\x[0-9a-fA-F]{2})+)'`)
	if bashHexPattern.MatchString(command) {
		decoded := bashHexPattern.ReplaceAllStringFunc(command, func(match string) string {
			hexStr := bashHexPattern.FindStringSubmatch(match)[1]
			return decodeHexString(hexStr)
		})
		return true, decoded, 0.8
	}
	
	return false, command, 0.0
}

// detectUnicodeEscapes detects unicode escape sequences
func detectUnicodeEscapes(command string) (bool, string, float64) {
	// Pattern for unicode escapes
	unicodePattern := regexp.MustCompile(`\\u[0-9a-fA-F]{4}|\\U[0-9a-fA-F]{8}`)
	
	if unicodePattern.MatchString(command) {
		decoded := unicodePattern.ReplaceAllStringFunc(command, func(match string) string {
			// Decode unicode escape
			var codePoint int64
			if strings.HasPrefix(match, "\\u") {
				codePoint, _ = strconv.ParseInt(match[2:], 16, 32)
			} else {
				codePoint, _ = strconv.ParseInt(match[2:], 16, 32)
			}
			return string(rune(codePoint))
		})
		return true, decoded, 0.7
	}
	
	return false, command, 0.0
}

// detectVariableSubstitution detects variable-based obfuscation
func detectVariableSubstitution(command string) (bool, string, float64) {
	// Pattern for variable construction like a="r"; b="m"; $a$b -rf /
	varPattern := regexp.MustCompile(`(\w+)=["']?(\w)["']?;\s*`)
	
	vars := make(map[string]string)
	if varPattern.MatchString(command) {
		matches := varPattern.FindAllStringSubmatch(command, -1)
		for _, match := range matches {
			vars[match[1]] = match[2]
		}
		
		// Check for variable usage
		varUsagePattern := regexp.MustCompile(`\$(\w+)`)
		if varUsagePattern.MatchString(command) && len(vars) > 0 {
			decoded := varUsagePattern.ReplaceAllStringFunc(command, func(match string) string {
				varName := match[1:]
				if val, ok := vars[varName]; ok {
					return val
				}
				return match
			})
			
			// Remove variable declarations
			for varName := range vars {
				decoded = regexp.MustCompile(varName+`=["']?\w["']?;\s*`).ReplaceAllString(decoded, "")
			}
			
			return true, strings.TrimSpace(decoded), 0.8
		}
	}
	
	return false, command, 0.0
}

// detectCharacterSubstitution detects ${IFS} and similar substitutions
func detectCharacterSubstitution(command string) (bool, string, float64) {
	// Common character substitutions
	substitutions := map[string]string{
		`${IFS}`:     " ",
		`$IFS`:       " ",
		`${PATH##*:}`: "",
		`${##}`:      "",
	}
	
	decoded := command
	found := false
	
	for pattern, replacement := range substitutions {
		if strings.Contains(command, pattern) {
			found = true
			decoded = strings.ReplaceAll(decoded, pattern, replacement)
		}
	}
	
	if found {
		return true, decoded, 0.7
	}
	
	return false, command, 0.0
}

// detectCommandSubstitution detects nested command substitution
func detectCommandSubstitution(command string) (bool, string, float64) {
	// Count nesting levels
	backtickCount := strings.Count(command, "`")
	dollarParenCount := strings.Count(command, "$(")
	
	totalNesting := backtickCount + dollarParenCount
	
	if totalNesting > 2 {
		// High nesting is suspicious
		confidence := float64(totalNesting) / 10.0
		if confidence > 1.0 {
			confidence = 1.0
		}
		return true, command, confidence
	}
	
	return false, command, 0.0
}

// detectEvalChains detects eval command chains
func detectEvalChains(command string) (bool, string, float64) {
	evalPattern := regexp.MustCompile(`eval\s+`)
	
	if evalPattern.MatchString(command) {
		// eval is always suspicious
		return true, command, 0.9
	}
	
	// Check for source with URLs
	sourceURLPattern := regexp.MustCompile(`source\s+<\(curl\s+|wget\s+-O\s*-|source\s+.*http`)
	if sourceURLPattern.MatchString(command) {
		return true, command, 0.95
	}
	
	return false, command, 0.0
}

// Helper functions

// looksLikeCommand checks if a string appears to be a shell command
func looksLikeCommand(s string) bool {
	// Common command indicators
	commandPatterns := []string{
		`^(rm|ls|cat|echo|curl|wget|bash|sh|python|perl|nc|chmod|chown)`,
		`\s(-rf|-la|-al|&&|\|\||;|>|<|\|)\s`,
		`(\.sh|\.py|\.pl|\.rb)(\s|$)`,
	}
	
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return false
	}
	
	for _, pattern := range commandPatterns {
		if matched, _ := regexp.MatchString(pattern, s); matched {
			return true
		}
	}
	
	// Check if mostly printable ASCII
	printableCount := 0
	for _, r := range s {
		if unicode.IsPrint(r) {
			printableCount++
		}
	}
	
	return float64(printableCount)/float64(len(s)) > 0.8
}

// containsBase64String checks if string contains base64 encoded data
func containsBase64String(s string) bool {
	// Look for long base64-like strings
	base64Regex := regexp.MustCompile(`[A-Za-z0-9+/]{20,}={0,2}`)
	matches := base64Regex.FindAllString(s, -1)
	
	for _, match := range matches {
		// Try to decode
		if decoded, err := base64.StdEncoding.DecodeString(match); err == nil {
			if looksLikeCommand(string(decoded)) {
				return true
			}
		}
	}
	
	return false
}

// decodeHexString decodes hex escape sequences
func decodeHexString(hexStr string) string {
	// Replace \xNN with actual characters
	hexPattern := regexp.MustCompile(`\\x([0-9a-fA-F]{2})`)
	
	return hexPattern.ReplaceAllStringFunc(hexStr, func(match string) string {
		hexValue := match[2:]
		if value, err := hex.DecodeString(hexValue); err == nil {
			return string(value)
		}
		return match
	})
}

// generateExplanation creates a human-readable explanation
func (o *ObfuscationDetector) generateExplanation(result ObfuscationResult) string {
	explanation := fmt.Sprintf("This command appears to be obfuscated using %d technique(s):\n", 
		len(result.Techniques))
	
	for i, technique := range result.Techniques {
		explanation += fmt.Sprintf("%d. %s\n", i+1, technique)
	}
	
	explanation += fmt.Sprintf("\nConfidence: %.0f%%\n", result.Confidence*100)
	explanation += fmt.Sprintf("Risk Level: %s\n", result.RiskLevel)
	
	if result.Deobfuscated != "" {
		explanation += fmt.Sprintf("\nDeobfuscated command:\n%s\n", result.Deobfuscated)
	}
	
	explanation += "\n⚠️  Obfuscated commands are often used to hide malicious intent."
	
	return explanation
}

// IsObfuscated is a quick check for obfuscation
func (o *ObfuscationDetector) IsObfuscated(command string) bool {
	result := o.DetectObfuscation(command)
	return result.IsObfuscated && result.Confidence >= o.confidenceThreshold
}