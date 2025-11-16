// Copyright 2025 IBM Corp
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

package main

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/hypervisors"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/pagination"

	"k8s.io/apimachinery/pkg/util/wait"
)

func getServiceClient(ctx context.Context, serviceType string, cloud string) (client *gophercloud.ServiceClient, err error) {
	backoff := wait.Backoff{
		Duration: 1 * time.Minute,
		Factor:   1.1,
		Cap:      leftInContext(ctx),
		Steps:    math.MaxInt32,
	}

	err = wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
		var (
			err2 error
		)

		log.Debugf("getServiceClient: duration = %v, calling NewServiceClient(%s, %s)", leftInContext(ctx), serviceType, cloud)
		client, err2 = NewServiceClient(ctx, serviceType, DefaultClientOpts(cloud))
		if err2 != nil {
			log.Debugf("getServiceClient: Error: NewServiceClient returns error %v\n", err2)
			return false, nil
		}

		return true, nil
	})

	return
}

func findFlavor(ctx context.Context, cloudName string, name string) (foundFlavor flavors.Flavor, err error) {
	var (
		pager      pagination.Page
		allFlavors []flavors.Flavor
		flavor     flavors.Flavor
	)

	connCompute, err := getServiceClient(ctx, "compute", cloudName)
//	log.Debugf("findFlavor: connCompute = %+v\n", connCompute)
	if err != nil {
		err = fmt.Errorf("findFlavor: getServiceClient returns %v", err)
		return
	}

	backoff := wait.Backoff{
		Duration: 1 * time.Minute,
		Factor:   1.1,
		Cap:      leftInContext(ctx),
		Steps:    math.MaxInt32,
	}

	err = wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
		var (
			err2 error
		)

		log.Debugf("findFlavor: duration = %v, calling flavors.ListDetail", leftInContext(ctx))
		pager, err2 = flavors.ListDetail(connCompute, flavors.ListOpts{}).AllPages(ctx)
		if err2 != nil {
			return false, nil
		}
//		log.Debugf("findFlavor: pager = %+v", pager)

		allFlavors, err2 = flavors.ExtractFlavors(pager)
		if err2 != nil {
			return false, nil
		}
//		log.Debugf("findFlavor: allFlavors = %+v", allFlavors)

		return true, nil
	})
	if err != nil {
		return
	}

	for _, flavor = range allFlavors {
//		log.Debugf("findFlavor: flavor.Name = %s, flavor.ID = %s", flavor.Name, flavor.ID)

		if flavor.Name == name {
			foundFlavor = flavor
			return
		}
	}

	err = fmt.Errorf("Could not find flavor named %s", name)
	return
}

func findImage(ctx context.Context, cloudName string, name string) (foundImage images.Image, err error) {
	var (
		pager      pagination.Page
		allImages  []images.Image
		image      images.Image
	)

	connImage, err := getServiceClient(ctx, "image", cloudName)
	if err != nil {
		err = fmt.Errorf("findImage: getServiceClient returns %v", err)
		return
	}
//	log.Debugf("findImage: connImage = %+v\n", connImage)

	backoff := wait.Backoff{
		Duration: 1 * time.Minute,
		Factor:   1.1,
		Cap:      leftInContext(ctx),
		Steps:    math.MaxInt32,
	}

	err = wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
		var (
			err2 error
		)

		log.Debugf("findImage: duration = %v, calling images.List", leftInContext(ctx))
		pager, err2 = images.List(connImage, images.ListOpts{}).AllPages(ctx)
		if err2 != nil {
			return false, nil
		}
//		log.Debugf("findImage: pager = %+v", pager)

		allImages, err2 = images.ExtractImages(pager)
		if err2 != nil {
			return false, nil
		}
//		log.Debugf("findImage: allImages = %+v", allImages)

		return true, nil
	})
	if err != nil {
		return
	}

	for _, image = range allImages {
		log.Debugf("findImage: image.Name = %s, image.ID = %s", image.Name, image.ID)

		if image.Name == name {
			foundImage = image
			return
		}
	}

	err = fmt.Errorf("Could not find image named %s", name)
	return
}

