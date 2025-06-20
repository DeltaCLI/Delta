package syntax

import (
	"fmt"
	"strings"
	"delta/validation"
)

// BashParser implements Parser for Bash syntax
type BashParser struct {
	*BaseParser
	strictMode bool
}

// NewBashParser creates a new Bash parser
func NewBashParser(strictMode bool) *BashParser {
	return &BashParser{
		BaseParser: NewBaseParser(validation.ShellBash),
		strictMode: strictMode,
	}
}

// Parse parses a Bash command into an AST
func (p *BashParser) Parse(command string) (*validation.AST, error) {
	p.input = command
	
	// Tokenize the command
	tokens, err := p.tokenize()
	if err != nil {
		return nil, fmt.Errorf("tokenization failed: %w", err)
	}
	p.tokens = tokens
	p.position = 0
	
	// Parse the tokens into an AST
	root, err := p.parseCommandList()
	if err != nil {
		return nil, fmt.Errorf("parsing failed: %w", err)
	}
	
	return &validation.AST{
		Root: root,
		Metadata: map[string]interface{}{
			"shell":  "bash",
			"strict": p.strictMode,
			"errors": p.errors,
		},
	}, nil
}

// tokenize breaks the input into tokens
func (p *BashParser) tokenize() ([]validation.Token, error) {
	lexer := NewBashLexer(p.input)
	tokens := []validation.Token{}
	
	for {
		token, err := lexer.NextToken()
		if err != nil {
			return nil, err
		}
		
		tokens = append(tokens, token)
		
		if token.Type == validation.TokenEOF {
			break
		}
	}
	
	return tokens, nil
}

// Validate performs Bash-specific validation
func (p *BashParser) Validate(ast *validation.AST) []validation.ValidationError {
	errors := []validation.ValidationError{}
	
	// Walk the AST and validate each node
	validation.Walk(ast.Root, func(node validation.Node) bool {
		// Basic node validation
		nodeErrors := node.Validate()
		errors = append(errors, nodeErrors...)
		
		// Bash-specific validations
		switch n := node.(type) {
		case *validation.StringNode:
			errors = append(errors, p.validateQuoting(n)...)
		case *validation.VariableNode:
			errors = append(errors, p.validateVariable(n)...)
		case *validation.CommandNode:
			errors = append(errors, p.validateCommand(n)...)
		case *validation.RedirectNode:
			errors = append(errors, p.validateRedirection(n)...)
		}
		
		return true // continue walking
	})
	
	// Add any parsing errors
	if metadata, ok := ast.Metadata["errors"]; ok {
		if parsingErrors, ok := metadata.([]validation.ValidationError); ok {
			errors = append(errors, parsingErrors...)
		}
	}
	
	return errors
}

// validateQuoting checks for proper quote matching and escaping
func (p *BashParser) validateQuoting(node *validation.StringNode) []validation.ValidationError {
	errors := []validation.ValidationError{}
	
	// Check for unescaped quotes within strings
	if node.QuoteType == validation.QuoteDouble {
		// In double quotes, check for unescaped double quotes
		unescaped := 0
		escaped := false
		for _, ch := range node.Value {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				unescaped++
			}
		}
		
		if unescaped > 0 {
			errors = append(errors, validation.ValidationError{
				Type:     validation.ErrorSyntax,
				Severity: validation.SeverityError,
				Position: node.Position(),
				Message:  "Unescaped double quote within double-quoted string",
				Suggestion: "Escape with backslash: \\\"",
			})
		}
	}
	
	return errors
}

// validateVariable checks variable syntax
func (p *BashParser) validateVariable(node *validation.VariableNode) []validation.ValidationError {
	errors := []validation.ValidationError{}
	
	// Check for valid variable name
	if !isValidBashVariableName(node.Name) {
		errors = append(errors, validation.ValidationError{
			Type:       validation.ErrorSyntax,
			Severity:   validation.SeverityError,
			Position:   node.Position(),
			Message:    fmt.Sprintf("Invalid variable name: %s", node.Name),
			Suggestion: "Variable names must start with letter or underscore",
		})
	}
	
	return errors
}

// validateCommand performs command-specific validation
func (p *BashParser) validateCommand(node *validation.CommandNode) []validation.ValidationError {
	errors := []validation.ValidationError{}
	
	// Check for common Bash syntax errors
	if p.strictMode {
		// Check for deprecated backticks
		for _, arg := range node.Args {
			if strings.Contains(arg, "`") {
				errors = append(errors, validation.ValidationError{
					Type:       validation.ErrorDeprecated,
					Severity:   validation.SeverityWarning,
					Position:   node.Position(),
					Message:    "Backticks for command substitution are deprecated",
					Suggestion: "Use $(...) instead of `...`",
				})
			}
		}
	}
	
	return errors
}

// validateRedirection checks redirection syntax
func (p *BashParser) validateRedirection(node *validation.RedirectNode) []validation.ValidationError {
	errors := []validation.ValidationError{}
	
	// Check for common redirection errors
	if node.Target == "" {
		errors = append(errors, validation.ValidationError{
			Type:     validation.ErrorSyntax,
			Severity: validation.SeverityError,
			Position: node.Position(),
			Message:  "Missing redirection target",
		})
	}
	
	// Check for invalid file descriptors
	if node.Fd < 0 || node.Fd > 9 {
		errors = append(errors, validation.ValidationError{
			Type:     validation.ErrorSyntax,
			Severity: validation.SeverityError,
			Position: node.Position(),
			Message:  fmt.Sprintf("Invalid file descriptor: %d", node.Fd),
		})
	}
	
	return errors
}

