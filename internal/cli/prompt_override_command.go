package cli

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/NextNowX/internal/domain"
	"github.com/HappyOnigiri/NextNowX/internal/prompt"
)

func promptOverrideCommands(
	write func(any, humanRenderer) error,
	set func(context.Context, string, prompt.Kind, string) (any, error),
	unset func(context.Context, string, prompt.Kind) (any, error),
	resource string,
) *cobra.Command {
	command := &cobra.Command{
		Use:   "prompt",
		Short: "Manage prompt template overrides",
		Long: "Manage prompt template overrides for a project or feature.\n\n" +
			"KIND is design, implementation, batch, or batch_design. An unset kind inherits from its parent.",
	}
	command.AddCommand(
		promptOverrideSetCommand(write, set, resource),
		promptOverrideUnsetCommand(write, unset, resource),
	)
	return command
}

func (s *state) projectPromptCommand() *cobra.Command {
	return promptOverrideCommands(
		s.write,
		func(ctx context.Context, id string, kind prompt.Kind, body string) (any, error) {
			update := promptOverrideUpdate(kind, &body)
			return s.service.UpdateProject(ctx, id, domain.ProjectUpdate{PromptOverrides: &update})
		},
		func(ctx context.Context, id string, kind prompt.Kind) (any, error) {
			empty := ""
			update := promptOverrideUpdate(kind, &empty)
			return s.service.UpdateProject(ctx, id, domain.ProjectUpdate{PromptOverrides: &update})
		},
		"project",
	)
}

func (s *state) featurePromptCommand() *cobra.Command {
	return promptOverrideCommands(
		s.write,
		func(ctx context.Context, id string, kind prompt.Kind, body string) (any, error) {
			update := promptOverrideUpdate(kind, &body)
			return s.service.UpdateFeature(ctx, id, domain.FeatureUpdate{PromptOverrides: &update})
		},
		func(ctx context.Context, id string, kind prompt.Kind) (any, error) {
			empty := ""
			update := promptOverrideUpdate(kind, &empty)
			return s.service.UpdateFeature(ctx, id, domain.FeatureUpdate{PromptOverrides: &update})
		},
		"feature",
	)
}

func promptOverrideSetCommand(
	write func(any, humanRenderer) error,
	set func(context.Context, string, prompt.Kind, string) (any, error),
	resource string,
) *cobra.Command {
	var file string
	var stdin bool
	command := &cobra.Command{
		Use:     "set " + strings.ToUpper(resource) + "_ID KIND",
		Short:   "Set a prompt template override",
		Example: "nnx " + resource + " prompt set " + resource[:1] + "-1 design --file prompt.txt",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, err := parsePromptOverrideKind(args[1])
			if err != nil {
				return err
			}
			body, err := readPromptOverride(cmd, file, stdin)
			if err != nil {
				return err
			}
			value, err := set(cmd.Context(), args[0], kind, body)
			if err != nil {
				return err
			}
			return write(value, renderMessage("Set %s prompt override %s (%s).", resource, args[0], kind))
		},
	}
	command.Flags().StringVar(&file, "file", "", "read the template from a file")
	command.Flags().BoolVar(&stdin, "stdin", false, "read the template from standard input")
	return command
}

func promptOverrideUnsetCommand(
	write func(any, humanRenderer) error,
	unset func(context.Context, string, prompt.Kind) (any, error),
	resource string,
) *cobra.Command {
	command := &cobra.Command{
		Use:     "unset " + strings.ToUpper(resource) + "_ID KIND",
		Short:   "Remove a prompt template override",
		Example: "nnx " + resource + " prompt unset " + resource[:1] + "-1 design",
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, err := parsePromptOverrideKind(args[1])
			if err != nil {
				return err
			}
			value, err := unset(cmd.Context(), args[0], kind)
			if err != nil {
				return err
			}
			return write(value, renderMessage("Unset %s prompt override %s (%s).", resource, args[0], kind))
		},
	}
	return command
}

func parsePromptOverrideKind(value string) (prompt.Kind, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch prompt.Kind(normalized) {
	case prompt.KindDesign, prompt.KindImplementation, prompt.KindBatch, prompt.KindBatchDesign:
		return prompt.Kind(normalized), nil
	default:
		return "", domain.NewError(
			domain.DomainErrorCodeInvalidPromptTemplate,
			"prompt template kind must be design, implementation, batch, or batch_design",
		)
	}
}

func readPromptOverride(command *cobra.Command, file string, stdin bool) (string, error) {
	if (file == "") == !stdin {
		return "", domain.NewError(
			domain.DomainErrorCodeInvalidPromptTemplate,
			"specify exactly one of --file or --stdin",
		)
	}
	var (
		body []byte
		err  error
	)
	if stdin {
		body, err = io.ReadAll(command.InOrStdin())
	} else {
		body, err = os.ReadFile(file)
	}
	if err != nil {
		return "", domain.NewError(
			domain.DomainErrorCodeInvalidPromptTemplate,
			"could not read prompt template: %v",
			err,
		)
	}
	if strings.TrimSpace(string(body)) == "" {
		return "", domain.NewError(domain.DomainErrorCodeInvalidPromptTemplate, "prompt template is empty")
	}
	return string(body), nil
}

func promptOverrideUpdate(kind prompt.Kind, body *string) domain.PromptTemplateOverridesUpdate {
	update := domain.PromptTemplateOverridesUpdate{}
	switch kind {
	case prompt.KindDesign:
		update.Design = body
	case prompt.KindImplementation:
		update.Implementation = body
	case prompt.KindBatch:
		update.Batch = body
	case prompt.KindBatchDesign:
		update.BatchDesign = body
	}
	return update
}
