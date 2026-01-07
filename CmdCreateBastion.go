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
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"strings"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/keypairs"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"

	"github.com/IBM/networking-go-sdk/dnsrecordsv1"

	"github.com/sirupsen/logrus"

	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/utils/ptr"
)

const (
	bastionIpFilename = "/tmp/bastionIp"
)

var (
	enableHAProxy  = true
)

func createBastionCommand(createBastionFlags *flag.FlagSet, args []string) error {
	var (
		out            io.Writer
		ptrCloud       *string
		ptrBastionName *string
		ptrFlavorName  *string
		ptrImageName   *string
		ptrNetworkName *string
		ptrSshKeyName  *string
		ptrDomainName  *string
		ptrEnableHAP   *string
		ptrServerIP    *string
		ptrShouldDebug *string
		ctx            context.Context
		cancel         context.CancelFunc
		err            error
	)

	ptrCloud = createBastionFlags.String("cloud", "", "The cloud to use in clouds.yaml")
	ptrBastionName = createBastionFlags.String("bastionName", "", "The name of the bastion VM to use")
	ptrFlavorName = createBastionFlags.String("flavorName", "", "The name of the flavor to use")
	ptrImageName = createBastionFlags.String("imageName", "", "The name of the image to use")
	ptrNetworkName = createBastionFlags.String("networkName", "", "The name of the network to use")
	ptrSshKeyName = createBastionFlags.String("sshKeyName", "", "The name of the ssh keypair to use")
	// NOTE: This is optional
	ptrDomainName = createBastionFlags.String("domainName", "", "The DNS domain to use")
	ptrEnableHAP = createBastionFlags.String("enableHAProxy", "false", "Should install and enable HA Proxy demon")
	ptrServerIP = createBastionFlags.String("serverIP", "", "The IP address of the server to send the command to")
	ptrShouldDebug = createBastionFlags.String("shouldDebug", "false", "Should output debug output")

	createBastionFlags.Parse(args)

	if ptrCloud == nil || *ptrCloud == "" {
		return fmt.Errorf("Error: --cloud not specified")
	}
	if ptrBastionName == nil || *ptrBastionName == "" {
		return fmt.Errorf("Error: --bastionName not specified")
	}
	if ptrFlavorName == nil || *ptrFlavorName == "" {
		return fmt.Errorf("Error: --flavorName not specified")
	}
	if ptrImageName == nil || *ptrImageName == "" {
		return fmt.Errorf("Error: --imageName not specified")
	}
	if ptrNetworkName == nil || *ptrNetworkName == "" {
		return fmt.Errorf("Error: --networkName not specified")
	}
	if ptrSshKeyName == nil || *ptrSshKeyName == "" {
		return fmt.Errorf("Error: --sshKeyName not specified")
	}

	switch strings.ToLower(*ptrEnableHAP) {
	case "true":
		enableHAProxy = true
	case "false":
		enableHAProxy = false
	default:
		return fmt.Errorf("Error: enableHAProxy is not true/false (%s)\n", *ptrEnableHAP)
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

	ctx, cancel = context.WithTimeout(context.TODO(), 15*time.Minute)
	defer cancel()

	err = os.Remove(bastionIpFilename)
	if err != nil {
		errstr := strings.TrimSpace(err.Error())
		if !strings.HasSuffix(errstr, "no such file or directory") {
			return err
		}
	}

	_, err = findServer(ctx, *ptrCloud, *ptrBastionName)
	if err != nil {
		log.Debugf("findServer(first) returns %+v", err)
		if strings.HasPrefix(err.Error(), "Could not find server named") {
			fmt.Printf("Could not find server %s, creating...\n", *ptrBastionName)

			err = createServer(ctx,
				*ptrCloud,
				*ptrFlavorName,
				*ptrImageName,
				*ptrNetworkName,
				*ptrSshKeyName,
				*ptrBastionName,
				nil,
			)
			if err != nil {
				return err
			}

			fmt.Println("Done!")
		} else {
			return err
		}
	}

	// It should exist now
	_, err = findServer(ctx, *ptrCloud, *ptrBastionName)
	if err != nil {
		log.Debugf("findServer(second) returns %+v", err)
		return err
	}

	if ptrServerIP != nil && *ptrServerIP != "" {
		// Ask to set it up remotely
		err = sendCreateBastion(*ptrServerIP, *ptrCloud, *ptrBastionName, *ptrDomainName)
		if err != nil {
			return err
		}
	} else {
		// Set it up locally
		err = setupBastionServer(ctx, *ptrCloud, *ptrBastionName, *ptrDomainName)
		if err != nil {
			log.Debugf("setupBastionServer returns %+v", err)
			return err
		}
	}

	return writeBastionIP(ctx, *ptrCloud, *ptrBastionName)
}

func createServer(ctx context.Context, cloudName string, flavorName string, imageName string, networkName string, sshKeyName string, bastionName string, userData []byte) error {
	var (
		flavor           flavors.Flavor
		image            images.Image
		network          networks.Network
		sshKeyPair       keypairs.KeyPair
		builder          ports.CreateOptsBuilder
		portCreateOpts   ports.CreateOpts
		portList         []servers.Network
		serverCreateOpts servers.CreateOptsBuilder
		newServer        *servers.Server
		err              error
	)

	flavor, err = findFlavor(ctx, cloudName, flavorName)
	if err != nil {
		return err
	}
	log.Debugf("flavor = %+v", flavor)

	image, err = findImage(ctx, cloudName, imageName)
	if err != nil {
		return err
	}
	log.Debugf("image = %+v", image)

	network, err = findNetwork(ctx, cloudName, networkName)
	if err != nil {
		return err
	}
	log.Debugf("network = %+v", network)

	if sshKeyName != "" {
		sshKeyPair, err = findKeyPair(ctx, cloudName, sshKeyName)
		if err != nil {
			return err
		}
	}

	connNetwork, err := NewServiceClient(ctx, "network", DefaultClientOpts(cloudName))
	if err != nil {
		return err
	}
	fmt.Printf("connNetwork = %+v\n", connNetwork)

	portCreateOpts = ports.CreateOpts{
		Name:                  fmt.Sprintf("%s-port", bastionName),
		NetworkID:		network.ID,
		Description:           "hamzy test",
		AdminStateUp:          nil,
		MACAddress:            ptr.Deref(nil, ""),
		AllowedAddressPairs:   nil,
		ValueSpecs:            nil,
		PropagateUplinkStatus: nil,
	}

	builder = portCreateOpts
	log.Debugf("builder = %+v\n", builder)

	port, err := ports.Create(ctx, connNetwork, builder).Extract()
	if err != nil {
		return err
	}
	log.Debugf("port = %+v\n", port)
	log.Debugf("port.ID = %v\n", port.ID)

	connCompute, err := NewServiceClient(ctx, "compute", DefaultClientOpts(cloudName))
	if err != nil {
		return err
	}
	fmt.Printf("connCompute = %+v\n", connCompute)

	portList = []servers.Network{
		{ Port: port.ID, },
	}

	serverCreateOpts = servers.CreateOpts{
		AvailabilityZone: "s1022",
		FlavorRef:        flavor.ID,
		ImageRef:         image.ID,
		Name:             bastionName,
		Networks:         portList,
		UserData:         userData,
		// Additional properties are not allowed ('tags' was unexpected)
//		Tags:             tags[:],
//              KeyName:          "",
//
//		Metadata:         instanceSpec.Metadata,
//		ConfigDrive:      &instanceSpec.ConfigDrive,
//		BlockDevice:      blockDevices,
	}
	log.Debugf("serverCreateOpts = %+v\n", serverCreateOpts)

	if sshKeyName != "" {
		newServer, err = servers.Create(ctx,
			connCompute,
			keypairs.CreateOptsExt{
				CreateOptsBuilder: serverCreateOpts,
				KeyName:           sshKeyPair.Name,
			},
			nil).Extract()
	} else {
		newServer, err = servers.Create(ctx, connCompute, serverCreateOpts, nil).Extract()
	}
	if err != nil {
		return err
	}
	log.Debugf("newServer = %+v\n", newServer)

	err = waitForServer(ctx, cloudName, bastionName)
	log.Debugf("waitForServer = %v\n", err)
	if err != nil {
		return err
	}

	return err
}

func setupBastionServer(ctx context.Context, cloudName string, serverName string, domainName string) error {
	var (
		server       servers.Server
		ipAddress    string
		homeDir      string
		installerRsa string
		outb         []byte
		outs         string
		exitError    *exec.ExitError
		apiKey       string
		err          error
	)

	server, err = findServer(ctx, cloudName, serverName)
	log.Debugf("setupBastionServer: server = %+v", server)
	if err != nil {
		return err
	}

	_, ipAddress, err = findIpAddress(server)
	if err != nil {
		return err
	}
	if ipAddress == "" {
		return fmt.Errorf("ip address is empty for server %s", server.Name)
	}

	log.Debugf("setupBastionServer: ipAddress = %s", ipAddress)

	homeDir, err = os.UserHomeDir()
	if err != nil {
		return err
	}
	log.Debugf("setupBastionServer: homeDir = %s", homeDir)

	installerRsa = path.Join(homeDir, ".ssh/id_installer_rsa")
	log.Debugf("setupBastionServer: installerRsa = %s", installerRsa)

	outb, err = runSplitCommand2([]string{
		"ssh-keygen",
		"-H",
		"-F",
		ipAddress,
	})
	outs = strings.TrimSpace(string(outb))
	log.Debugf("setupBastionServer: outs = \"%s\"", outs)
	if errors.As(err, &exitError) {
		log.Debugf("setupBastionServer: exitError.ExitCode() = %+v\n", exitError.ExitCode())

		log.Debugf("setupBastionServer: %v", exitError.ExitCode() == 1)
		if exitError.ExitCode() == 1 {

			outb, err = keyscanServer(ctx, ipAddress, false)
			if err != nil {
				return err
			}

			knownHosts := path.Join(homeDir, ".ssh/known_hosts")
			log.Debugf("setupBastionServer: knownHosts = %s", knownHosts)

			fileKnownHosts, err := os.OpenFile(knownHosts, os.O_APPEND|os.O_RDWR, 0644)
			if err != nil {
				return err
			}

			fileKnownHosts.Write(outb)

			defer fileKnownHosts.Close()
		}
	}

	fmt.Printf("Setting up server %s...\n", server.Name)

	if enableHAProxy {
		outb, err = runSplitCommand2([]string{
			"ssh",
			"-i",
			installerRsa,
			fmt.Sprintf("cloud-user@%s", ipAddress),
			"rpm",
			"-q",
			"haproxy",
		})
		outs = strings.TrimSpace(string(outb))
		log.Debugf("setupBastionServer: outs = \"%s\"", outs)
		if errors.As(err, &exitError) {
			log.Debugf("setupBastionServer: exitError.ExitCode() = %+v\n", exitError.ExitCode())

			if exitError.ExitCode() == 1 && outs == "package haproxy is not installed" {
				outb, err = runSplitCommand2([]string{
					"ssh",
					"-i",
					installerRsa,
					fmt.Sprintf("cloud-user@%s", ipAddress),
					"sudo",
					"dnf",
					"install",
					"-y",
					"haproxy",
				})
				outs = strings.TrimSpace(string(outb))
				log.Debugf("setupBastionServer: outs = %s", outs)
				log.Debugf("setupBastionServer: err = %+v", err)
			}
		} else if err != nil {
			log.Debugf("setupBastionServer: err = %+v", err)
			return err
		}

		outb, err = runSplitCommand2([]string{
			"ssh",
			"-i",
			installerRsa,
			fmt.Sprintf("cloud-user@%s", ipAddress),
			"sudo",
			"stat",
			"-c",
			"%a",
			"/etc/haproxy/haproxy.cfg",
		})
		outs = strings.TrimSpace(string(outb))
		log.Debugf("setupBastionServer: outb = \"%s\"", outs)
		if err != nil {
			log.Debugf("setupBastionServer: err = %+v", err)
			return err
		}
		if outs != "646" {
			outb, err = runSplitCommand2([]string{
				"ssh",
				"-i",
				installerRsa,
				fmt.Sprintf("cloud-user@%s", ipAddress),
				"sudo",
				"chmod",
				"646",
				"/etc/haproxy/haproxy.cfg",
			})
			outs = strings.TrimSpace(string(outb))
			log.Debugf("setupBastionServer: outb = \"%s\"", outs)
			if err != nil {
				log.Debugf("setupBastionServer: err = %+v", err)
				return err
			}
		}

		outb, err = runSplitCommand2([]string{
			"ssh",
			"-i",
			installerRsa,
			fmt.Sprintf("cloud-user@%s", ipAddress),
			"sudo",
			"getsebool",
			"haproxy_connect_any",
		})
		outs = strings.TrimSpace(string(outb))
		log.Debugf("setupBastionServer: outb = \"%s\"", outs)
		if err != nil {
			log.Debugf("setupBastionServer: err = %+v", err)
			return err
		}
		if outs != "haproxy_connect_any --> on" {
			outb, err = runSplitCommand2([]string{
				"ssh",
				"-i",
				installerRsa,
				fmt.Sprintf("cloud-user@%s", ipAddress),
				"sudo",
				"setsebool",
				"-P",
				"haproxy_connect_any=1",
			})
			outs = strings.TrimSpace(string(outb))
			log.Debugf("setupBastionServer: outb = \"%s\"", outs)
			if err != nil {
				log.Debugf("setupBastionServer: err = %+v", err)
				return err
			}
		}

		outb, err = runSplitCommand2([]string{
			"ssh",
			"-i",
			installerRsa,
			fmt.Sprintf("cloud-user@%s", ipAddress),
			"sudo",
			"systemctl",
			"enable",
			"haproxy.service",
		})
		outs = strings.TrimSpace(string(outb))
		log.Debugf("setupBastionServer: outb = \"%s\"", outs)
		if err != nil {
			log.Debugf("setupBastionServer: err = %+v", err)
			return err
		}

		outb, err = runSplitCommand2([]string{
			"ssh",
			"-i",
			installerRsa,
			fmt.Sprintf("cloud-user@%s", ipAddress),
			"sudo",
			"systemctl",
			"start",
			"haproxy.service",
		})
		outs = strings.TrimSpace(string(outb))
		log.Debugf("setupBastionServer: outb = \"%s\"", outs)
		if err != nil {
			log.Debugf("setupBastionServer: err = %+v", err)
			return err
		}
	}

	// NOTE: This is optional
	apiKey = os.Getenv("IBMCLOUD_API_KEY")

	if apiKey != "" {
		err = dnsForServer(ctx, cloudName, apiKey, serverName, domainName)
		if err != nil {
			return err
		}
	} else {
		fmt.Println("Warning: IBMCLOUD_API_KEY not set.  Make sure DNS is supported via another way.")
	}

	return err
}

func writeBastionIP(ctx context.Context, cloudName string, serverName string) error {
	var (
		server       servers.Server
		ipAddress    string
		err          error
	)

	server, err = findServer(ctx, cloudName, serverName)
	log.Debugf("writeBastionIP: server = %+v", server)
	if err != nil {
		return err
	}

	_, ipAddress, err = findIpAddress(server)
	if err != nil {
		return err
	}
	if ipAddress == "" {
		return fmt.Errorf("ip address is empty for server %s", server.Name)
	}

	log.Debugf("writeBastionIP: ipAddress = %s", ipAddress)

	fileBastionIp, err := os.OpenFile(bastionIpFilename, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	fileBastionIp.Write([]byte(ipAddress))

	defer fileBastionIp.Close()

	return nil
}

func removeCommentLines(input string) string {
	var (
		inputLines  []string
		resultLines []string
	)

	log.Debugf("removeCommentLines: input = \"%s\"", input)

	inputLines = strings.Split(input, "\n")

	for _, line := range inputLines {
		if !strings.HasPrefix(line, "#") {
			resultLines = append(resultLines, line)
		}
	}

	log.Debugf("removeCommentLines: resultLines = \"%s\"", resultLines)

	return strings.Join(resultLines, "\n")
}

func keyscanServer(ctx context.Context, ipAddress string, silent bool) ([]byte, error) {
	var (
		outb []byte
		outs string
		err  error
	)

	backoff := wait.Backoff{
		Duration: 1 * time.Second,
		Factor:   1.1,
		Cap:      leftInContext(ctx),
		Steps:    math.MaxInt32,
	}

	err = wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
		var (
			err2 error
		)

		outb, err2 = runSplitCommandNoErr([]string{
			"ssh-keyscan",
			ipAddress,
		},
			silent)
		outs = strings.TrimSpace(string(outb))
		log.Debugf("keyscanServer: outs = %s", outs)
		if err2 != nil {
			return false, nil
		}

		return true, nil
	})

	if err == nil {
		// Get rid of the comment lines generated by ssh-keyscan
		outLines := removeCommentLines(outs)
		outb = []byte(outLines)
	}

	return outb, err
}

