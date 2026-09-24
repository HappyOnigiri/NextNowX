package rpc

import (
	"time"

	nnxv1 "github.com/HappyOnigiri/NextNowX/gen/nnx/v1"
	"github.com/HappyOnigiri/NextNowX/internal/domain"
)

func protoDebugReport(v domain.DebugReport) *nnxv1.DebugReport {
	result := &nnxv1.DebugReport{
		Build:      protoDebugBuild(v.Build),
		Runtime:    protoDebugRuntime(v.Runtime),
		Daemon:     protoDebugDaemon(v.Daemon),
		Paths:      protoDebugPaths(v.Paths),
		Config:     protoDebugConfig(v.Config),
		Storage:    protoDebugStorage(v.Storage),
		Records:    protoDebugData(v.Records),
		GithubSync: protoDebugGitHubSync(v.GitHubSync),
	}
	for _, problem := range v.Problems {
		result.Problems = append(result.Problems, &nnxv1.DebugProblem{
			Code:        protoDebugProblemCode(problem.Code),
			Target:      problem.Target,
			Evidence:    problem.Evidence,
			NextCommand: problem.NextCommand,
		})
	}
	return result
}

func protoDebugBuild(v domain.DebugBuild) *nnxv1.DebugBuild {
	return &nnxv1.DebugBuild{
		Version:     v.Version,
		Development: v.Development,
		GoVersion:   v.GoVersion,
		Os:          v.OS,
		Arch:        v.Arch,
	}
}

func protoDebugRuntime(v domain.DebugRuntime) *nnxv1.DebugRuntime {
	return &nnxv1.DebugRuntime{
		Mode:          v.Mode,
		Demo:          v.Demo,
		GithubFixture: v.GitHubFixture,
		GeneratedAt:   v.GeneratedAt.Format(timeFormat),
		TimeZone:      v.TimeZone,
		ListenAddress: v.ListenAddress,
		StartedAt:     protoDebugTime(v.StartedAt),
		UptimeSeconds: v.UptimeSeconds,
	}
}

func protoDebugDaemon(v domain.DebugDaemon) *nnxv1.DebugDaemon {
	return &nnxv1.DebugDaemon{
		Supported:     v.Supported,
		Installed:     v.Installed,
		PlistStatus:   v.PlistStatus,
		PlistPath:     v.PlistPath,
		LogPath:       v.LogPath,
		Running:       v.Running,
		Address:       v.Address,
		Pid:           int32(v.PID),
		Version:       v.Version,
		BinaryMatches: v.BinaryMatches,
		Error:         v.Error,
	}
}

func protoDebugPaths(v domain.DebugPaths) *nnxv1.DebugPaths {
	result := &nnxv1.DebugPaths{
		DatabasePath:       v.DatabasePath,
		DatabasePathSource: v.DatabasePathSource,
		DatabaseFileExists: v.DatabaseFileExists,
		ConfigPath:         v.ConfigPath,
		ConfigPathSource:   v.ConfigPathSource,
		ConfigFileExists:   v.ConfigFileExists,
		ConfigPermissions:  v.ConfigPermissions,
	}
	for _, variable := range v.EnvironmentVariables {
		result.EnvironmentVariables = append(result.EnvironmentVariables, &nnxv1.DebugEnvironmentVariable{
			Name: variable.Name,
			Set:  variable.Set,
		})
	}
	return result
}

func protoDebugConfig(v domain.DebugConfig) *nnxv1.DebugConfig {
	result := &nnxv1.DebugConfig{
		Version:                 int32(v.Version),
		Valid:                   v.Valid,
		Errors:                  v.Errors,
		Warnings:                v.Warnings,
		AutoSyncIntervalSeconds: v.AutoSyncIntervalSeconds,
	}
	for _, host := range v.Hosts {
		result.Hosts = append(result.Hosts, &nnxv1.DebugConfigHost{
			Host:       host.Host,
			ApiUrl:     host.APIURL,
			GraphqlUrl: host.GraphQLURL,
		})
	}
	for _, method := range v.AuthMethods {
		result.AuthMethods = append(result.AuthMethods, &nnxv1.DebugConfigAuthMethod{
			Id:               method.ID,
			Host:             method.Host,
			Type:             method.Type,
			SecretConfigured: method.SecretConfigured,
		})
	}
	return result
}

