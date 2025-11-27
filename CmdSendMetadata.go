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
		out            io.Writer
		ptrMetadata    *string
		ptrServerIP    *string
		ptrServerPort  *string
		ptrShouldDebug *string
		err            error
	)

	ptrMetadata = sendMetadataFlags.String("metadata", "", "The location of the metadata.json file")
	ptrServerIP = sendMetadataFlags.String("serverIP", "", "The IP address of the server to send the file to")
	ptrServerPort = sendMetadataFlags.String("serverPort", "", "The port of the server to send the file to")
	ptrShouldDebug = sendMetadataFlags.String("shouldDebug", "false", "Should output debug output")

	sendMetadataFlags.Parse(args)

	if ptrMetadata == nil || *ptrMetadata == "" {
		return fmt.Errorf("Error: --metadata not specified")
	}
	if ptrServerIP == nil || *ptrServerIP == "" {
		return fmt.Errorf("Error: --serverIP not specified")
	}
	if ptrServerPort == nil || *ptrServerPort == "" {
		return fmt.Errorf("Error: --serverPort not specified")
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

	err = sendMetadata(*ptrMetadata, *ptrServerIP, *ptrServerPort)

	return err
}

func sendMetadata(metadataFile string, serverIP string, serverPort string) error {
	var (
		content []byte
		err     error
	)

	// Avoid: address format "%s:%s" does not work with IPv6
	// Connect to the server
	conn, err := net.Dial("tcp", net.JoinHostPort(serverIP, serverPort))
	if err != nil {
		return err
	}

	content, err = ioutil.ReadFile(metadataFile)
	if err != nil {
		return err
	}
	log.Debugf("sendMetadata: content = %s", content)

	// Send some data to the server
	_, err = conn.Write(content)
	if err != nil {
		return err
	}

	// Close the connection
	conn.Close()

	return nil
}
