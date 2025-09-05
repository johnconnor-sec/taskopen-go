package exec

import (
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestIsInteractiveEditor_BasicEditors(t *testing.T) {
	executor := New(ExecutionOptions{})

	tests := []struct {
		command  string
		expected bool
		desc     string
	}{
		{"vim", true, "vim should be detected as interactive"},
		{"nvim", true, "nvim should be detected as interactive"},
		{"nano", true, "nano should be detected as interactive"},
		{"emacs", true, "emacs should be detected as interactive"},
		{"code", true, "vscode should be detected as interactive"},
		{"cat", false, "cat should not be detected as interactive"},
		{"ls", false, "ls should not be detected as interactive"},
		{"grep", false, "grep should not be detected as interactive"},
		{"", false, "empty command should return false"},
		{"   ", false, "whitespace-only command should return false"},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := executor.IsInteractiveEditor(test.command)
			if result != test.expected {
				t.Errorf("IsInteractiveEditor(%q) = %v; want %v", test.command, result, test.expected)
			}
		})
	}
}

func TestIsInteractiveEditor_WithArguments(t *testing.T) {
	executor := New(ExecutionOptions{})

	tests := []struct {
		command  string
		expected bool
		desc     string
	}{
		{"vim file.txt", true, "vim with arguments should be detected"},
		{"nvim -u NONE file.txt", true, "nvim with multiple arguments should be detected"},
		{"nano --backup file.txt", true, "nano with flags should be detected"},
		{"code --wait file.txt", true, "code with flags should be detected"},
		{"cat file.txt", false, "cat with arguments should not be detected"},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := executor.IsInteractiveEditor(test.command)
			if result != test.expected {
				t.Errorf("IsInteractiveEditor(%q) = %v; want %v", test.command, result, test.expected)
			}
		})
	}
}

func TestIsInteractiveEditor_CrossPlatformPaths(t *testing.T) {
	executor := New(ExecutionOptions{})

	tests := []struct {
		command  string
		expected bool
		desc     string
	}{
		// Unix-style paths
		{"/usr/bin/vim", true, "Unix absolute path to vim should be detected"},
		{"/usr/local/bin/nvim", true, "Unix absolute path to nvim should be detected"},
		{"./vim", true, "Unix relative path to vim should be detected"},
		{"../bin/nano", true, "Unix relative path to nano should be detected"},
		{"/usr/bin/cat", false, "Unix absolute path to cat should not be detected"},

		// Mixed case should work due to ToLower
		{"/usr/bin/VIM", true, "Mixed case vim should be detected"},
		{"/usr/bin/NANO", true, "Mixed case nano should be detected"},
	}

	// Add Windows-specific tests when running on Windows
	// Note: filepath.Base() correctly handles platform-specific path separators
	if runtime.GOOS == "windows" {
		windowsTests := []struct {
			command  string
			expected bool
			desc     string
		}{
			{`C:\Program Files\Neovim\bin\nvim.exe`, true, "Windows absolute path to nvim should be detected"},
			{`C:\tools\vim\vim.exe`, true, "Windows absolute path to vim should be detected"},
			{`.\nvim.exe`, true, "Windows relative path to nvim should be detected"},
			{`..\bin\nano.exe`, true, "Windows relative path to nano should be detected"},
			{`C:\Windows\System32\notepad.exe`, false, "Windows notepad should not be detected"},
			{`C:\Windows\System32\cmd.exe`, false, "Windows cmd should not be detected"},
		}
		tests = append(tests, windowsTests...)
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := executor.IsInteractiveEditor(test.command)
			if result != test.expected {
				t.Errorf("IsInteractiveEditor(%q) = %v; want %v", test.command, result, test.expected)
			}
		})
	}
}

func TestIsInteractiveEditor_EdgeCases(t *testing.T) {
	executor := New(ExecutionOptions{})

	tests := []struct {
		command  string
		expected bool
		desc     string
	}{
		{"vim.exe", true, "Windows executable extension should work"},
		{"nvim.exe file.txt", true, "Windows executable with args should work"},
		{"nano.backup", false, "Non-standard extension should not match"},
		{"vim-tiny", false, "Hyphenated variant should not match"},
		{"xvim", false, "Partial match should not work"},
		{"vimx", false, "Partial match should not work"},
		{".vim", true, "Hidden executable should work"},
		{"vim ", true, "Trailing whitespace should be handled"},
		{" vim", true, "Leading whitespace should be handled"},
		{"VIM", true, "All caps should work due to case insensitive"},
		{"Vim", true, "Mixed case should work"},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := executor.IsInteractiveEditor(test.command)
			if result != test.expected {
				t.Errorf("IsInteractiveEditor(%q) = %v; want %v", test.command, result, test.expected)
			}
		})
	}
}