func protoDebugStorage(v domain.DebugStorage) *nnxv1.DebugStorage {
	return &nnxv1.DebugStorage{
		AppliedSchemaVersion:  int32(v.AppliedSchemaVersion),
		EmbeddedSchemaVersion: int32(v.EmbeddedSchemaVersion),
		IntegrityValid:        v.IntegrityValid,
		IntegrityErrors:       v.IntegrityErrors,
		DatabaseFile: &nnxv1.DebugDatabaseFile{
			Applicable:   v.DatabaseFile.Applicable,
			SizeBytes:    v.DatabaseFile.SizeBytes,
			WalPresent:   v.DatabaseFile.WALPresent,
			WalSizeBytes: v.DatabaseFile.WALSizeBytes,
			ShmPresent:   v.DatabaseFile.SHMPresent,
			Writable:     v.DatabaseFile.Writable,
			WriteError:   v.DatabaseFile.WriteError,
		},
		CliSchemaVersion: v.CLISchemaVersion,
		Error:            v.Error,
	}
}

func protoDebugData(v domain.DebugData) *nnxv1.DebugData {
	return &nnxv1.DebugData{
		Projects:                 int32(v.Projects),
		ProjectStates:            protoDebugCounts(v.ProjectStates),
		Features:                 int32(v.Features),
		Tasks:                    int32(v.Tasks),
		Dependencies:             int32(v.Dependencies),
		PullRequests:             int32(v.PullRequests),
		Documents:                int32(v.Documents),
		FeatureStatuses:          protoDebugCounts(v.FeatureStatuses),
		TaskDisplayStates:        protoDebugCounts(v.TaskDisplayStates),
		PullRequestDisplayStates: protoDebugCounts(v.PullRequestDisplayStates),
		PullRequestHosts:         protoDebugCounts(v.PullRequestHosts),
		DocumentKinds:            protoDebugCounts(v.DocumentKinds),
		Error:                    v.Error,
	}
}

func protoDebugGitHubSync(v domain.DebugGitHubSync) *nnxv1.DebugGitHubSync {
	result := &nnxv1.DebugGitHubSync{
		Status:                    protoGitHubSyncStatus(v.Status),
		NextRunAt:                 protoDebugTime(v.NextRunAt),
		Due:                       v.Due,
		SecondsSinceLastUpdate:    v.SecondsSinceLastUpdate,
		StalePullRequests:         int32(v.StalePullRequests),
		FailedPullRequests:        int32(v.FailedPullRequests),
		HostFailures:              protoDebugFailures(v.HostFailures),
		RepositoryFailures:        protoDebugFailures(v.RepositoryFailures),
		OmittedRepositoryFailures: int32(v.OmittedRepositoryFailures),
		OmittedErrorGroups:        int32(v.OmittedErrorGroups),
		OmittedAuthCacheEntries:   int32(v.OmittedAuthCacheEntries),
		Error:                     v.Error,
	}
	for _, group := range v.ErrorGroups {
		result.ErrorGroups = append(result.ErrorGroups, &nnxv1.DebugErrorGroup{
			Message:        group.Message,
			Count:          int32(group.Count),
			TaskIds:        group.TaskIDs,
			TotalTaskCount: int32(group.TotalTaskCount),
		})
	}
	for _, entry := range v.AuthCache {
		result.AuthCache = append(result.AuthCache, &nnxv1.DebugAuthCacheEntry{
			Host:            entry.Host,
			Owner:           entry.Owner,
			Repository:      entry.Repository,
			AuthMethodId:    entry.AuthMethodID,
			LastSucceededAt: entry.LastSucceededAt.Format(timeFormat),
		})
	}
	return result
}

