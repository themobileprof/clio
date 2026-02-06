package modules

import (
	"bufio"
	"clio/internal/safeexec"
	"clio/internal/setup"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// FullModuleYAML represents the complete module structure including flows
type FullModuleYAML struct {
	Name           string   `yaml:"name"`
	ID             string   `yaml:"id"`
	Version        string   `yaml:"version"`
	Description    string   `yaml:"description"`
	Tags           []string `yaml:"tags"`
	RequiresTermux bool     `yaml:"requires_termux"`
	EstimatedTime  string   `yaml:"estimated_time"`
	Flows          []Flow   `yaml:"flows"`
}

// Flow represents a workflow with multiple steps
type Flow struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Steps       []Step `yaml:"steps"`
}

// Step represents a single action in a flow
type Step struct {
	Type            string `yaml:"type"`
	Content         string `yaml:"content"`
	Prompt          string `yaml:"prompt"`
	Default         string `yaml:"default"`
	OnNo            string `yaml:"on_no"`
	OnYes           string `yaml:"on_yes"`
	OnExists        string `yaml:"on_exists"`
	OnMissing       string `yaml:"on_missing"`
	Command         string `yaml:"command"`
	Description     string `yaml:"description"`
	ShowOutput      bool   `yaml:"show_output"`
	Interactive     bool   `yaml:"interactive"`
	ContinueOnError bool   `yaml:"continue_on_error"`
	Title           string `yaml:"title"`
	Steps           []Step `yaml:"steps"` // For sections
	Path            string `yaml:"path"`
	Name            string `yaml:"name"`  // For labels
	Label           string `yaml:"label"` // For goto
	Operation       string `yaml:"operation"`
	Variable        string `yaml:"variable"`
	Required        bool   `yaml:"required"`
	Condition       string `yaml:"condition"`
}

// ExecutionContext holds runtime state for module execution
type ExecutionContext struct {
	Variables map[string]string
	Scanner   *bufio.Scanner
	Labels    map[string]int // Map label names to step indices
}

// LoadModule loads and parses a module from YAML content
func LoadModule(yamlContent string) (*FullModuleYAML, error) {
	var module FullModuleYAML
	if err := yaml.Unmarshal([]byte(yamlContent), &module); err != nil {
		return nil, fmt.Errorf("yaml parse error: %w", err)
	}
	return &module, nil
}

// ExecuteModule runs a module's flow
func ExecuteModule(module *FullModuleYAML, flowName string, scanner *bufio.Scanner) error {
	// Find the flow
	var flow *Flow
	for i := range module.Flows {
		if module.Flows[i].Name == flowName {
			flow = &module.Flows[i]
			break
		}
	}

	if flow == nil {
		return fmt.Errorf("flow '%s' not found in module", flowName)
	}

	// Check Termux requirement
	if module.RequiresTermux && !setup.IsTermux() {
		return fmt.Errorf("this module requires Termux")
	}

	ctx := &ExecutionContext{
		Variables: make(map[string]string),
		Scanner:   scanner,
		Labels:    buildLabelMap(flow.Steps),
	}

	return executeSteps(flow.Steps, ctx, 0, 0)
}

// buildLabelMap creates a map of label names to step indices (depth-first)
func buildLabelMap(steps []Step) map[string]int {
	labels := make(map[string]int)
	buildLabelMapRecursive(steps, labels, 0)
	return labels
}

func buildLabelMapRecursive(steps []Step, labels map[string]int, baseIndex int) int {
	currentIndex := baseIndex
	for _, step := range steps {
		if step.Type == "label" {
			labels[step.Name] = currentIndex
		}
		currentIndex++

		// Recurse into section steps
		if step.Type == "section" && len(step.Steps) > 0 {
			currentIndex = buildLabelMapRecursive(step.Steps, labels, currentIndex)
		}
	}
	return currentIndex
}

// executeSteps executes a list of steps
func executeSteps(steps []Step, ctx *ExecutionContext, sectionNum, totalSections int) error {
	for i := 0; i < len(steps); i++ {
		step := steps[i]

		if err := executeStep(&step, ctx, sectionNum, totalSections); err != nil {
			if err.Error() == "abort" {
				return fmt.Errorf("setup cancelled")
			}
			if err.Error() == "skip" {
				continue
			}
			if strings.HasPrefix(err.Error(), "goto:") {
				// Handle goto
				label := strings.TrimPrefix(err.Error(), "goto:")
				if idx, ok := ctx.Labels[label]; ok {
					i = idx - 1 // -1 because loop will increment
					continue
				}
				return fmt.Errorf("label not found: %s", label)
			}
			return err
		}
	}
	return nil
}