func findNetwork(ctx context.Context, cloudName string, name string) (foundNetwork networks.Network, err error) {
	var (
		pager      pagination.Page
		allNetworks  []networks.Network
		network      networks.Network
	)

	connNetwork, err := getServiceClient(ctx, "network", cloudName)
	if err != nil {
		err = fmt.Errorf("findNetwork: getServiceClient returns %v", err)
		return
	}
//	log.Debugf("findNetwork: connNetwork = %+v\n", connNetwork)

	backoff := wait.Backoff{
		Duration: 1 * time.Minute,
		Factor:   1.1,
		Cap:      leftInContext(ctx),
		Steps:    math.MaxInt32,
	}

	err = wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
		var (
			err2 error
		)

		log.Debugf("findNetwork: duration = %v, calling networks.List", leftInContext(ctx))
		pager, err2 = networks.List(connNetwork, networks.ListOpts{}).AllPages(ctx)
		if err2 != nil {
			return false, nil
		}
//		log.Debugf("findNetwork: pager = %+v", pager)

		allNetworks, err2 = networks.ExtractNetworks(pager)
		if err2 != nil {
			return false, nil
		}
//		log.Debugf("findNetwork: allNetworks = %+v", allNetworks)

		return true, nil
	})
	if err != nil {
		return
	}

	for _, network = range allNetworks {
		log.Debugf("findNetwork: network.Name = %s, network.ID = %s", network.Name, network.ID)

		if network.Name == name {
			foundNetwork = network
			return
		}
	}

	err = fmt.Errorf("Could not find network named %s", name)
	return
}

func findServer(ctx context.Context, cloudName string, name string) (foundServer servers.Server, err error) {
	var (
		allServers []servers.Server
		server     servers.Server
	)

	allServers, err = getAllServers(ctx, cloudName)
	if err != nil {
		err = fmt.Errorf("getAllServers returns %v", err)
		return
	}

	for _, server = range allServers {
		log.Debugf("findServer: server.Name = %s, server.ID = %s", server.Name, server.ID)

		if server.Name == name {
			foundServer = server
			return
		}
	}

	err = fmt.Errorf("Could not find server named %s", name)
	return
}

func waitForServer(ctx context.Context, cloudName string, name string) error {
	var (
		err error
	)

	backoff := wait.Backoff{
		Duration: 15 * time.Second,
		Factor:   1.1,
		Cap:      leftInContext(ctx),
		Steps:    math.MaxInt32,
	}

	err = wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
		var (
			foundServer servers.Server
			err2        error
		)

		// Check
		foundServer, err2 = findServer(ctx, cloudName, name)
		if err2 != nil {
			log.Debugf("waitForServer: findServer returned %v", err2)

			if strings.HasPrefix(err2.Error(), "Could not find server named") {
				return false, nil
			}

			return false, err2
		}

		log.Debugf("waitForServer: foundServer.Status = %s, foundServer.PowerState = %d", foundServer.Status, foundServer.PowerState)
		if foundServer.Status == "ACTIVE" && foundServer.PowerState == servers.RUNNING {
			log.Debugf("waitForServer: found server")
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return err
	}

	return nil
}

func getAllServers(ctx context.Context, cloud string) (allServers []servers.Server, err error) {
	var (
		connCompute *gophercloud.ServiceClient
		duration    time.Duration
		pager       pagination.Page
	)

	connCompute, err = getServiceClient(ctx, "compute", cloud)
	if err != nil {
		err = fmt.Errorf("getAllServers: getServiceClient returns %v", err)
		return
	}
//	log.Debugf("getAllServers: connCompute = %+v\n", connCompute)

	backoff := wait.Backoff{
		Duration: 1 * time.Minute,
		Factor:   1.1,
		Cap:      leftInContext(ctx),
		Steps:    math.MaxInt32,
	}

	err = wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
		var (
			err2 error
		)

		duration = leftInContext(ctx)
		log.Debugf("getAllServers: duration = %v, calling servers.List", duration)
		pager, err2 = servers.List(connCompute, nil).AllPages(ctx)
		if err2 != nil {
			log.Debugf("getAllServers: servers.List returned error %v", err2)

			// if strings.Contains(err2.Error(), "The request you have made requires authentication") {

			return false, nil
		}
//		log.Debugf("getAllServers: pager = %+v", pager)

		allServers, err2 = servers.ExtractServers(pager)
		if err2 != nil {
			log.Debugf("getAllServers: servers.ExtractServers returned error %v", err2)
			return false, nil
		}
//		log.Debugf("getAllServers: allServers = %+v", allServers)

		return true, nil
	})

	return
}

func findServerInList(allServers []servers.Server, name string) (foundServer servers.Server, err error) {
	var (
		server servers.Server
	)

	for _, server = range allServers {
		if server.Name == name {
			foundServer = server
			return
		}
	}

	err = fmt.Errorf("Could not find server named %s", name)
	return
}

