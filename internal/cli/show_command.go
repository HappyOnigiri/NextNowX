package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/NextNowX/internal/domain"
)

func (s *state) showCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "show ID",
		Short:   "Show a project, feature, task, or task group by public ID",
		Example: "nnx show F-1\nnnx show T-1\nnnx show P-1\nnnx show G-1",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 && strings.HasPrefix(args[0], "T-") {
				snapshot, err := s.service.Snapshot(cmd.Context())
				if err != nil {
					return err
				}
				for _, task := range snapshot.Tasks {
					if task.ID == args[0] {
						return s.write(task, renderTaskDetailWithAppearances(task, featureAppearances(snapshot)))
					}
				}
				return domain.NewError(domain.DomainErrorCodeNotFound, "task %q was not found", args[0])
			}
			value, err := s.service.GetNode(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return s.write(value, renderNode(value))
		},
	}
}
