package validation

// AST represents the abstract syntax tree of a command
type AST struct {
	Root     Node
	Metadata map[string]interface{}
}

// NodeType categorizes AST nodes
type NodeType string

const (
	NodeCommand     NodeType = "command"
	NodePipeline    NodeType = "pipeline"
	NodeRedirect    NodeType = "redirect"
	NodeSubshell    NodeType = "subshell"
	NodeVariable    NodeType = "variable"
	NodeString      NodeType = "string"
	NodeGlob        NodeType = "glob"
	NodeBackground  NodeType = "background"
	NodeList        NodeType = "list"
	NodeAnd         NodeType = "and"
	NodeOr          NodeType = "or"
)

// Node represents a node in the AST
type Node interface {
	Type() NodeType
	Position() Position
	Children() []Node
	Validate() []ValidationError
	String() string
}

// BaseNode provides common node functionality
type BaseNode struct {
	NodeType NodeType
	Pos      Position
	ChildNodes []Node
}

func (n *BaseNode) Type() NodeType       { return n.NodeType }
func (n *BaseNode) Position() Position   { return n.Pos }
func (n *BaseNode) Children() []Node     { return n.ChildNodes }
func (n *BaseNode) Validate() []ValidationError { return []ValidationError{} }
func (n *BaseNode) String() string       { return string(n.NodeType) }

// CommandNode represents a single command
type CommandNode struct {
	BaseNode
	Name      string
	Args      []string
	Redirects []RedirectNode
	Pos       Position // Direct position field for compatibility
}

// PipelineNode represents a command pipeline
type PipelineNode struct {
	BaseNode
	Commands []Node
}

// RedirectNode represents input/output redirection
type RedirectNode struct {
	BaseNode
	Type     RedirectType
	Fd       int    // File descriptor
	Target   string // Target file or descriptor
	Append   bool   // >> vs >
}

// RedirectType specifies the type of redirection
type RedirectType string

const (
	RedirectInput  RedirectType = "input"   // <
	RedirectOutput RedirectType = "output"  // >
	RedirectError  RedirectType = "error"   // 2>
	RedirectAppend RedirectType = "append"  // >>
	RedirectHere   RedirectType = "here"    // <<
)

// VariableNode represents a variable reference
type VariableNode struct {
	BaseNode
	Name         string
	Value        string
	IsAssignment bool
	IsExport     bool
}

// StringNode represents a quoted string
type StringNode struct {
	BaseNode
	Value      string
	QuoteType  QuoteType
	Expansions []Node // Variable expansions within the string
}

// QuoteType specifies the type of quoting
type QuoteType string

const (
	QuoteSingle QuoteType = "single" // '...'
	QuoteDouble QuoteType = "double" // "..."
	QuoteNone   QuoteType = "none"   // unquoted
)

// SubshellNode represents a subshell execution
type SubshellNode struct {
	BaseNode
	Command Node
	IsCommand bool // $(...) vs (...)
}

// ListNode represents a command list
type ListNode struct {
	BaseNode
	Commands  []Node
	Separator ListSeparator
	Children  []Node // Direct children field for compatibility
}

// ListSeparator specifies how commands are separated
type ListSeparator string

const (
	SeparatorSemicolon  ListSeparator = ";"  // Sequential execution
	SeparatorAmpersand  ListSeparator = "&"  // Background execution
	SeparatorAnd        ListSeparator = "&&" // Conditional AND
	SeparatorOr         ListSeparator = "||" // Conditional OR
)

// GlobNode represents a glob pattern
type GlobNode struct {
	BaseNode
	Pattern string
}

// NewCommandNode creates a new command node
func NewCommandNode(name string, args []string, pos Position) *CommandNode {
	return &CommandNode{
		BaseNode: BaseNode{
			NodeType: NodeCommand,
			Pos:      pos,
		},
		Name: name,
		Args: args,
	}
}

// NewPipelineNode creates a new pipeline node
func NewPipelineNode(commands []Node, pos Position) *PipelineNode {
	return &PipelineNode{
		BaseNode: BaseNode{
			NodeType:   NodePipeline,
			Pos:        pos,
			ChildNodes: commands,
		},
		Commands: commands,
	}
}

// NewStringNode creates a new string node
func NewStringNode(value string, quoteType QuoteType, pos Position) *StringNode {
	return &StringNode{
		BaseNode: BaseNode{
			NodeType: NodeString,
			Pos:      pos,
		},
		Value:     value,
		QuoteType: quoteType,
	}
}

// NewVariableNode creates a new variable node
func NewVariableNode(name string, pos Position) *VariableNode {
	return &VariableNode{
		BaseNode: BaseNode{
			NodeType: NodeVariable,
			Pos:      pos,
		},
		Name: name,
	}
}

// Walk traverses the AST and calls the visitor function for each node
func Walk(node Node, visitor func(Node) bool) {
	if node == nil || !visitor(node) {
		return
	}
	
	for _, child := range node.Children() {
		Walk(child, visitor)
	}
}

// FindNodes finds all nodes of a specific type in the AST
func FindNodes(root Node, nodeType NodeType) []Node {
	var nodes []Node
	Walk(root, func(n Node) bool {
		if n.Type() == nodeType {
			nodes = append(nodes, n)
		}
		return true
	})
	return nodes
}

// Validate methods for specific node types

func (n *CommandNode) Validate() []ValidationError {
	errors := []ValidationError{}
	
	// Check for empty command
	if n.Name == "" {
		errors = append(errors, ValidationError{
			Type:     ErrorSyntax,
			Severity: SeverityError,
			Position: n.Pos,
			Message:  "Empty command",
		})
	}
	
	return errors
}

func (n *PipelineNode) Validate() []ValidationError {
	errors := []ValidationError{}
	
	// Check for empty pipeline
	if len(n.Commands) == 0 {
		errors = append(errors, ValidationError{
			Type:     ErrorSyntax,
			Severity: SeverityError,
			Position: n.Pos,
			Message:  "Empty pipeline",
		})
	}
	
	// Check for trailing pipe
	if len(n.Commands) > 0 {
		lastCmd := n.Commands[len(n.Commands)-1]
		if lastCmd == nil {
			errors = append(errors, ValidationError{
				Type:     ErrorSyntax,
				Severity: SeverityError,
				Position: n.Pos,
				Message:  "Unexpected end of command after pipe",
			})
		}
	}
	
	return errors
}

func (n *StringNode) Validate() []ValidationError {
	errors := []ValidationError{}
	
	// For now, basic validation only
	// More complex quote matching will be in the parser
	
	return errors
}

func (n *RedirectNode) Validate() []ValidationError {
	errors := []ValidationError{}
	
	// Check for empty redirect target
	if n.Target == "" {
		errors = append(errors, ValidationError{
			Type:     ErrorSyntax,
			Severity: SeverityError,
			Position: n.Pos,
			Message:  "Missing redirect target",
		})
	}
	
	return errors
}