package cli

import (
	"io"
	"strings"

	"github.com/spf13/cobra"
)

// promptResponse は `prx prompt` の JSON 形式。kind を持たせるのは、呼び出し側が
// task から導出し直さずに設計依頼と実装依頼を区別できるようにするため。
type promptResponse struct {
	TaskID string `json:"task_id"`
	Kind   string `json:"kind"`
	Prompt string `json:"prompt"`
}

func (s *state) promptCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "prompt TASK_ID",
		Short: "Print the agent prompt for a task",
		Long: "Print the agent prompt for a task.\n\n" +
			"A task without an implementation plan gets the design prompt, and a task with one gets " +
			"the implementation prompt.\n" +
			"The result resolves global, project, and feature overrides, so the WebUI copies the same text.",
		Example: "prx prompt T-1\nprx prompt T-1 --json",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, body, err := s.service.GetTaskPrompt(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			value := promptResponse{TaskID: args[0], Kind: string(kind), Prompt: body}
			return s.write(value, renderPrompt(body))
		},
	}
}

// renderPrompt はプロンプト本文だけを出力する。それ以外があると、別のエージェントに
// 渡す前に手で消す必要が出てしまう。
func renderPrompt(body string) humanRenderer {
	return func(out io.Writer) error {
		if _, err := io.WriteString(out, body); err != nil {
			return err
		}
		if strings.HasSuffix(body, "\n") {
			return nil
		}
		_, err := io.WriteString(out, "\n")
		return err
	}
}
