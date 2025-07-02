package main

import "fmt"

// showEnhancedHelp displays an enhanced help message with all available Delta commands
func showEnhancedHelp() {
	fmt.Println("Delta (∆) CLI Internal Commands:")

	// AI Commands
	fmt.Println("  AI Assistant:")
	fmt.Println("  :ai [on|off]      - Enable or disable AI assistant")
	fmt.Println("  :ai model <n>     - Change AI model (e.g., phi4:latest)")
	fmt.Println("  :ai model custom <path> - Use custom trained model")
	fmt.Println("  :ai feedback <type> [correction] - Provide feedback on predictions (helpful|unhelpful|correction)")
	fmt.Println("  :ai status        - Show AI assistant status")

	// Jump Commands
	fmt.Println("")
	fmt.Println("  Navigation:")
	fmt.Println("  :jump <location>  - Jump to predefined location")
	fmt.Println("  :jump add <n> [path] - Add a new jump location")
	fmt.Println("  :jump remove <n>     - Remove a jump location")
	fmt.Println("  :jump import jumpsh  - Import locations from jump.sh")
	fmt.Println("  :j <location>     - Shorthand for jump")

	// Memory Commands
	fmt.Println("")
	fmt.Println("  Memory System:")
	fmt.Println("  :memory status    - Show memory system status")
	fmt.Println("  :memory enable    - Enable memory collection")
	fmt.Println("  :memory disable   - Disable memory collection")
	fmt.Println("  :memory stats     - Show detailed memory statistics")
	fmt.Println("  :memory config    - View or modify memory configuration")
	fmt.Println("  :memory list      - List available data shards")
	fmt.Println("  :memory export    - Export data for a specific date")
	fmt.Println("  :memory clear     - Clear all collected data (requires confirmation)")
	fmt.Println("  :memory train     - Memory training commands")
	fmt.Println("  :mem              - Shorthand for memory commands")

	// Tokenizer Commands
	fmt.Println("")
	fmt.Println("  Tokenizer System:")
	fmt.Println("  :tokenizer status - Show tokenizer status")
	fmt.Println("  :tokenizer stats  - Show detailed tokenizer statistics")
	fmt.Println("  :tokenizer process - Process command data for training")
	fmt.Println("  :tokenizer vocab  - Show vocabulary information")
	fmt.Println("  :tokenizer test   - Test tokenization on sample commands")
	fmt.Println("  :tok              - Shorthand for tokenizer commands")

	// Inference and Learning Commands
	fmt.Println("")
	fmt.Println("  Inference and Learning:")
	fmt.Println("  :inference [enable|disable] - Control inference system")
	fmt.Println("  :inference feedback <type>  - Provide feedback on predictions (helpful|unhelpful|correction)")
	fmt.Println("  :inference stats   - Show detailed inference statistics")
	fmt.Println("  :inference examples - Show training examples")
	fmt.Println("  :inference model   - Manage custom models")
	fmt.Println("  :inference config  - Configure inference system")
	fmt.Println("  :inf               - Shorthand for inference commands")
	fmt.Println("  :feedback <type>   - Shorthand for feedback commands (helpful|unhelpful|correction)")

	// Training Commands
	fmt.Println("")
	fmt.Println("  Training Data:")
	fmt.Println("  :training              - Show training data status")
	fmt.Println("  :training extract      - Extract training data with options")
	fmt.Println("  :training stats        - Show detailed training statistics")
	fmt.Println("  :training evaluate     - Evaluate training data quality")
	fmt.Println("  :train                 - Shorthand for training commands")

	// Learning System Commands
	fmt.Println("")
	fmt.Println("  Learning System:")
	fmt.Println("  :learning              - Show learning system status")
	fmt.Println("  :learning enable       - Enable learning from commands")
	fmt.Println("  :learning disable      - Disable learning from commands")
	fmt.Println("  :learning feedback     - Provide interactive feedback")
	fmt.Println("  :learning train        - Manage training pipeline")
	fmt.Println("  :learning patterns     - Show learned command patterns")
	fmt.Println("  :learning process      - Process learning data manually")
	fmt.Println("  :learning stats        - Show learning statistics")
	fmt.Println("  :learning config       - Configure learning settings")
	fmt.Println("  :learn                 - Shorthand for learning commands")

	// Vector Database Commands
	fmt.Println("")
	fmt.Println("  Vector Database:")
	fmt.Println("  :vector [enable|disable] - Control vector database")
	fmt.Println("  :vector search <cmd>  - Search for similar commands")
	fmt.Println("  :vector embed <cmd>   - Generate embedding for a command")
	fmt.Println("  :vector stats         - Show detailed vector database statistics")
	fmt.Println("  :vector config        - Configure vector database")

	// Embedding Commands
	fmt.Println("")
	fmt.Println("  Embedding System:")
	fmt.Println("  :embedding [enable|disable] - Control embedding system")
	fmt.Println("  :embedding generate <cmd>   - Generate embedding for a command")
	fmt.Println("  :embedding stats            - Show detailed embedding statistics")
	fmt.Println("  :embedding config           - Configure embedding system")

	// Speculative Decoding Commands
	fmt.Println("")
	fmt.Println("  Speculative Decoding:")
	fmt.Println("  :speculative [enable|disable] - Control speculative decoding")
	fmt.Println("  :speculative draft <text>     - Test speculative drafting")
	fmt.Println("  :speculative stats            - Show detailed statistics")
	fmt.Println("  :speculative config           - Configure speculative decoding")
	fmt.Println("  :specd                        - Shorthand for speculative commands")

	// Knowledge Extraction Commands
	fmt.Println("")
	fmt.Println("  Knowledge Extraction:")
	fmt.Println("  :knowledge [enable|disable]   - Control knowledge extraction")
	fmt.Println("  :knowledge query <text>       - Search for knowledge")
	fmt.Println("  :knowledge context            - Show current environment context")
	fmt.Println("  :knowledge scan               - Scan current directory for knowledge")
	fmt.Println("  :knowledge project            - Show project information")
	fmt.Println("  :knowledge stats              - Show detailed statistics")
	fmt.Println("  :know                         - Shorthand for knowledge commands")

	// Agent System Commands
	fmt.Println("")
	fmt.Println("  Agent System:")
	fmt.Println("  :agent [enable|disable]       - Control agent manager")
	fmt.Println("  :agent list                   - List all agents")
	fmt.Println("  :agent show <id>              - Show agent details")
	fmt.Println("  :agent run <id>               - Run an agent")
	fmt.Println("  :agent create <n>          - Create a new agent")
	fmt.Println("  :agent edit <id>              - Edit agent configuration")
	fmt.Println("  :agent delete <id>            - Delete an agent")
	fmt.Println("  :agent learn <cmds>           - Learn a new agent from command sequence")
	fmt.Println("  :agent docker [list|cache|build] - Manage Docker integration")
	fmt.Println("  :agent stats                  - Show agent statistics")

	// Configuration Commands
	fmt.Println("")
	fmt.Println("  Configuration:")
	fmt.Println("  :config                - Show configuration status")
	fmt.Println("  :config list           - List all configurations")
	fmt.Println("  :config export <path>  - Export configuration to a file")
	fmt.Println("  :config import <path>  - Import configuration from a file")
	fmt.Println("  :config edit <comp>    - View or modify component configuration")
	fmt.Println("  :config reset          - Reset all configurations to default values")

	// Update System Commands
	fmt.Println("")
	fmt.Println("  Update System:")
	fmt.Println("  :update                - Show update status")
	fmt.Println("  :update status         - Show detailed update status")
	fmt.Println("  :update config         - Show update configuration")
	fmt.Println("  :update config <k> <v> - Set update configuration value")
	fmt.Println("  :update version        - Show version information")

	// Spell Checker Commands
	fmt.Println("")
	fmt.Println("  Spell Checker:")
	fmt.Println("  :spellcheck [enable|disable] - Control spell checking")
	fmt.Println("  :spellcheck status     - Show spell checker status")
	fmt.Println("  :spellcheck config     - Configure spell checker")
	fmt.Println("  :spellcheck add <word> - Add word to custom dictionary")
	fmt.Println("  :spellcheck remove <word> - Remove word from dictionary")
	fmt.Println("  :spellcheck test <cmd> - Test spell checking on a command")
	fmt.Println("  :spell                 - Shorthand for spellcheck commands")

	// History Analysis Commands
	fmt.Println("")
	fmt.Println("  History Analysis:")
	fmt.Println("  :history               - Show recent command history")
	fmt.Println("  :history show [limit]  - Show history with limit")
	fmt.Println("  :history search <query> - Search command history")
	fmt.Println("  :history suggest       - Show command suggestions")
	fmt.Println("  :history stats         - Show detailed statistics")
	fmt.Println("  :history patterns      - Show command patterns")
	fmt.Println("  :history info <cmd>    - Show info about a command")
	fmt.Println("  :history config        - Configure history analysis")
	fmt.Println("  :hist                  - Shorthand for history commands")

	// Command Suggestions
	fmt.Println("")
	fmt.Println("  Command Suggestions:")
	fmt.Println("  :suggest <description> - Get command suggestions from natural language")
	fmt.Println("  :suggest explain <cmd> - Explain what a command does")
	fmt.Println("  :suggest last          - Show last suggestions")
	fmt.Println("  :suggest clear         - Clear suggestion cache")
	fmt.Println("  :s                     - Shorthand for suggest")

	// Command Validation
	fmt.Println("")
	fmt.Println("  Command Validation:")
	fmt.Println("  :validate <command>    - Check command syntax and safety")
	fmt.Println("  :validation safety <cmd> - Analyze command safety risks")
	fmt.Println("  :validation config     - Configure validation settings")
	fmt.Println("  :v                     - Shorthand for validate")

	// Documentation Commands
	fmt.Println("")
	fmt.Println("  Documentation:")
	fmt.Println("  :docs              - Build and open web documentation")
	fmt.Println("  :docs build        - Build documentation")
	fmt.Println("  :docs dev          - Start documentation dev server")
	fmt.Println("  :man               - Manage man pages")
	fmt.Println("  :man generate      - Generate man pages")
	fmt.Println("  :man install       - Install man pages to system")
	fmt.Println("  :man view          - View installed man pages")

	// Other Commands
	fmt.Println("")
	fmt.Println("  System:")
	fmt.Println("  :init             - Initialize configuration files")
	fmt.Println("  :help             - Show this help message")

	// Shell Navigation
	fmt.Println("")
	fmt.Println("Shell Navigation:")
	fmt.Println("  cd [directory]    - Change current directory")
	fmt.Println("  pwd               - Print current working directory")
	fmt.Println("  exit, quit        - Exit Delta shell")
	fmt.Println("  sub               - Enter subcommand mode")
	fmt.Println("  end               - Exit subcommand mode")
}
