package cli

import (
	"io"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/NextNowX/internal/domain"
)

func (s *state) taskGroupCommand() *cobra.Command {
	command := &cobra.Command{Use: "group", Short: "Manage task groups in a feature"}
	command.AddCommand(
		&cobra.Command{
			Use: "create FEATURE_ID NAME", Short: "Create a task group", Args: cobra.ExactArgs(2),
			Example: "nnx task group create F-1 Checkout",
			RunE: func(cmd *cobra.Command, args []string) error {
				group, err := s.service.CreateTaskGroup(cmd.Context(), args[0], args[1])
				if err != nil {
					return err
				}
				return s.write(group, renderMessage("Created task group %s.", group.ID))
			},
		},
		&cobra.Command{
			Use: "list FEATURE_ID", Short: "List task groups in a feature", Args: cobra.ExactArgs(1),
			Example: "nnx task group list F-1",
			RunE:    s.listTaskGroups,
		},
		&cobra.Command{
			Use: "rename GROUP_ID NAME", Short: "Rename a task group", Args: cobra.ExactArgs(2),
			Example: "nnx task group rename G-1 Payments",
			RunE: func(cmd *cobra.Command, args []string) error {
				group, err := s.service.RenameTaskGroup(cmd.Context(), args[0], args[1])
				if err != nil {
					return err
				}
				return s.write(group, renderMessage("Renamed task group %s.", group.ID))
			},
		},
		&cobra.Command{
			Use: "delete GROUP_ID", Short: "Delete a group and ungroup its tasks", Args: cobra.ExactArgs(1),
			Example: "nnx task group delete G-1",
			RunE: func(cmd *cobra.Command, args []string) error {
				if err := s.service.DeleteTaskGroup(cmd.Context(), args[0]); err != nil {
					return err
				}
				return s.write(map[string]string{"deleted": args[0]}, renderMessage("Deleted task group %s.", args[0]))
			},
		},
	)
	return command
}

func (s *state) listTaskGroups(cmd *cobra.Command, args []string) error {
	snapshot, err := s.service.Snapshot(cmd.Context())
	if err != nil {
		return err
	}
	for _, feature := range snapshot.Features {
		if feature.ID == args[0] {
			groups := featureTaskGroups(snapshot.TaskGroups, feature.ID)
			return s.write(map[string]any{"task_groups": groups}, renderTaskGroups(groups))
		}
	}
	return domain.NewError(domain.DomainErrorCodeNotFound, "feature %q was not found", args[0])
}

func featureTaskGroups(groups []domain.TaskGroup, featureID string) []domain.TaskGroup {
	result := make([]domain.TaskGroup, 0)
	for _, group := range groups {
		if group.FeatureID == featureID {
			result = append(result, group)
		}
	}
	return result
}

func renderTaskGroups(groups []domain.TaskGroup) humanRenderer {
	if len(groups) == 0 {
		return renderMessage("No task groups found.")
	}
	return func(out io.Writer) error {
		return writeTable(out, []string{"ID", "NAME"}, func(table *tabwriter.Writer) {
			for _, group := range groups {
				writeRow(table, group.ID, group.Name)
			}
		})
	}
}
