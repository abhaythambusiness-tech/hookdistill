package detect

import (
	"github.com/abhaythambusiness-tech/hookdistill/internal/transcript"
)

type ev struct {
	tool, subject, text string
	side                bool
}

func call(tool, subject string) ev     { return ev{tool: tool, subject: subject} }
func sideCall(tool, subject string) ev { return ev{tool: tool, subject: subject, side: true} }
func say(text string) ev               { return ev{text: text} }

// mkSession builds an in-memory session; Seq is the 1-based event index.
func mkSession(id string, evs ...ev) transcript.Session {
	s := transcript.Session{ID: id, Project: "p"}
	for i, e := range evs {
		seq := i + 1
		if e.tool == "" {
			s.Msgs = append(s.Msgs, transcript.UserMsg{Session: id, Seq: seq, Text: e.text, Project: "p"})
			continue
		}
		in := map[string]any{}
		if e.tool == "Bash" {
			in["command"] = e.subject
		} else {
			in["file_path"] = e.subject
		}
		s.Calls = append(s.Calls, transcript.ToolCall{Session: id, Seq: seq, Tool: e.tool, Input: in, Sidechain: e.side, Project: "p"})
	}
	return s
}