func dnsForServer(ctx context.Context, cloudName string, apiKey string, bastionName string, domainName string) error {
	var (
		server       servers.Server
		ipAddress    string
		cisServiceID string
		crnstr       string
		zoneID       string
		dnsService   *dnsrecordsv1.DnsRecordsV1
		err          error
	)

	server, err = findServer(ctx, cloudName, bastionName)
	if err != nil {
		return err
	}
//	log.Debugf("server = %+v", server)

	_, ipAddress, err = findIpAddress(server)
	if err != nil {
		return err
	}
	if ipAddress == "" {
		return fmt.Errorf("ip address is empty for server %s", server.Name)
	}

	cisServiceID, _, err = getServiceInfo(ctx, apiKey, "internet-svcs", "")
	if err != nil {
		log.Errorf("getServiceInfo returns %v", err)
		return err
	}
	log.Debugf("dnsForServer: cisServiceID = %s", cisServiceID)

	crnstr, zoneID, err = getDomainCrn(ctx, apiKey, cisServiceID, domainName)
	log.Debugf("dnsForServer: crnstr = %s, zoneID = %s, err = %+v", crnstr, zoneID, err)
	if err != nil {
		log.Errorf("getDomainCrn returns %v", err)
		return err
	}

	dnsService, err = loadDnsServiceAPI(apiKey, crnstr, zoneID)
	if err != nil {
		return err
	}
	log.Debugf("dnsForServer: dnsService = %+v", dnsService)

	err = createOrDeletePublicDNSRecord(ctx,
		dnsrecordsv1.CreateDnsRecordOptions_Type_A,
		fmt.Sprintf("api.%s.%s", bastionName, domainName),
		ipAddress,
		true,
		dnsService)
	err = createOrDeletePublicDNSRecord(ctx,
		dnsrecordsv1.CreateDnsRecordOptions_Type_A,
		fmt.Sprintf("api-int.%s.%s", bastionName, domainName),
		ipAddress,
		true,
		dnsService)
	err = createOrDeletePublicDNSRecord(ctx,
		dnsrecordsv1.CreateDnsRecordOptions_Type_Cname,
		fmt.Sprintf("*.apps.%s.%s", bastionName, domainName),
		fmt.Sprintf("api.%s.%s", bastionName, domainName),
		true,
		dnsService)

	return nil
}

func leftInContext(ctx context.Context) time.Duration {
	deadline, ok := ctx.Deadline()
	if !ok {
		return math.MaxInt64
	}

	duration := time.Until(deadline)

	return duration
}
