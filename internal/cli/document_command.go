package cli

import (
	"context"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/HappyOnigiri/NextNowX/internal/domain"
)

type documentSourceFlags struct {
	url, localFile, markdownFile string
	stdin                        bool
}

func (s *state) documentCommand() *cobra.Command {
	var filter domain.DocumentParent
	command := &cobra.Command{
		Use:     "document",
		Aliases: []string{"doc"},
		Short:   "List or manage documents",
		Long:    "List or manage URL, local file, and stored Markdown documents.\n\nAlias: doc.",
		Example: "nnx document\nnnx document --task T-1\nnnx document --group G-1\n" +
			"nnx document --project P-1\nnnx doc",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			snapshot, err := s.service.Snapshot(cmd.Context())
			if err != nil {
				return err
			}
			parent, err := s.resolveDocumentFilter(cmd.Context(), snapshot, filter)
			if err != nil {
				return err
			}
			documents := make([]domain.Document, 0, len(snapshot.Documents))
			for _, document := range snapshot.Documents {
				if parent.ProjectID != "" && document.ProjectID != parent.ProjectID {
					continue
				}
				if parent.FeatureID != "" && document.FeatureID != parent.FeatureID {
					continue
				}
				if parent.TaskID != "" && document.TaskID != parent.TaskID {
					continue
				}
				if parent.GroupID != "" && document.GroupID != parent.GroupID {
					continue
				}
				documents = append(documents, document)
			}
			return s.write(map[string]any{"documents": documents}, renderDocumentList(documents))
		},
	}
	command.Flags().StringVar(&filter.ProjectID, "project", "", "filter by project ID")
	command.Flags().StringVar(&filter.FeatureID, "feature", "", "filter by feature ID")
	command.Flags().StringVar(&filter.TaskID, "task", "", "filter by task ID")
	command.Flags().StringVar(&filter.GroupID, "group", "", "filter by task group ID")

	command.AddCommand(
		s.documentAddCommand(),
		s.documentGetCommand(),
		s.documentUpdateCommand(),
		s.documentDeleteCommand(),
	)
	return command
}

// resolveDocumentFilter は一覧の絞り込み条件を public identifier に変換し、
// スナップショットが持つ値と同じもので比較できるようにする。
func (s *state) resolveDocumentFilter(
	ctx context.Context,
	snapshot domain.Snapshot,
	filter domain.DocumentParent,
) (domain.DocumentParent, error) {
	if filter.ProjectID != "" {
		project, err := s.service.ResolveProject(ctx, filter.ProjectID)
		if err != nil {
			return domain.DocumentParent{}, err
		}
		filter.ProjectID = project.ID
	}
	if filter.FeatureID != "" {
		feature, err := s.service.ResolveFeature(ctx, filter.FeatureID)
		if err != nil {
			return domain.DocumentParent{}, err
		}
		filter.FeatureID = feature.ID
	}
	if filter.TaskID != "" {
		if err := requireTask(snapshot, filter.TaskID); err != nil {
			return domain.DocumentParent{}, err
		}
	}
	if filter.GroupID != "" && !slices.ContainsFunc(snapshot.TaskGroups, func(group domain.TaskGroup) bool {
		return group.ID == filter.GroupID
	}) {
		return domain.DocumentParent{}, domain.NewError(
			domain.DomainErrorCodeNotFound, "task group %q was not found", filter.GroupID,
		)
	}
	return filter, nil
}

func (s *state) documentAddCommand() *cobra.Command {
	var title string
	var implementationPlan bool
	var source documentSourceFlags
	command := &cobra.Command{
		Use:   "add PARENT_ID",
		Short: "Add a document to a project, feature, task, or task group",
		Long: "Add a document to a project, feature, task, or task group.\n\n" +
			"The operand is a public project, feature, task, or task group ID.",
		Example: "nnx document add T-1 --url https://example.com\n" +
			"nnx document add P-1 --url https://example.com\n" +
			"nnx document add F-1 --markdown-file notes.md\n" +
			"nnx document add G-1 --url https://example.com/runbook",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			document, err := readDocumentSource(cmd, source, domain.DomainErrorCodeInvalidDocument)
			if err != nil {
				return err
			}
			parent, err := s.documentParentForID(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			doc, err := s.service.AddDocument(
				cmd.Context(),
				parent,
				document.Kind,
				title,
				document.Locator,
				document.Content,
				implementationPlan,
			)
			if err != nil {
				return err
			}
			return s.write(doc, renderMessage("Added document %s.", doc.ID))
		},
	}
	command.Flags().StringVar(&title, "title", "", "document title")
	command.Flags().BoolVar(&implementationPlan, "implementation-plan", false, "mark as the task implementation plan")
	bindDocumentSourceFlags(command, &source)
	return command
}

