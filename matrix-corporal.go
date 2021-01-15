// matrix-corporal is a reconciliation and gateway program for Matrix servers
// Copyright (C) 2018 Slavi Pantaleev
//
// http://devture.com/
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"devture-matrix-corporal/corporal/configuration"
	"devture-matrix-corporal/corporal/container"
	"devture-matrix-corporal/corporal/httpapi"
	"devture-matrix-corporal/corporal/httpgateway"
	"devture-matrix-corporal/corporal/policy/provider"
	"devture-matrix-corporal/corporal/reconciliation/reconciler"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

// Following variables will be statically linked at the time of compiling
// Source: https://oddcode.daveamit.com/2018/08/17/embed-versioning-information-in-golang-binary/

// GitCommit holds short commit hash of source tree
var GitCommit string

// GitBranch holds current branch name the code is built off
var GitBranch string

// GitState shows whether there are uncommitted changes
var GitState string

// GitSummary holds output of git describe --tags --dirty --always
var GitSummary string

// BuildDate holds RFC3339 formatted UTC date (build time)
var BuildDate string

// Version holds contents of ./VERSION file, if exists, or the value passed via the -version option
var Version string

func main() {
	fmt.Printf(`
                 _        _                                                _
 _ __ ___   __ _| |_ _ __(_)_  __      ___ ___  _ __ _ __   ___  _ __ __ _| |
| '_ \ _ \ / _\ | __| '__| \ \/ /____ / __/ _ \| '__| '_ \ / _ \| '__/ _\ | |
| | | | | | (_| | |_| |  | |>  <_____| (_| (_) | |  | |_) | (_) | | | (_| | |
|_| |_| |_|\__,_|\__|_|  |_/_/\_\     \___\___/|_|  | .__/ \___/|_|  \__,_|_|
                                                    |_|
---------------------------------------------------------- [ Version: %s ]
GitCommit: %s
GitBranch: %s
GitState: %s
GitSummary: %s
BuildDate: %s

`, Version, GitCommit, GitBranch, GitState, GitSummary, BuildDate)

	// Starting with a debug logger, but we may tone it down it below.
	logger := logrus.New()
	logger.Level = logrus.DebugLevel

	configPath := flag.String("config", "config.json", "configuration file to use")
	flag.Parse()

	configuration, err := configuration.LoadConfiguration(*configPath, logger)
	if err != nil {
		panic(err)
	}

	if !configuration.Misc.Debug {
		logger.Level = logrus.InfoLevel
	}

	container, shutdownHandler := container.BuildContainer(*configuration, logger)

	httpGatewayServer := container.Get("httpgateway.server").(*httpgateway.Server)
	err = httpGatewayServer.Start()
	if err != nil {
		panic(err)
	}

	if configuration.HttpApi.Enabled {
		httpApiServer := container.Get("httpapi.server").(*httpapi.Server)
		err = httpApiServer.Start()
		if err != nil {
			panic(err)
		}
	} else {
		logger.Infof("Not starting HTTP API server: disabled by configuration")
	}

	// This needs to start before the policy provider,
	// as it would listen for notifications from the policy store and we don't want it to miss any.
	storeDrivenReconciler := container.Get("reconciliation.store_driven_reconciler").(*reconciler.StoreDrivenReconciler)
	err = storeDrivenReconciler.Start()
	if err != nil {
		panic(err)
	}

	policyProvider := container.Get("policy.provider").(provider.Provider)
	err = policyProvider.Start()
	if err != nil {
		panic(err)
	}

	channelComplete := make(chan bool)
	setupSignalHandling(
		channelComplete,
		shutdownHandler,
	)

	<-channelComplete
}

func setupSignalHandling(
	channelComplete chan bool,
	shutdownHandler *container.ContainerShutdownHandler,
) {
	signalChannel := make(chan os.Signal, 2)
	signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChannel

		shutdownHandler.Shutdown()

		channelComplete <- true
	}()
}