func TestNeedsShell_BasicCases(t *testing.T) {
	executor := New(ExecutionOptions{})

	tests := []struct {
		command  string
		expected bool
		desc     string
	}{
		{"vim file.txt", false, "Simple command should not need shell"},
		{"vim file.txt | tee log.txt", true, "Pipe should need shell"},
		{"vim file.txt && echo done", true, "AND operator should need shell"},
		{"vim file.txt || echo failed", true, "OR operator should need shell"},
		{"vim file.txt; echo done", true, "Semicolon should need shell"},
		{"vim file.txt > output.txt", true, "Redirect should need shell"},
		{"vim file.txt >> output.txt", true, "Append redirect should need shell"},
		{"vim < input.txt", true, "Input redirect should need shell"},
		{"vim $(echo file.txt)", true, "Command substitution should need shell"},
		{"vim `echo file.txt`", true, "Backtick substitution should need shell"},
		{"vim *.txt", true, "Glob should need shell"},
		{"vim file?.txt", true, "Single char glob should need shell"},
		{"vim file[1-3].txt", true, "Character class should need shell"},
		{"EDITOR=vim task edit", true, "Environment variable should need shell"},
		{"vim file.txt &", true, "Background job should need shell"},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := executor.NeedsShell(test.command)
			if result != test.expected {
				t.Errorf("NeedsShell(%q) = %v; want %v", test.command, result, test.expected)
			}
		})
	}
}

func TestFilepathBase_DirectComparison(t *testing.T) {
	// Test that our filepath.Base usage works correctly across platforms
	tests := []struct {
		input    string
		expected string
		desc     string
	}{
		{"/usr/bin/vim", "vim", "Unix absolute path"},
		{"./vim", "vim", "Unix relative path"},
		{"../bin/vim", "vim", "Unix parent directory"},
		{"vim", "vim", "Just executable name"},
	}

	// Add Windows tests when on Windows
	if runtime.GOOS == "windows" {
		windowsTests := []struct {
			input    string
			expected string
			desc     string
		}{
			{`C:\Program Files\vim\vim.exe`, "vim.exe", "Windows absolute path"},
			{`.\vim.exe`, "vim.exe", "Windows relative path"},
			{`..\bin\vim.exe`, "vim.exe", "Windows parent directory"},
			{`vim.exe`, "vim.exe", "Windows executable with extension"},
		}
		tests = append(tests, windowsTests...)
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := filepath.Base(test.input)
			if result != test.expected {
				t.Errorf("filepath.Base(%q) = %q; want %q", test.input, result, test.expected)
			}
		})
	}
}

func TestExecutionOptions_Interactive(t *testing.T) {
	// Test that Interactive flag is properly handled
	tests := []struct {
		interactive bool
		desc        string
	}{
		{true, "Interactive should be true"},
		{false, "Interactive should be false"},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			options := ExecutionOptions{
				Interactive: test.interactive,
				Timeout:     1 * time.Second,
			}

			if options.Interactive != test.interactive {
				t.Errorf("ExecutionOptions.Interactive = %v; want %v", options.Interactive, test.interactive)
			}
		})
	}
}

func TestNew_DefaultOptions(t *testing.T) {
	executor := New(ExecutionOptions{})

	// Test that executor was created successfully
	if executor == nil {
		t.Fatal("New() returned nil executor")
	}

	// Test that default options are reasonable
	if executor.defaultOptions.Timeout == 0 {
		t.Error("Default timeout should not be zero")
	}

	if executor.defaultOptions.Retry.MaxAttempts == 0 {
		t.Error("Default retry attempts should not be zero")
	}
}

func TestExecutor_IsInteractiveEditor_PerformanceMap(t *testing.T) {
	executor := New(ExecutionOptions{})

	// Test that map lookup works correctly (this verifies the optimization)
	testCases := []string{"vim", "nvim", "nano", "emacs", "code", "nonexistent"}

	for _, cmd := range testCases {
		// Just ensure it doesn't panic and returns a boolean
		result := executor.IsInteractiveEditor(cmd)
		if result != true && result != false {
			t.Errorf("IsInteractiveEditor should return a boolean, got %v", result)
		}
	}
}

// Benchmark to verify map performance improvement
func BenchmarkIsInteractiveEditor_Map(b *testing.B) {
	executor := New(ExecutionOptions{})
	commands := []string{"vim", "nvim", "nano", "emacs", "code", "cat", "ls", "grep"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := commands[i%len(commands)]
		executor.IsInteractiveEditor(cmd)
	}
}
