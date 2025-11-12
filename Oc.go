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
	"fmt"
)

const (
	OcName = "OpenShiftCluster"
)

type Oc struct {
	services *Services
}

func NewOc(services *Services) ([]RunnableObject, []error) {
	var (
		ocs  []*Oc
		errs []error
		ros  []RunnableObject
	)

	ocs, errs = innerNewOc(services)

	ros = make([]RunnableObject, len(ocs))
	// Go does not support type converting the entire array.
	// So we do it manually.
	for i, v := range ocs {
		ros[i] = RunnableObject(v)
	}

	return ros, errs
}

func NewOcAlt(services *Services) ([]*Oc, []error) {
	return innerNewOc(services)
}

func innerNewOc(services *Services) ([]*Oc, []error) {
	var (
		ocs  []*Oc
		errs []error
	)

	ocs = make([]*Oc, 1)
	errs = make([]error, 1)

	ocs[0] = &Oc{
		services: services,
	}

	return ocs, errs
}

func (oc *Oc) Name() (string, error) {
	return OcName, nil
}

func (oc *Oc) ObjectName() (string, error) {
	return OcName, nil
}

func (oc *Oc) Run() error {
	// Nothing needs to be done here.
	return nil
}

func (oc *Oc) ClusterStatus() {
	var (
		cmds       = []string{
			"oc --request-timeout=5s get clusterversion",
			"oc --request-timeout=5s get co",
			"oc --request-timeout=5s get nodes -o=wide",
			"oc --request-timeout=5s get pods -n openshift-machine-api",
			"oc --request-timeout=5s get machines.machine.openshift.io -n openshift-machine-api",
			"oc --request-timeout=5s get machineset.machine.openshift.io -n openshift-machine-api",
			"oc --request-timeout=5s logs -l k8s-app=controller -c machine-controller -n openshift-machine-api",
			"oc --request-timeout=5s describe co/cloud-controller-manager",
			"oc --request-timeout=5s describe cm/cloud-provider-config -n openshift-config",
//			"oc --request-timeout=5s get pod -l k8s-app=cloud-manager-operator -n openshift-cloud-controller-manager-operator",
			"oc --request-timeout=5s get pods -n openshift-cloud-controller-manager-operator",
//			"oc --request-timeout=5s describe pod -l k8s-app=openstack-cloud-controller-manager -n openshift-cloud-controller-manager",
			"oc --request-timeout=5s get events -n openshift-cloud-controller-manager",
			"oc --request-timeout=5s -n openshift-cloud-controller-manager-operator logs deployment/cluster-cloud-controller-manager-operator -c cluster-cloud-controller-manager",
			"oc --request-timeout=5s get co/network",
			"oc --request-timeout=5s get co/kube-controller-manager",
			"oc --request-timeout=5s get co/etcd",
			"oc --request-timeout=5s get machines.machine.openshift.io -n openshift-machine-api",
			"oc --request-timeout=5s get machineset.m -n openshift-machine-api",
			"oc --request-timeout=5s get pods -n openshift-machine-api",
			"oc --request-timeout=5s get pods -n openshift-kube-controller-manager",
			"oc --request-timeout=5s get pods -n openshift-ovn-kubernetes",
			"oc --request-timeout=5s describe co/machine-config",
			// oc logs kube-controller-manager-rdr-hamzy-openstack-nxqfq-master-0 -n openshift-kube-controller-manager -c kube-controller-manager | grep 'context deadline exceeded'
		}
		pipeCmds   = [][]string{
			{
				"oc --request-timeout=5s get pods -A -o=wide",
				"sed -e /\\(Running\\|Completed\\)/d",
			},
			{
				"oc get csr",
				"grep Pending",
			},
		}
		kubeConfig string
		err        error
	)

	kubeConfig = oc.services.GetKubeConfig()

	for _, cmd := range cmds {
		err = runCommand(kubeConfig, cmd)
		if err != nil {
			fmt.Printf("Error: could not run command: %v\n", err)
		}
	}

	for _, twoCmds := range pipeCmds {
		err = runTwoCommands(kubeConfig, twoCmds[0], twoCmds[1])
		if err != nil {
			fmt.Printf("Error: could not run command: %v\n", err)
		}
	}
}

func (oc *Oc) Priority() (int, error) {
	return -1, nil
}
