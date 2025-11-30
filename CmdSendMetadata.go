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
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"net"
	"os"

	"github.com/sirupsen/logrus"
)

func sendMetadataCommand(sendMetadataFlags *flag.FlagSet, args []string) error {
	var (
		out                  io.Writer
		ptrCreateMetadata    *string
		ptrDeleteMetadata    *string
		ptrServerIP          *string
		ptrShouldDebug       *string
		shouldCreateMetadata bool
		shouldDeleteMetadata bool
		metadataFile         string
		err                  error
	)

	ptrCreateMetadata = sendMetadataFlags.String("createMetadata", "", "Create the metadata of this file")
	ptrDeleteMetadata = sendMetadataFlags.String("deleteMetadata", "", "Delete the metadata of this file")
	ptrServerIP = sendMetadataFlags.String("serverIP", "", "The IP address of the server to send the command to")
	ptrShouldDebug = sendMetadataFlags.String("shouldDebug", "false", "Should output debug output")

	sendMetadataFlags.Parse(args)

	if ptrCreateMetadata != nil && *ptrCreateMetadata != "" {
		shouldCreateMetadata = true
		metadataFile = *ptrCreateMetadata
	}
	if ptrDeleteMetadata != nil && *ptrDeleteMetadata != "" {
		shouldDeleteMetadata = true
		metadataFile = *ptrDeleteMetadata
	}

	if !shouldCreateMetadata && !shouldDeleteMetadata {
		return fmt.Errorf("Error: Either --createMetadata or --deleteMetadata should be specified")
	}
	if ptrServerIP == nil || *ptrServerIP == "" {
		return fmt.Errorf("Error: --serverIP not specified")
	}

	switch strings.ToLower(*ptrShouldDebug) {
	case "true":
		shouldDebug = true
	case "false":
		shouldDebug = false
	default:
		return fmt.Errorf("Error: shouldDebug is not true/false (%s)\n", *ptrShouldDebug)
	}

	if shouldDebug {
		out = os.Stderr
	} else {
		out = io.Discard
	}
	log = &logrus.Logger{
		Out:       out,
		Formatter: new(logrus.TextFormatter),
		Level:     logrus.DebugLevel,
	}

	fmt.Fprintf(os.Stderr, "Program version is %v, release = %v\n", version, release)

	err = sendMetadata(metadataFile, *ptrServerIP, shouldCreateMetadata)

	return err
}

type CommandHeader struct {
        Command string `json:"Command"`
}

type CommandSendMetadata struct {
	Command  string          `json:"Command"`
	Metadata MinimalMetadata
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

	return nil
}
