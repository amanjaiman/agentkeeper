// Package prompt builds the instruction injected into the agent when its limit
// resets. M1 ships the static builder; the local-LLM builder (M4) will satisfy
// the same interface so the supervisor stays agnostic.
package prompt

// Context carries the information a builder may use to compose a resume prompt.
// M1's static builder ignores it; it exists so M4 can add transcript/diff data
// without changing the supervisor call site.
type Context struct {
	Agent string
	Cwd   string
}

// Builder produces the next instruction to send to the agent.
type Builder interface {
	Build(Context) (string, error)
}

// DefaultText is used when no custom resume prompt is supplied.
const DefaultText = "Usage limit reset. Continue with the prior task."

// Static always returns a fixed string.
type Static struct {
	Text string
}

// NewStatic returns a Static builder, defaulting to DefaultText when text is empty.
func NewStatic(text string) Static {
	if text == "" {
		text = DefaultText
	}
	return Static{Text: text}
}

func (s Static) Build(Context) (string, error) { return s.Text, nil }
