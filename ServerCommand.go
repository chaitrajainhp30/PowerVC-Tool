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
	"net"
)

type CommandHeader struct {
        Command string `json:"Command"`
}

type CommandSendMetadata struct {
	Command  string          `json:"Command"`
	Metadata CreateMetadata
}

type CommandCreateBastion struct {
	Command    string        `json:"Command"`
	CloudName  string        `json:"cloudName"`
	ServerName string        `json:"serverName"`
	DomainName string        `json:"domainName"`
}

func sendMetadata(metadataFile string, serverIP string, shouldCreateMetadata bool) error {
	var (
		content        []byte
		cmd            CommandSendMetadata
		marshalledData []byte
		err            error
	)

	// Avoid: address format "%s:%s" does not work with IPv6
	// Connect to the server
	conn, err := net.Dial("tcp", net.JoinHostPort(serverIP, "8080"))
	if err != nil {
		log.Debugf("sendMetadata: net.Dial return %v", err)
		return err
	}

	// Read metadata.json into a buffer
	content, err = ioutil.ReadFile(metadataFile)
	if err != nil {
		log.Debugf("sendMetadata: ioutil.ReadFile return %v", err)
		return err
	}
	log.Debugf("sendMetadata: content = %s", content)

	// Create the command JSON structure
	if shouldCreateMetadata {
		cmd.Command = "create-metadata"
	} else {
		cmd.Command = "delete-metadata"
	}
	err = json.Unmarshal(content, &cmd.Metadata)
	if err != nil {
		log.Debugf("sendMetadata: json.Unmarshal return %v", err)
		return err
	}
	log.Debugf("sendMetadata: cmd = %+v", cmd)

	marshalledData, err = json.Marshal(cmd)
	if err != nil {
		log.Debugf("sendMetadata: json.Marshal return %v", err)
		return err
	}
	log.Debugf("sendMetadata: marshalledData = %v", string(marshalledData))

	// Send some data to the server
	_, err = conn.Write(marshalledData)
	if err != nil {
		log.Debugf("sendMetadata: conn.Write return %v", err)
		return err
	}

	// Close the connection
	conn.Close()

	log.Debugf("sendMetadata: Done!")

	return nil
}

func sendCreateBastion(serverIP string, cloudName string, serverName string, domainName string) error {
	var (
		cmd            CommandCreateBastion
		marshalledData []byte
		err            error
	)

	cmd = CommandCreateBastion{
		Command:    "create-bastion",
		CloudName:  cloudName,
		ServerName: serverName,
		DomainName: domainName,
	}

	// Avoid: address format "%s:%s" does not work with IPv6
	// Connect to the server
	conn, err := net.Dial("tcp", net.JoinHostPort(serverIP, "8080"))
	if err != nil {
		log.Debugf("sendMetadata: net.Dial return %v", err)
		return err
	}

	marshalledData, err = json.Marshal(cmd)
	if err != nil {
		log.Debugf("sendCreateBastion: json.Marshal return %v", err)
		return err
	}
	log.Debugf("sendCreateBastion: marshalledData = %v", string(marshalledData))

	// Send some data to the server
	_, err = conn.Write(marshalledData)
	if err != nil {
		log.Debugf("sendCreateBastion: conn.Write return %v", err)
		return err
	}

	// Close the connection
	conn.Close()

	log.Debugf("sendCreateBastion: Done!")

	return nil
}