func (s *state) documentParentForID(ctx context.Context, id string) (domain.DocumentParent, error) {
	if strings.HasPrefix(id, "G-") {
		return domain.DocumentParent{GroupID: id}, nil
	}
	node, err := s.service.GetNode(ctx, id)
	if err != nil {
		return domain.DocumentParent{}, err
	}
	switch typed := node.(type) {
	case domain.Project:
		return domain.DocumentParent{ProjectID: typed.ID}, nil
	case domain.Feature:
		return domain.DocumentParent{FeatureID: typed.ID}, nil
	case domain.Task:
		return domain.DocumentParent{TaskID: typed.ID}, nil
	default:
		return domain.DocumentParent{}, domain.NewError(
			domain.DomainErrorCodeInvalidParent, "%q resolved to an unsupported document parent", id,
		)
	}
}

func (s *state) documentGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "get DOCUMENT_ID",
		Short:   "Get one document",
		Example: "nnx document get DOCUMENT_ID",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			document, err := s.service.GetDocument(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return s.write(document, renderDocumentDetail(document))
		},
	}
}

func (s *state) documentUpdateCommand() *cobra.Command {
	var updateTitle string
	var updatePlan bool
	var updateSource documentSourceFlags
	command := &cobra.Command{
		Use:     "update DOCUMENT_ID",
		Short:   "Update a document",
		Example: "nnx document update DOCUMENT_ID --title Runbook",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var titlePointer *string
			if cmd.Flags().Changed("title") {
				titlePointer = &updateTitle
			}
			var sourcePointer *domain.Document
			if documentSourceCount(updateSource) > 0 {
				value, err := readDocumentSource(cmd, updateSource, domain.DomainErrorCodeInvalidDocument)
				if err != nil {
					return err
				}
				sourcePointer = &value
			}
			var planPointer *bool
			if cmd.Flags().Changed("implementation-plan") {
				planPointer = &updatePlan
			}
			document, err := s.service.UpdateDocument(
				cmd.Context(), args[0], titlePointer, sourcePointer, planPointer,
			)
			if err != nil {
				return err
			}
			return s.write(document, renderDocumentDetail(document))
		},
	}
	command.Flags().StringVar(&updateTitle, "title", "", "replace the document title")
	command.Flags().BoolVar(&updatePlan, "implementation-plan", false, "set or clear the plan designation")
	bindDocumentSourceFlags(command, &updateSource)
	return command
}

func (s *state) documentDeleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "delete DOCUMENT_ID",
		Short:   "Delete a document; missing documents return not_found",
		Example: "nnx document delete DOCUMENT_ID",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := s.service.DeleteDocument(cmd.Context(), args[0]); err != nil {
				return err
			}
			return s.write(map[string]string{"deleted": args[0]}, renderMessage("Deleted document %s.", args[0]))
		},
	}
}

func requireTask(snapshot domain.Snapshot, taskID string) error {
	for _, task := range snapshot.Tasks {
		if task.ID == taskID {
			return nil
		}
	}
	return domain.NewError(domain.DomainErrorCodeNotFound, "task %q was not found", taskID)
}

func bindDocumentSourceFlags(command *cobra.Command, flags *documentSourceFlags) {
	command.Flags().StringVar(&flags.url, "url", "", "HTTP or HTTPS URL")
	command.Flags().StringVar(&flags.localFile, "local-file", "", "registered local file path")
	command.Flags().StringVar(&flags.markdownFile, "markdown-file", "", "read stored Markdown from a file")
	command.Flags().BoolVar(&flags.stdin, "stdin", false, "read stored Markdown from standard input")
}

func documentSourceCount(flags documentSourceFlags) int {
	count := 0
	for _, value := range []bool{
		flags.url != "", flags.localFile != "", flags.markdownFile != "", flags.stdin,
	} {
		if value {
			count++
		}
	}
	return count
}

func readDocumentSource(
	command *cobra.Command,
	flags documentSourceFlags,
	errorCode domain.DomainErrorCode,
) (domain.Document, error) {
	if documentSourceCount(flags) != 1 {
		return domain.Document{}, domain.NewError(errorCode, "specify exactly one document source")
	}
	if flags.url != "" {
		return domain.Document{Kind: domain.DocumentKindURL, Locator: flags.url}, nil
	}
	if flags.localFile != "" {
		return domain.Document{Kind: domain.DocumentKindLocalFile, Locator: flags.localFile}, nil
	}
	var content []byte
	var err error
	if flags.stdin {
		content, err = io.ReadAll(command.InOrStdin())
	} else {
		content, err = os.ReadFile(flags.markdownFile)
	}
	if err != nil {
		return domain.Document{}, domain.NewError(errorCode, "could not read Markdown: %v", err)
	}
	return domain.Document{Kind: domain.DocumentKindMarkdown, Content: string(content)}, nil
}
