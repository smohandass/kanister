// Copyright 2022 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"time"

	"github.com/go-openapi/strfmt"
	"github.com/pkg/errors"

	"github.com/kanisterio/kanister/pkg/kopia/command/storage"
)

// RepositoryCommandArgs contains fields that are needed for
// creating or connecting to a Kopia repository
type RepositoryCommandArgs struct {
	*CommandArgs
	CacheDirectory  string
	Hostname        string
	ContentCacheMB  int
	MetadataCacheMB int
	Username        string
	RepoPathPrefix  string
	// PITFlag is only effective if set while repository connect
	PITFlag  strfmt.DateTime
	Location map[string][]byte
}

// RepositoryConnectCommand returns the kopia command for connecting to an existing repo
func RepositoryConnectCommand(cmdArgs RepositoryCommandArgs) ([]string, error) {
	args := commonArgs(cmdArgs.CommandArgs)
	args = args.AppendLoggable(repositorySubCommand, connectSubCommand, noCheckForUpdatesFlag)

	args = kopiaCacheArgs(args, cmdArgs.CacheDirectory, cmdArgs.ContentCacheMB, cmdArgs.MetadataCacheMB)

	if cmdArgs.Hostname != "" {
		args = args.AppendLoggableKV(overrideHostnameFlag, cmdArgs.Hostname)
	}

	if cmdArgs.Username != "" {
		args = args.AppendLoggableKV(overrideUsernameFlag, cmdArgs.Username)
	}

	bsArgs, err := storage.KopiaStorageArgs(&storage.StorageCommandParams{
		Location:       cmdArgs.Location,
		RepoPathPrefix: cmdArgs.RepoPathPrefix,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate storage args")
	}

	if !time.Time(cmdArgs.PITFlag).IsZero() {
		bsArgs = bsArgs.AppendLoggableKV(pointInTimeConnectionFlag, cmdArgs.PITFlag.String())
	}

	return stringSliceCommand(args.Combine(bsArgs)), nil
}

// RepositoryCreateCommand returns the kopia command for creation of a repo
func RepositoryCreateCommand(cmdArgs RepositoryCommandArgs) ([]string, error) {
	args := commonArgs(cmdArgs.CommandArgs)
	args = args.AppendLoggable(repositorySubCommand, createSubCommand, noCheckForUpdatesFlag)

	args = kopiaCacheArgs(args, cmdArgs.CacheDirectory, cmdArgs.ContentCacheMB, cmdArgs.MetadataCacheMB)

	if cmdArgs.Hostname != "" {
		args = args.AppendLoggableKV(overrideHostnameFlag, cmdArgs.Hostname)
	}

	if cmdArgs.Username != "" {
		args = args.AppendLoggableKV(overrideUsernameFlag, cmdArgs.Username)
	}

	bsArgs, err := storage.KopiaStorageArgs(&storage.StorageCommandParams{
		Location:       cmdArgs.Location,
		RepoPathPrefix: cmdArgs.RepoPathPrefix,
	})
	if err != nil {
		return nil, errors.Wrap(err, "Failed to generate storage args")
	}

	return stringSliceCommand(args.Combine(bsArgs)), nil
}

// RepositoryServerCommandArgs contains fields required for connecting
// to Kopia Repository API server
type RepositoryServerCommandArgs struct {
	UserPassword    string
	ConfigFilePath  string
	LogDirectory    string
	CacheDirectory  string
	Hostname        string
	ServerURL       string
	Fingerprint     string
	Username        string
	ContentCacheMB  int
	MetadataCacheMB int
}

// RepositoryConnectServerCommand returns the kopia command for connecting to a remote
// repository on Kopia Repository API server
func RepositoryConnectServerCommand(cmdArgs RepositoryServerCommandArgs) []string {
	args := commonArgs(&CommandArgs{
		RepoPassword:   cmdArgs.UserPassword,
		ConfigFilePath: cmdArgs.ConfigFilePath,
		LogDirectory:   cmdArgs.LogDirectory,
	})
	args = args.AppendLoggable(repositorySubCommand, connectSubCommand, serverSubCommand, noCheckForUpdatesFlag, noGrpcFlag)

	args = kopiaCacheArgs(args, cmdArgs.CacheDirectory, cmdArgs.ContentCacheMB, cmdArgs.MetadataCacheMB)

	if cmdArgs.Hostname != "" {
		args = args.AppendLoggableKV(overrideHostnameFlag, cmdArgs.Hostname)
	}

	if cmdArgs.Username != "" {
		args = args.AppendLoggableKV(overrideUsernameFlag, cmdArgs.Username)
	}
	args = args.AppendLoggableKV(urlFlag, cmdArgs.ServerURL)

	args = args.AppendRedactedKV(serverCertFingerprint, cmdArgs.Fingerprint)

	return stringSliceCommand(args)
}

type RepositoryStatusCommandArgs struct {
	*CommandArgs
	GetJsonOutput bool
}

// RepositoryStatusCommand returns the kopia command for checking status of the Kopia repository
func RepositoryStatusCommand(cmdArgs RepositoryStatusCommandArgs) []string {
	// Default to info log level unless specified otherwise.
	if cmdArgs.LogLevel == "" {
		// Make a copy of the common command args, set the log level to info.
		common := *cmdArgs.CommandArgs
		common.LogLevel = LogLevelInfo
		cmdArgs.CommandArgs = &common
	}

	args := commonArgs(cmdArgs.CommandArgs)
	args = args.AppendLoggable(repositorySubCommand, statusSubCommand)
	if cmdArgs.GetJsonOutput {
		args = args.AppendLoggable(jsonFlag)
	}

	return stringSliceCommand(args)
}
