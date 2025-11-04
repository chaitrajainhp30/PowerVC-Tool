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
	"encoding/json"
	"io/ioutil"

	configv1 "github.com/openshift/api/config/v1"
)

type Metadata struct {
	createMetadata CreateMetadata
}

type CreateMetadata struct {
	ClusterName               string                       `json:"clusterName"`
	ClusterID                 string                       `json:"clusterID"`
	InfraID                   string                       `json:"infraID"`
	OSClusterPlatformMetadata                              `json:",inline"`
	PVClusterPlatformMetadata                              `json:",inline"`
	FeatureSet                configv1.FeatureSet          `json:"featureSet"`
	CustomFeatureSet          *configv1.CustomFeatureGates `json:"customFeatureSet"`
}

type OSClusterPlatformMetadata struct {
	OpenStack *OpenStackSMetadata `json:"openstack,omitempty"`
}

type PVClusterPlatformMetadata struct {
	PowerVC *OpenStackSMetadata `json:"powervc,omitempty"`
}

type OpenStackIdentifier struct {
	OpenshiftClusterID string `json:"openshiftClusterID"`
}

type OpenStackSMetadata struct {
	Cloud      string              `json:"cloud"`
	Identifier OpenStackIdentifier `json:"identifier"`
}

func NewMetadataFromCCMetadata(filename string) (*Metadata, error) {
	var (
		content  []byte
		metadata Metadata
		err      error
	)

	content, err = ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal("Error when opening file: ", err)
		return nil, err
	}

	log.Debugf("NewMetadataFromCCMetadata: content = %s", string(content))

	err = json.Unmarshal(content, &metadata.createMetadata)
	if err != nil {
		log.Fatal("Error during Unmarshal(): ", err)
		return nil, err
	}

	log.Debugf("NewMetadataFromCCMetadata: metadata = %+v", metadata)
	log.Debugf("NewMetadataFromCCMetadata: metadata.createMetadata = %+v", metadata.createMetadata)
	log.Debugf("NewMetadataFromCCMetadata: metadata.createMetadata.OpenStack = %+v", metadata.createMetadata.OpenStack)
	log.Debugf("NewMetadataFromCCMetadata: metadata.createMetadata.PowerVC = %+v", metadata.createMetadata.PowerVC)

	return &metadata, nil
}

func (m *Metadata) GetClusterName() string {
	return m.createMetadata.ClusterName
}

func (m *Metadata) GetInfraID() string {
	return m.createMetadata.InfraID
}

func (m *Metadata) GetCloud() string {
	if m.createMetadata.OpenStack != nil {
		return m.createMetadata.OpenStack.Cloud
	} else if m.createMetadata.PowerVC != nil {
		return m.createMetadata.PowerVC.Cloud
	} else {
		return ""
	}
}
