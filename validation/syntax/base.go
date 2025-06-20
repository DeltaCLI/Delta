package syntax

import (
	validation "delta/validation"
)

// BaseParser provides common parsing functionality
type BaseParser struct {
	shellType validation.ShellType
	input     string
	tokens    []validation.Token
	position  int
	errors    []validation.ValidationError
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
	token := p.current()
	if p.position < len(p.tokens) {
		p.position++
	}
	return token
}

// expectToken checks if the current token matches the expected type
func (p *BaseParser) expectToken(tokenType validation.TokenType) bool {
	return p.current().Type == tokenType
}

// consumeToken consumes a token if it matches the expected type
func (p *BaseParser) consumeToken(tokenType validation.TokenType) bool {
	if p.expectToken(tokenType) {
		p.advance()
		return true
	}
	return false
}

// addError adds a parsing error
func (p *BaseParser) addError(message string, position validation.Position) {
	p.errors = append(p.errors, validation.ValidationError{
		Type:     validation.ErrorSyntax,
		Severity: validation.SeverityError,
		Position: position,
		Message:  message,
	})
}

// parseCommandList parses a list of commands
func (p *BaseParser) parseCommandList() (validation.Node, error) {
	commands := []validation.Node{}
	
	for p.current().Type != validation.TokenEOF {
		cmd, err := p.parseCommand()
		if err != nil {
			return nil, err
		}
		if cmd != nil {
			commands = append(commands, cmd)
		}
		
		// Consume separators
		if p.consumeToken(validation.TokenSemicolon) ||
			p.consumeToken(validation.TokenNewline) ||
			p.consumeToken(validation.TokenAnd) ||
			p.consumeToken(validation.TokenOr) {
			continue
		}
		
		// If no separator and not EOF, we might have an error
		if p.current().Type != validation.TokenEOF {
			p.addError("unexpected token", p.current().Position)
			p.advance() // Skip the problematic token
		}
	}
	
	if len(commands) == 0 {
		return nil, nil
	}
	
	if len(commands) == 1 {
		return commands[0], nil
	}
	
	return &validation.ListNode{
		Children: commands,
	}, nil
}

// parseCommand parses a single command
func (p *BaseParser) parseCommand() (validation.Node, error) {
	startPos := p.current().Position
	
	// Skip empty lines
	for p.consumeToken(validation.TokenNewline) {
		// Keep consuming newlines
	}
	
	if p.current().Type == validation.TokenEOF {
		return nil, nil
	}
	
	// Parse the command name
	if p.current().Type != validation.TokenWord {
		return nil, nil
	}
	
	cmdName := p.advance().Value
	args := []string{}
	
	// Parse arguments
	for {
		token := p.current()
		switch token.Type {
		case validation.TokenWord, validation.TokenString:
			args = append(args, token.Value)
			p.advance()
		case validation.TokenVariable:
			// Handle variable expansion
			args = append(args, "$"+token.Value)
			p.advance()
		case validation.TokenPipe:
			// Handle pipeline
			p.advance()
			nextCmd, err := p.parseCommand()
			if err != nil {
				return nil, err
			}
			return &validation.PipelineNode{
				Commands: []validation.Node{
					&validation.CommandNode{
						Name: cmdName,
						Args: args,
						Pos:  startPos,
					},
					nextCmd,
				},
			}, nil
		case validation.TokenRedirectIn, validation.TokenRedirectOut,
			validation.TokenRedirectAppend, validation.TokenRedirectErr:
			// Handle redirection
			redirectType := token.Type
			p.advance()
			target := p.current()
			if target.Type != validation.TokenWord && target.Type != validation.TokenString {
				p.addError("expected filename after redirection", target.Position)
				return nil, nil
			}
			p.advance()
			// For now, we'll just add it as an argument
			// TODO: Properly handle redirections in AST
			args = append(args, string(redirectType), target.Value)
		default:
			// End of command
			goto done
		}
	}
	
done:
	return &validation.CommandNode{
		Name: cmdName,
		Args: args,
		Pos:  startPos,
	}, nil
}