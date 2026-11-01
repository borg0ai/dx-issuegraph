package ui

import "strings"

// richOutput reports whether styled message output is on. It is a variable so
// tests can force either mode.
var richOutput = func() bool { return ShouldUseColor() && !IsAgentMode() }

// Rich reports whether commands should print styled messages. It is false for
// NO_COLOR, CLICOLOR=0, non-TTY output, git hooks and agent mode, where the
// helpers below return the plain text unchanged.
func Rich() bool { return richOutput() }

// Banner renders a command title with a rule underneath.
func Banner(title string) string {
	if !Rich() {
		return title + "\n" + strings.Repeat("=", len(title))
	}
	return AccentStyle.Bold(true).Render(title) + "\n" + MutedStyle.Render(strings.Repeat("━", len([]rune(title))))
}

// Section renders a group heading.
func Section(title string) string {
	if !Rich() {
		return title + ":"
	}
	return AccentStyle.Bold(true).Render(title)
}

// Field renders a label and value pair. Callers add any indent.
func Field(label, value string) string {
	if !Rich() {
		return label + ": " + value
	}
	return MutedStyle.Render(label+":") + " " + BoldStyle.Render(value)
}

// Success renders a success message with a green check.
func Success(msg string) string {
	if !Rich() {
		return msg
	}
	return RenderPassIcon() + " " + PassStyle.Render(msg)
}

// Warn renders a warning with an amber icon.
func Warn(msg string) string {
	if !Rich() {
		return msg
	}
	return RenderWarnIcon() + " " + WarnStyle.Render(msg)
}

// Error renders an error with a red icon.
func Error(msg string) string {
	if !Rich() {
		return msg
	}
	return RenderFailIcon() + " " + FailStyle.Render(msg)
}

// Info renders an informational message with a blue icon.
func Info(msg string) string {
	if !Rich() {
		return msg
	}
	return RenderInfoIcon() + " " + msg
}

// Hint renders a next-step suggestion in a muted color.
func Hint(msg string) string {
	if !Rich() {
		return msg
	}
	return MutedStyle.Render("→ " + msg)
}
