package validation

import "fmt"

// TokenType categorizes tokens
type TokenType string

const (
	// Literals
	TokenWord       TokenType = "word"
	TokenString     TokenType = "string"
	TokenNumber     TokenType = "number"
	
	// Operators
	TokenPipe       TokenType = "pipe"       // |
	TokenAmpersand  TokenType = "ampersand"  // &
	TokenSemicolon  TokenType = "semicolon"  // ;
	TokenAnd        TokenType = "and"        // &&
	TokenOr         TokenType = "or"         // ||
	
	// Redirections
	TokenRedirectIn    TokenType = "redirect_in"    // <
	TokenRedirectOut   TokenType = "redirect_out"   // >
	TokenRedirectErr   TokenType = "redirect_err"   // 2>
	TokenRedirectAppend TokenType = "redirect_append" // >>
	TokenRedirectHere  TokenType = "redirect_here"  // <<
	
	// Quotes
	TokenSingleQuote TokenType = "single_quote" // '
	TokenDoubleQuote TokenType = "double_quote" // "
	TokenBacktick    TokenType = "backtick"     // `
	
	// Special
	TokenVariable   TokenType = "variable"   // $VAR or ${VAR}
	TokenGlob       TokenType = "glob"       // *, ?, [...]
	TokenSubshell   TokenType = "subshell"   // $(...) or (...)
	TokenBackground TokenType = "background" // &
	
	// Delimiters
	TokenNewline    TokenType = "newline"
	TokenEOF        TokenType = "eof"
	TokenLeftParen  TokenType = "left_paren"  // (
	TokenRightParen TokenType = "right_paren" // )
	TokenLeftBrace  TokenType = "left_brace"  // {
	TokenRightBrace TokenType = "right_brace" // }
	
	// Errors
	TokenError      TokenType = "error"
	TokenUnknown    TokenType = "unknown"
)

// Token represents a lexical token
type Token struct {
	Type     TokenType
	Value    string
	Position Position
}

// String returns a string representation of the token
func (t Token) String() string {
	if t.Value != "" {
		return fmt.Sprintf("%s(%s)", t.Type, t.Value)
	}
	return string(t.Type)
}

// IsOperator returns true if the token is an operator
func (t Token) IsOperator() bool {
	switch t.Type {
	case TokenPipe, TokenAmpersand, TokenSemicolon, TokenAnd, TokenOr:
		return true
	default:
		return false
	}
}

// IsRedirect returns true if the token is a redirection
func (t Token) IsRedirect() bool {
	switch t.Type {
	case TokenRedirectIn, TokenRedirectOut, TokenRedirectErr, 
	     TokenRedirectAppend, TokenRedirectHere:
		return true
	default:
		return false
	}
}

// IsQuote returns true if the token is a quote
func (t Token) IsQuote() bool {
	switch t.Type {
	case TokenSingleQuote, TokenDoubleQuote, TokenBacktick:
		return true
	default:
		return false
	}
}

// IsDelimiter returns true if the token is a delimiter
func (t Token) IsDelimiter() bool {
	switch t.Type {
	case TokenNewline, TokenEOF, TokenSemicolon:
		return true
	default:
		return false
	}
}

// Lexer interface for tokenizing commands
type Lexer interface {
	NextToken() (Token, error)
	PeekToken() (Token, error)
	Position() Position
}

// BaseLexer provides common lexing functionality
type BaseLexer struct {
	Input    string // Exported for embedded types
	input    string
	position int
	line     int
	column   int
}

// NewBaseLexer creates a new base lexer
func NewBaseLexer(input string) *BaseLexer {
	return &BaseLexer{
		Input:    input, // Set exported field
		input:    input,
		position: 0,
		line:     1,
		column:   1,
	}
}

// Position returns the current position
func (l *BaseLexer) Position() Position {
	return Position{
		Line:   l.line,
		Column: l.column,
		Offset: l.position,
	}
}

// Peek returns the next character without advancing (exported)
func (l *BaseLexer) Peek() byte {
	return l.peek()
}

// Advance moves to the next character (exported)
func (l *BaseLexer) Advance() byte {
	return l.advance()
}

// SkipWhitespace skips whitespace characters (exported)
func (l *BaseLexer) SkipWhitespace() {
	l.skipWhitespace()
}

// ReadWord reads a word token (exported)
func (l *BaseLexer) ReadWord() string {
	return l.readWord()
}

// ReadQuotedString reads a quoted string (exported)
func (l *BaseLexer) ReadQuotedString(quote byte) (string, error) {
	return l.readQuotedString(quote)
}

// peek returns the next character without advancing
func (l *BaseLexer) peek() byte {
	if l.position >= len(l.input) {
		return 0
	}
	return l.input[l.position]
}

// advance moves to the next character
func (l *BaseLexer) advance() byte {
	if l.position >= len(l.input) {
		return 0
	}
	ch := l.input[l.position]
	l.position++
	if ch == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	return ch
}

// skipWhitespace skips whitespace characters
func (l *BaseLexer) skipWhitespace() {
	for l.position < len(l.input) {
		ch := l.peek()
		if ch == ' ' || ch == '\t' || ch == '\r' {
			l.advance()
		} else {
			break
		}
	}
}

// readWord reads a word token
func (l *BaseLexer) readWord() string {
	start := l.position
	for l.position < len(l.input) {
		ch := l.peek()
		if isWordChar(ch) {
			l.advance()
		} else {
			break
		}
	}
	return l.input[start:l.position]
}

// readQuotedString reads a quoted string
func (l *BaseLexer) readQuotedString(quote byte) (string, error) {
	start := l.position
	l.advance() // Skip opening quote
	
	escaped := false
	for l.position < len(l.input) {
		ch := l.advance()
		if ch == 0 {
			return "", fmt.Errorf("unterminated string")
		}
		
		if escaped {
			escaped = false
			continue
		}
		
		if ch == '\\' && quote == '"' {
			escaped = true
			continue
		}
		
		if ch == quote {
			return l.input[start+1 : l.position-1], nil
		}
	}
	
	return "", fmt.Errorf("unterminated string")
}

// isWordChar returns true if the character can be part of a word
func isWordChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') ||
		ch == '_' || ch == '-' || ch == '.' || ch == '/'
}

// isOperatorChar returns true if the character is an operator
func isOperatorChar(ch byte) bool {
	switch ch {
	case '|', '&', ';', '<', '>', '(', ')', '{', '}':
		return true
	default:
		return false
	}
}