// executeStep executes a single step
func executeStep(step *Step, ctx *ExecutionContext, sectionNum, totalSections int) error {
	switch step.Type {
	case "message":
		content := expandTemplate(step.Content, ctx.Variables)
		fmt.Println(content)

	case "confirm":
		prompt := step.Prompt
		if step.Default != "" {
			if step.Default == "yes" || step.Default == "y" {
				prompt += " [Y/n]: "
			} else {
				prompt += " [y/N]: "
			}
		} else {
			prompt += " [y/n]: "
		}

		fmt.Print(prompt)
		if !ctx.Scanner.Scan() {
			return fmt.Errorf("input error")
		}

		answer := strings.ToLower(strings.TrimSpace(ctx.Scanner.Text()))

		// Determine result
		isYes := false
		if answer == "" {
			isYes = (step.Default == "yes" || step.Default == "y")
		} else {
			isYes = (answer == "y" || answer == "yes")
		}

		if !isYes && step.OnNo != "" {
			if step.OnNo == "abort" {
				return errors.New("abort")
			}
			return errors.New("goto:" + step.OnNo)
		}

		if isYes && step.OnYes != "" {
			return errors.New("goto:" + step.OnYes)
		}

	case "input":
		fmt.Print(step.Prompt + ": ")
		if !ctx.Scanner.Scan() {
			return fmt.Errorf("input error")
		}

		value := strings.TrimSpace(ctx.Scanner.Text())
		if value == "" && step.Required {
			fmt.Println("This field is required.")
			return executeStep(step, ctx, sectionNum, totalSections) // Retry
		}

		if step.Variable != "" {
			ctx.Variables[step.Variable] = value
		}

	case "command":
		// Check condition
		if step.Condition != "" {
			condition := expandTemplate(step.Condition, ctx.Variables)
			if condition == "" || condition == "false" {
				return nil // Skip this command
			}
		}

		if step.Description != "" {
			fmt.Println(step.Description + "...")
		}

		cmdStr := expandTemplate(step.Command, ctx.Variables)

		var cmd *exec.Cmd
		if step.Interactive {
			cmd = exec.Command("sh", "-c", cmdStr)
			cmd.Stdin = os.Stdin
		} else {
			cmd = safeexec.Command("sh", "-c", cmdStr)
		}

		if step.ShowOutput || step.Interactive {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}

		if err := cmd.Run(); err != nil {
			if step.ContinueOnError {
				fmt.Printf("⚠️  Warning: %v\n", err)
				return nil
			}
			return fmt.Errorf("command failed: %w", err)
		}

	case "section":
		fmt.Printf("\n[%d/%d] %s\n", sectionNum+1, totalSections, step.Title)
		fmt.Println(strings.Repeat("─", 60))

		if err := executeSteps(step.Steps, ctx, sectionNum, totalSections); err != nil {
			if step.ContinueOnError {
				fmt.Printf("⚠️  Warning: %s failed: %v\n", step.Title, err)
				fmt.Print("Continue anyway? [Y/n]: ")
				if ctx.Scanner.Scan() {
					ans := strings.ToLower(strings.TrimSpace(ctx.Scanner.Text()))
					if ans == "n" || ans == "no" {
						return fmt.Errorf("aborted")
					}
				}
				return nil
			}
			return err
		}

		fmt.Printf("✅ %s complete\n", step.Title)

	case "check_command":
		if _, err := safeexec.LookPath(step.Command); err != nil {
			if step.OnMissing != "" {
				if step.OnMissing == "skip" {
					return errors.New("skip")
				}
				return errors.New("goto:" + step.OnMissing)
			}
		} else {
			if step.OnExists != "" {
				return errors.New("goto:" + step.OnExists)
			}
		}

	case "check_path":
		if _, err := os.Stat(expandPath(step.Path)); err == nil {
			if step.OnExists != "" {
				if step.OnExists == "skip" {
					return errors.New("skip")
				}
				return errors.New("goto:" + step.OnExists)
			}
		} else {
			if step.OnMissing != "" {
				return errors.New("goto:" + step.OnMissing)
			}
		}

	case "label":
		// Labels are no-ops during execution

	case "goto":
		return errors.New("goto:" + step.Label)

	case "file_operation":
		if step.Description != "" {
			fmt.Println(step.Description + "...")
		}

		switch step.Operation {
		case "create_vimrc":
			if err := setup.CreateVimConfig(); err != nil {
				return fmt.Errorf("vim config failed: %w", err)
			}
		case "configure_zshrc":
			if err := configureZshrc(); err != nil {
				return fmt.Errorf("zshrc config failed: %w", err)
			}
		case "mark_complete":
			if err := setup.MarkSetupComplete(); err != nil {
				fmt.Printf("Warning: Could not mark setup as complete: %v\n", err)
			}
		default:
			return fmt.Errorf("unknown file operation: %s", step.Operation)
		}

	default:
		return fmt.Errorf("unknown step type: %s", step.Type)
	}

	return nil
}

// expandTemplate replaces {{.Variable}} with values from context
func expandTemplate(text string, vars map[string]string) string {
	tmpl, err := template.New("expand").Parse(text)
	if err != nil {
		return text
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, vars); err != nil {
		return text
	}

	return buf.String()
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return strings.Replace(path, "~", home, 1)
	}
	return path
}

// configureZshrc configures the .zshrc file
func configureZshrc() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	zshrcPath := home + "/.zshrc"
	content, err := os.ReadFile(zshrcPath)
	if err != nil {
		// File doesn't exist, create basic one
		content = []byte(`# Zsh configuration
export ZSH="$HOME/.oh-my-zsh"
ZSH_THEME="powerlevel10k/powerlevel10k"
plugins=(git z sudo colored-man-pages)
source $ZSH/oh-my-zsh.sh
`)
	}

	contentStr := string(content)

	// Update theme if not already set
	if !strings.Contains(contentStr, "powerlevel10k") {
		contentStr = strings.Replace(contentStr,
			`ZSH_THEME="robbyrussell"`,
			`ZSH_THEME="powerlevel10k/powerlevel10k"`, 1)
	}

	// Add useful aliases if not present
	if !strings.Contains(contentStr, "alias ll=") {
		contentStr += "\n# Useful aliases\n"
		contentStr += `alias ll="ls -lah"` + "\n"
		contentStr += `alias la="ls -A"` + "\n"
		contentStr += `alias l="ls -CF"` + "\n"
		contentStr += `alias ..="cd .."` + "\n"
		contentStr += `alias ...="cd ../.."` + "\n"
	}

	return os.WriteFile(zshrcPath, []byte(contentStr), 0644)
}

// CountSections counts the number of section steps
func CountSections(steps []Step) int {
	count := 0
	for _, step := range steps {
		if step.Type == "section" {
			count++
		}
	}
	return count
}
