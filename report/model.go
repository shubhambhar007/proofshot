package report

import "time"

type CommandResult struct {
	Command    string        `json:"command"`
	Output     string        `json:"output"`
	ExitCode   int           `json:"exitCode"`
	Duration   time.Duration `json:"durationNs"`
	StartedAt  time.Time     `json:"startedAt"`
	TimedOut   bool          `json:"timedOut"`
	Redactions int           `json:"redactions"`
}

type Report struct {
	Title     string          `json:"title"`
	CreatedAt time.Time       `json:"createdAt"`
	Duration  time.Duration   `json:"durationNs"`
	Failed    int             `json:"failed"`
	Commands  []CommandResult `json:"commands"`
}

func New(title string, commands []CommandResult) Report {
	r := Report{Title: title, CreatedAt: time.Now(), Commands: commands}
	for _, command := range commands {
		r.Duration += command.Duration
		if command.ExitCode != 0 {
			r.Failed++
		}
	}
	return r
}
