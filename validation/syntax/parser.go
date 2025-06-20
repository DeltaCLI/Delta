package syntax

import (
	"fmt"
	"delta/validation"
)

// Parser interface for shell-specific parsing
type Parser interface {
	Parse(command string) (*validation.AST, error)
	Validate(ast *validation.AST) []validation.ValidationError
	GetShellType() validation.ShellType
}

// BaseParser provides common parsing functionality
type BaseParser struct {
	input     string
	tokens    []validation.Token
	position  int
	errors    []validation.ValidationError
	shellType validation.ShellType
}

// NewBaseParser creates a new base parser
func NewBaseParser(shellType validation.ShellType) *BaseParser {
	return &BaseParser{
		shellType: shellType,
		errors:    []validation.ValidationError{},
	}
}

// GetShellType returns the shell type
func (p *BaseParser) GetShellType() validation.ShellType {
	return p.shellType
}

// current returns the current token
func (p *BaseParser) current() validation.Token {
	if p.position >= len(p.tokens) {
		return validation.Token{Type: validation.TokenEOF}
	}
	return p.tokens[p.position]
}

// peek returns the next token without advancing
func (p *BaseParser) peek() validation.Token {
	if p.position+1 >= len(p.tokens) {
		return validation.Token{Type: validation.TokenEOF}
	}
	return p.tokens[p.position+1]
}

// advance moves to the next token
func (p *BaseParser) advance() validation.Token {
	if p.position >= len(p.tokens) {
		return validation.Token{Type: validation.TokenEOF}
	}
	token := p.tokens[p.position]
	p.position++
	return token
}

// expect consumes a token of the expected type
func (p *BaseParser) expect(tokenType validation.TokenType) (validation.Token, error) {
	token := p.current()
	if token.Type != tokenType {
		return token, fmt.Errorf("expected %s, got %s", tokenType, token.Type)
	}
	return p.advance(), nil
}

// addError adds a validation error
func (p *BaseParser) addError(err validation.ValidationError) {
	p.errors = append(p.errors, err)
}

// parseCommand parses a single command
func (p *BaseParser) parseCommand() (validation.Node, error) {
	startPos := p.current().Position
	
	// Get command name
	nameToken := p.advance()
	if nameToken.Type != validation.TokenWord {
		return nil, fmt.Errorf("expected command name, got %s", nameToken.Type)
	}
	
	args := []string{}
	
	// Parse arguments
	for {
		token := p.current()
		if token.Type == validation.TokenEOF || token.IsDelimiter() || token.IsOperator() {
			break
		}
		
		switch token.Type {
		case validation.TokenWord:
			args = append(args, token.Value)
			p.advance()
		case validation.TokenString:
			args = append(args, token.Value)
			p.advance()
		case validation.TokenVariable:
			// Handle variable expansion
			args = append(args, token.Value)
			p.advance()
		default:
			if token.IsRedirect() {
				// Handle redirections
				break
			}
			p.advance()
		}
	}
	
	return validation.NewCommandNode(nameToken.Value, args, startPos), nil
}

// parsePipeline parses a pipeline of commands
func (p *BaseParser) parsePipeline() (validation.Node, error) {
	startPos := p.current().Position
	commands := []validation.Node{}
	
	// Parse first command
	cmd, err := p.parseCommand()
	if err != nil {
		return nil, err
	}
	commands = append(commands, cmd)
	
	// Parse additional piped commands
	for p.current().Type == validation.TokenPipe {
		p.advance() // consume pipe
		
		// Check for trailing pipe
		if p.current().Type == validation.TokenEOF || p.current().IsDelimiter() {
			p.addError(validation.ValidationError{
				Type:     validation.ErrorSyntax,
				Severity: validation.SeverityError,
				Position: p.current().Position,
				Message:  "Unexpected end of command after pipe",
			})
			break
		}
		
		cmd, err := p.parseCommand()
		if err != nil {
			return nil, err
		}
		commands = append(commands, cmd)
	}
	
	if len(commands) == 1 {
		return commands[0], nil
	}
	
	return validation.NewPipelineNode(commands, startPos), nil
}

// parseCommandList parses a list of commands
func (p *BaseParser) parseCommandList() (validation.Node, error) {
	commands := []validation.Node{}
	
	for p.current().Type != validation.TokenEOF {
		// Skip newlines
		if p.current().Type == validation.TokenNewline {
			p.advance()
			continue
		}
		
		// Parse pipeline or command
		cmd, err := p.parsePipeline()
		if err != nil {
			return nil, err
		}
		commands = append(commands, cmd)
		
		// Check for command separator
		token := p.current()
		if token.Type == validation.TokenSemicolon ||
		   token.Type == validation.TokenAmpersand ||
		   token.Type == validation.TokenNewline {
			p.advance()
		} else if token.Type != validation.TokenEOF {
			// Unexpected token
			p.addError(validation.ValidationError{
				Type:     validation.ErrorSyntax,
				Severity: validation.SeverityError,
				Position: token.Position,
				Message:  fmt.Sprintf("Unexpected token: %s", token.Type),
			})
			break
		}
	}
	
	if len(commands) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	
	if len(commands) == 1 {
		return commands[0], nil
	}
	
	// Create a list node
	return &validation.ListNode{
		BaseNode: validation.BaseNode{
			NodeType:   validation.NodeList,
			ChildNodes: commands,
		},
		Commands: commands,
	}, nil
}