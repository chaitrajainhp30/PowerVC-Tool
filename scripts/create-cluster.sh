#!/usr/bin/env bash

# Copyright 2025 IBM Corp
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# NOTE: IBMCLOUD_API_KEY is an optional environment variable
declare -a ENV_VARS
ENV_VARS=( "BASEDOMAIN" "BASTION_IMAGE_NAME" "BASTION_USERNAME" "CLOUD" "CLUSTER_DIR" "CLUSTER_NAME" "FLAVOR_NAME" "MACHINE_TYPE" "NETWORK_NAME" "RHCOS_IMAGE_NAME" "SSHKEY_NAME" )

for VAR in ${ENV_VARS[@]}
do
	if [[ ! -v ${VAR} ]]
	then
		echo "${VAR} must be set!"
		exit 1
	fi
	VALUE=$(eval "echo \"\${${VAR}}\"")
	if [[ -z "${VALUE}" ]]
	then
		echo "${VAR} must be set!"
		exit 1
	fi
done

set -euo pipefail

if ! hash PowerVC-Create-Cluster 1>/dev/null 2>&1
then
	echo "Error: Missing PowerVC-Create-Cluster program!"
	exit 1
fi

if ! hash openshift-install 1>/dev/null 2>&1
then
	echo "Error: Missing openshift-install program!"
	exit 1
fi

if ! hash openstack 1>/dev/null 2>&1
then
	echo "Error: Missing openstack program!"
	exit 1
fi

if ! hash jq 1>/dev/null 2>&1
then
	echo "Error: Missing jq program!"
	exit 1
fi

if ! openstack --os-cloud=${CLOUD} network show "${NETWORK_NAME}" --format shell > /dev/null 2>&1
then
	echo "Error: Is the OpenStack cloud (${CLOUD}) configured correctly?"
	exit 1
fi

SUBNET_ID=$(openstack --os-cloud=${CLOUD} network show "${NETWORK_NAME}" --format shell | grep ^subnets | sed -e "s,^[^']*',," -e "s,'.*$,,")
if [ -z "${SUBNET_ID}" ]
then
	echo "Error: SUBNET_ID is empty!"
	exit 1
fi

if [ -d ${CLUSTER_DIR} ]
then
	/bin/rm -rf ${CLUSTER_DIR}
fi
mkdir ${CLUSTER_DIR}

INSTALLER_SSHKEY=~/.ssh/id_installer_rsa.pub
if [ ! -f ${INSTALLER_SSHKEY} ]
then
	echo "Error: ${INSTALLER_SSHKEY} does not exist!"
	exit 1
fi
SSH_KEY=$(cat ${INSTALLER_SSHKEY})

PULLSECRET_FILE=~/.pullSecretCompact
if [ ! -f ${PULLSECRET_FILE} ]
then
	echo "Error: ${PULLSECRET_FILE} does not exist!"
	exit 1
fi
PULL_SECRET=$(cat ~/.pullSecretCompact)

PowerVC-Create-Cluster \
	create-bastion \
	--cloud "${CLOUD}" \
	--bastionName "${CLUSTER_NAME}" \
	--flavorName "${FLAVOR_NAME}" \
	--imageName "${BASTION_IMAGE_NAME}" \
	--networkName "${NETWORK_NAME}" \
	--sshKeyName "${SSHKEY_NAME}" \
	--domainName "${BASEDOMAIN}" \
	--shouldDebug true
RC=$?

if [ ${RC} -gt 0 ]
then
	echo "Error: PowerVC-Create-Cluster failed with an RC of ${RC}"
	exit 1
fi

if [ ! -f /tmp/bastionIp ]
then
	echo "Error: Expecting file /tmp/bastionIp"
	exit 1
fi

VIP_API=$(cat /tmp/bastionIp)
VIP_INGRESS=$(cat /tmp/bastionIp)

if [ -z "${VIP_API}" -o -z "${VIP_INGRESS}" ]
then
	echo "Error: VIP_API and VIP_INGRESS must be defined!"
	exit 1
fi

for (( TRIES=0; TRIES<=60; TRIES++ ))
do
	if getent ahostsv4 api.${CLUSTER_NAME}.${BASEDOMAIN} > /dev/null 2>&1
	then
		break
	else
		echo "Error: Cannot resolve api.${CLUSTER_NAME}.${BASEDOMAIN}"
		exit 1
	fi
	sleep 5s
done
IP=$(getent ahostsv4 api.${CLUSTER_NAME}.${BASEDOMAIN} | grep STREAM | cut -f1 -d' ')
if [ "${IP}" != "${VIP_API}" ]
then
	echo "Error: VIP_API (${VIP_API}) is not the same as ${IP}"
	exit 1
fi

#
# Create the openshift-installer's install configuration file
#
cat << ___EOF___ > ${CLUSTER_DIR}/install-config.yaml
apiVersion: v1
baseDomain: ${BASEDOMAIN}
compute:
- architecture: ppc64le
  hyperthreading: Enabled
  name: worker
  platform:
    powervc:
      zones:
        - ${MACHINE_TYPE}
  replicas: 3
controlPlane:
  architecture: ppc64le
  hyperthreading: Enabled
  name: master
  platform:
    powervc:
      zones:
        - ${MACHINE_TYPE}
  replicas: 3
metadata:
  creationTimestamp: null
  name: ${CLUSTER_NAME}
networking:
  clusterNetwork:
  - cidr: 10.128.0.0/14
    hostPrefix: 23
  machineNetwork:
  - cidr: 10.20.176.0/20
  networkType: OVNKubernetes
  serviceNetwork:
  - 172.30.0.0/16
platform:
  powervc:
    loadBalancer:
      type: UserManaged
    apiVIPs:
    - ${VIP_API}
    cloud: ${CLOUD}
    clusterOSImage: ${RHCOS_IMAGE_NAME}
    defaultMachinePlatform:
      type: ${FLAVOR_NAME}
    ingressVIPs:
    - ${VIP_INGRESS}
    controlPlanePort:
      fixedIPs:
        - subnet:
            id: ${SUBNET_ID}
credentialsMode: Passthrough
pullSecret: '${PULL_SECRET}'
sshKey: |
  ${SSH_KEY}
___EOF___

openshift-install version
RC=$?
if [ ${RC} -gt 0 ]
then
	exit 1
fi

openshift-install create install-config --dir=${CLUSTER_DIR}
RC=$?
if [ ${RC} -gt 0 ]
then
	exit 1
fi

openshift-install create ignition-configs --dir=${CLUSTER_DIR}
RC=$?
if [ ${RC} -gt 0 ]
then
	exit 1
fi

# By now, the infraID field in metadata.json is filled out
INFRA_ID=$(jq -r .infraID ${CLUSTER_DIR}/metadata.json)
echo "INFRA_ID=${INFRA_ID}"

#jq --arg NEW_INFRA_ID ${CLUSTER_NAME} -r -c '. | .infraID = $NEW_INFRA_ID' ${CLUSTER_DIR}/metadata.json
#jq --arg NEW_INFRA_ID ${CLUSTER_NAME} -r -c '. | .powervc.identifier.openshiftClusterID = $NEW_INFRA_ID' ${CLUSTER_DIR}/metadata.json

openshift-install create manifests --dir=${CLUSTER_DIR}
RC=$?
if [ ${RC} -gt 0 ]
then
	exit 1
fi

openshift-install create cluster --dir=${CLUSTER_DIR} --log-level=debug
RC=$?
if [ ${RC} -gt 0 ]
then
	exit 1
fi