// BashLexer tokenizes Bash commands
type BashLexer struct {
	*validation.BaseLexer
}

// NewBashLexer creates a new Bash lexer
func NewBashLexer(input string) *BashLexer {
	return &BashLexer{
		BaseLexer: validation.NewBaseLexer(input),
	}
}

// NextToken returns the next token
func (l *BashLexer) NextToken() (validation.Token, error) {
	l.skipWhitespace()
	
	if l.position >= len(l.input) {
		return validation.Token{
			Type:     validation.TokenEOF,
			Position: l.Position(),
		}, nil
	}
	
	startPos := l.Position()
	ch := l.peek()
	
	// Handle different token types
	switch ch {
	case '|':
		l.advance()
		if l.peek() == '|' {
			l.advance()
			return validation.Token{
				Type:     validation.TokenOr,
				Value:    "||",
				Position: startPos,
			}, nil
		}
		return validation.Token{
			Type:     validation.TokenPipe,
			Value:    "|",
			Position: startPos,
		}, nil
		
	case '&':
		l.advance()
		if l.peek() == '&' {
			l.advance()
			return validation.Token{
				Type:     validation.TokenAnd,
				Value:    "&&",
				Position: startPos,
			}, nil
		}
		return validation.Token{
			Type:     validation.TokenAmpersand,
			Value:    "&",
			Position: startPos,
		}, nil
		
	case ';':
		l.advance()
		return validation.Token{
			Type:     validation.TokenSemicolon,
			Value:    ";",
			Position: startPos,
		}, nil
		
	case '\n':
		l.advance()
		return validation.Token{
			Type:     validation.TokenNewline,
			Position: startPos,
		}, nil
		
	case '<':
		l.advance()
		if l.peek() == '<' {
			l.advance()
			return validation.Token{
				Type:     validation.TokenRedirectHere,
				Value:    "<<",
				Position: startPos,
			}, nil
		}
		return validation.Token{
			Type:     validation.TokenRedirectIn,
			Value:    "<",
			Position: startPos,
		}, nil
		
	case '>':
		l.advance()
		if l.peek() == '>' {
			l.advance()
			return validation.Token{
				Type:     validation.TokenRedirectAppend,
				Value:    ">>",
				Position: startPos,
			}, nil
		}
		return validation.Token{
			Type:     validation.TokenRedirectOut,
			Value:    ">",
			Position: startPos,
		}, nil
		
	case '"':
		str, err := l.readQuotedString('"')
		if err != nil {
			return validation.Token{
				Type:     validation.TokenError,
				Value:    err.Error(),
				Position: startPos,
			}, err
		}
		return validation.Token{
			Type:     validation.TokenString,
			Value:    str,
			Position: startPos,
		}, nil
		
	case '\'':
		str, err := l.readQuotedString('\'')
		if err != nil {
			return validation.Token{
				Type:     validation.TokenError,
				Value:    err.Error(),
				Position: startPos,
			}, err
		}
		return validation.Token{
			Type:     validation.TokenString,
			Value:    str,
			Position: startPos,
		}, nil
		
	case '$':
		return l.readVariable()
		
	case '(':
		l.advance()
		return validation.Token{
			Type:     validation.TokenLeftParen,
			Value:    "(",
			Position: startPos,
		}, nil
		
	case ')':
		l.advance()
		return validation.Token{
			Type:     validation.TokenRightParen,
			Value:    ")",
			Position: startPos,
		}, nil
		
	default:
		// Read a word
		word := l.readWord()
		if word == "" {
			l.advance() // Skip unknown character
			return validation.Token{
				Type:     validation.TokenUnknown,
				Value:    string(ch),
				Position: startPos,
			}, nil
		}
		return validation.Token{
			Type:     validation.TokenWord,
			Value:    word,
			Position: startPos,
		}, nil
	}
}

// readVariable reads a variable token
func (l *BashLexer) readVariable() (validation.Token, error) {
	startPos := l.Position()
	l.advance() // Skip $
	
	if l.peek() == '{' {
		// ${VAR} format
		l.advance() // Skip {
		name := ""
		for l.peek() != '}' && l.peek() != 0 {
			name += string(l.advance())
		}
		if l.peek() == '}' {
			l.advance()
		}
		return validation.Token{
			Type:     validation.TokenVariable,
			Value:    name,
			Position: startPos,
		}, nil
	}
	
	// $VAR format
	name := ""
	for isValidBashVariableChar(l.peek()) {
		name += string(l.advance())
	}
	
	return validation.Token{
		Type:     validation.TokenVariable,
		Value:    name,
		Position: startPos,
	}, nil
}

// isValidBashVariableName checks if a string is a valid Bash variable name
func isValidBashVariableName(name string) bool {
	if len(name) == 0 {
		return false
	}
	
	// First character must be letter or underscore
	first := name[0]
	if !((first >= 'a' && first <= 'z') || 
		(first >= 'A' && first <= 'Z') || 
		first == '_') {
		return false
	}
	
	// Rest can be letters, numbers, or underscore
	for i := 1; i < len(name); i++ {
		ch := name[i]
		if !((ch >= 'a' && ch <= 'z') || 
			(ch >= 'A' && ch <= 'Z') || 
			(ch >= '0' && ch <= '9') || 
			ch == '_') {
			return false
		}
	}
	
	return true
}

// isValidBashVariableChar checks if a character can be part of a variable name
func isValidBashVariableChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_'
}