func protoDebugCounts(values []domain.DebugCount) []*nnxv1.DebugCount {
	result := make([]*nnxv1.DebugCount, 0, len(values))
	for _, value := range values {
		result = append(result, &nnxv1.DebugCount{Name: value.Name, Count: int32(value.Count)})
	}
	return result
}

func protoDebugFailures(values []domain.DebugSyncFailure) []*nnxv1.DebugSyncFailure {
	result := make([]*nnxv1.DebugSyncFailure, 0, len(values))
	for _, value := range values {
		result = append(result, &nnxv1.DebugSyncFailure{Scope: value.Scope, Count: int32(value.Count)})
	}
	return result
}

func protoDebugTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(timeFormat)
}

func protoDebugProblemCode(value domain.DebugProblemCode) nnxv1.DebugProblemCode {
	switch value {
	case domain.DebugProblemCodeStorageUnavailable:
		return nnxv1.DebugProblemCode_DEBUG_PROBLEM_CODE_STORAGE_UNAVAILABLE
	case domain.DebugProblemCodeSchemaVersionAheadOfBinary:
		return nnxv1.DebugProblemCode_DEBUG_PROBLEM_CODE_SCHEMA_VERSION_AHEAD_OF_BINARY
	case domain.DebugProblemCodeDatabaseNotWritable:
		return nnxv1.DebugProblemCode_DEBUG_PROBLEM_CODE_DATABASE_NOT_WRITABLE
	case domain.DebugProblemCodeDatabaseIntegrityErrors:
		return nnxv1.DebugProblemCode_DEBUG_PROBLEM_CODE_DATABASE_INTEGRITY_ERRORS
	case domain.DebugProblemCodeConfigUnreadable:
		return nnxv1.DebugProblemCode_DEBUG_PROBLEM_CODE_CONFIG_UNREADABLE
	case domain.DebugProblemCodeConfigPermissionsTooOpen:
		return nnxv1.DebugProblemCode_DEBUG_PROBLEM_CODE_CONFIG_PERMISSIONS_TOO_OPEN
	case domain.DebugProblemCodeConfigUnknownFields:
		return nnxv1.DebugProblemCode_DEBUG_PROBLEM_CODE_CONFIG_UNKNOWN_FIELDS
	case domain.DebugProblemCodeNoAuthMethodForHost:
		return nnxv1.DebugProblemCode_DEBUG_PROBLEM_CODE_NO_AUTH_METHOD_FOR_HOST
	case domain.DebugProblemCodeGitHubSyncRunError:
		return nnxv1.DebugProblemCode_DEBUG_PROBLEM_CODE_GITHUB_SYNC_RUN_ERROR
	case domain.DebugProblemCodeGitHubSyncOverdue:
		return nnxv1.DebugProblemCode_DEBUG_PROBLEM_CODE_GITHUB_SYNC_OVERDUE
	case domain.DebugProblemCodeGitHubSyncNeverCompleted:
		return nnxv1.DebugProblemCode_DEBUG_PROBLEM_CODE_GITHUB_SYNC_NEVER_COMPLETED
	case domain.DebugProblemCodePullRequestsStale:
		return nnxv1.DebugProblemCode_DEBUG_PROBLEM_CODE_PULL_REQUESTS_STALE
	case domain.DebugProblemCodeDaemonPlistStale:
		return nnxv1.DebugProblemCode_DEBUG_PROBLEM_CODE_DAEMON_PLIST_STALE
	case domain.DebugProblemCodeDaemonNotRunning:
		return nnxv1.DebugProblemCode_DEBUG_PROBLEM_CODE_DAEMON_NOT_RUNNING
	case domain.DebugProblemCodeDaemonBinaryOutdated:
		return nnxv1.DebugProblemCode_DEBUG_PROBLEM_CODE_DAEMON_BINARY_OUTDATED
	default:
		return nnxv1.DebugProblemCode_DEBUG_PROBLEM_CODE_UNSPECIFIED
	}
}
