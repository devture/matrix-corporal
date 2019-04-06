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
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

func main() {
	configPath := flag.String("config", "config.json", "configuration file to use")
	flag.Parse()

	configuration, err := configuration.LoadConfiguration(*configPath)
	if err != nil {
		panic(err)
	}

	container, shutdownHandler := container.BuildContainer(*configuration)

	logger := container.Get("logger").(*logrus.Logger)

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