func findKeyPair(ctx context.Context, cloudName string, name string) (foundKeyPair keypairs.KeyPair, err error) {
	var (
		pager       pagination.Page
		allKeyPairs []keypairs.KeyPair
		keypair     keypairs.KeyPair
	)

	connServer, err := getServiceClient(ctx, "compute", cloudName)
	if err != nil {
		err = fmt.Errorf("findKeyPair: getServiceClient returns %v", err)
		return
	}
//	log.Debugf("findKeyPair: connServer = %+v\n", connServer)

	backoff := wait.Backoff{
		Duration: 1 * time.Minute,
		Factor:   1.1,
		Cap:      leftInContext(ctx),
		Steps:    math.MaxInt32,
	}

	err = wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
		var (
			err2 error
		)

		log.Debugf("findKeyPair: duration = %v, calling keypairs.List", leftInContext(ctx))
		pager, err2 = keypairs.List(connServer, keypairs.ListOpts{}).AllPages(ctx)
		if err2 != nil {
			return false, nil
		}
//		log.Debugf("findKeyPair: pager = %+v", pager)

		allKeyPairs, err2 = keypairs.ExtractKeyPairs(pager)
		if err2 != nil {
			return false, nil
		}
//		log.Debugf("findKeyPair: allKeyPairs = %+v", allKeyPairs)

		return true, nil
	})
	if err != nil {
		return
	}

	for _, keypair = range allKeyPairs {
		log.Debugf("findKeyPair: keypair.Name = %s", keypair.Name)

		if keypair.Name == name {
			foundKeyPair = keypair
			return
		}
	}

	err = fmt.Errorf("Could not find keypair named %s", name)
	return
}

func findHypervisor(ctx context.Context, cloudName string, name string) (foundHypervisor hypervisors.Hypervisor, err error) {
	var (
		pager          pagination.Page
		allHypervisors []hypervisors.Hypervisor
		hypervisor     hypervisors.Hypervisor
	)

	connServer, err := getServiceClient(ctx, "compute", cloudName)
	if err != nil {
		err = fmt.Errorf("findHypervisor: getServiceClient returns %v", err)
		return
	}
//	log.Debugf("findHypervisor: connServer = %+v\n", connServer)

	backoff := wait.Backoff{
		Duration: 1 * time.Minute,
		Factor:   1.1,
		Cap:      leftInContext(ctx),
		Steps:    math.MaxInt32,
	}

	err = wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
		var (
			err2 error
		)

		log.Debugf("findHypervisor: duration = %v, calling flavors.ListDetail", leftInContext(ctx))
		pager, err2 = hypervisors.List(connServer, nil).AllPages(ctx)
		if err2 != nil {
			return false, nil
		}
//		log.Debugf("findHypervisor: pager = %+v", pager)

		allHypervisors, err2 = hypervisors.ExtractHypervisors(pager)
		if err2 != nil {
			return false, nil
		}
//		log.Debugf("findHypervisor: allHypervisors = %+v", allHypervisors)

		return true, nil
	})
	if err != nil {
		return
	}

	for _, hypervisor = range allHypervisors {
		log.Debugf("findHypervisor: hypervisor.HypervisorHostname = %s, hypervisor.ID = %s", hypervisor.HypervisorHostname, hypervisor.ID)

		if hypervisor.HypervisorHostname == name {
			foundHypervisor = hypervisor
			return
		}
	}

	err = fmt.Errorf("Could not find hypervisor named %s", name)
	return
}

func getAllHypervisors(ctx context.Context, connCompute *gophercloud.ServiceClient) (allHypervisors []hypervisors.Hypervisor, err error) {
	var (
		duration time.Duration
		pager    pagination.Page
	)

	backoff := wait.Backoff{
		Duration: 1 * time.Minute,
		Factor:   1.1,
		Cap:      leftInContext(ctx),
		Steps:    math.MaxInt32,
	}

	err = wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
		var (
			err2 error
		)

		duration = leftInContext(ctx)
		log.Debugf("getAllHypervisors: duration = %v, calling hypervisors.List", duration)
		pager, err2 = hypervisors.List(connCompute, nil).AllPages(ctx)
		if err2 != nil {
			log.Debugf("getAllHypervisors: hypervisors.List returned error %v", err2)

			if strings.Contains(err2.Error(), "The request you have made requires authentication") {
				return true, err2
			}

			return false, nil
		}
//		log.Debugf("getAllHypervisors: pager = %+v", pager)

		allHypervisors, err2 = hypervisors.ExtractHypervisors(pager)
		if err2 != nil {
			log.Debugf("getAllHypervisors: hypervisors.ExtractHypervisors returned error %v", err2)
			return false, nil
		}
//		log.Debugf("getAllHypervisors: allHypervisors = %+v", allHypervisors)

		return true, nil
	})

	return
}

func findHypervisorverInList(allHypervisors []hypervisors.Hypervisor, name string) (foundHypervisor hypervisors.Hypervisor, err error) {
	var (
		hypervisor hypervisors.Hypervisor
	)

	for _, hypervisor = range allHypervisors {
		if hypervisor.HypervisorHostname == name {
			foundHypervisor = hypervisor
			return
		}
	}

	err = fmt.Errorf("Could not find hypervisor named %s", name)
	return
}
