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
	"time"

	"github.com/IBM/go-sdk-core/v5/core"

	"github.com/IBM/networking-go-sdk/dnsrecordsv1"
	"github.com/IBM/networking-go-sdk/zonesv1"

	"github.com/IBM/platform-services-go-sdk/globalcatalogv1"
	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"

	"k8s.io/apimachinery/pkg/util/wait"
)

func listResourceInstances(ctx context.Context, controllerSvc *resourcecontrollerv2.ResourceControllerV2, listResourceOptions *resourcecontrollerv2.ListResourceInstancesOptions) (resources *resourcecontrollerv2.ResourceInstancesList, response *core.DetailedResponse, err error) {
	backoff := wait.Backoff{
		Duration: 15 * time.Second,
		Factor:   1.1,
		Cap:      leftInContext(ctx),
		Steps:    math.MaxInt32,
	}

	err = wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
		var (
			err2 error
		)

		// https://github.com/IBM/platform-services-go-sdk/blob/main/resourcecontrollerv2/resource_controller_v2.go#L5008
		resources, response, err2 = controllerSvc.ListResourceInstancesWithContext(ctx, listResourceOptions)
		if err2 != nil {
			err2 = fmt.Errorf("ListResourceInstancesWithContext failed with: %v", err2)

			return false, err2
		}

		return true, nil
	})

	return
}

func listCatalogEntries(ctx context.Context, gcv1 *globalcatalogv1.GlobalCatalogV1, listCatalogEntriesOpt *globalcatalogv1.ListCatalogEntriesOptions) (result *globalcatalogv1.EntrySearchResult, response *core.DetailedResponse, err error) {
	backoff := wait.Backoff{
		Duration: 15 * time.Second,
		Factor:   1.1,
		Cap:      leftInContext(ctx),
		Steps:    math.MaxInt32,
	}

	err = wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
		var (
			err2 error
		)

		result, response, err2 = gcv1.ListCatalogEntriesWithContext(ctx, listCatalogEntriesOpt)
		if err2 != nil {
			err2 = fmt.Errorf("ListCatalogEntriesWithContex failed with: %v", err2)

			return false, err2
		}

		return true, nil
	})

	return
}

func GetChildObjects(ctx context.Context, gcv1 *globalcatalogv1.GlobalCatalogV1, getChildOpt *globalcatalogv1.GetChildObjectsOptions) (result *globalcatalogv1.EntrySearchResult, response *core.DetailedResponse, err error) {
	backoff := wait.Backoff{
		Duration: 15 * time.Second,
		Factor:   1.1,
		Cap:      leftInContext(ctx),
		Steps:    math.MaxInt32,
	}

	err = wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
		var (
			err2 error
		)

		result, response, err2 = gcv1.GetChildObjectsWithContext(ctx, getChildOpt)
		if err2 != nil {
			err2 = fmt.Errorf("GetChildObjects failed with: %v", err2)

			return false, err2
		}

		return true, nil
	})

	return
}

func listZones(ctx context.Context, zv1 *zonesv1.ZonesV1, listOpts *zonesv1.ListZonesOptions) (zoneList *zonesv1.ListZonesResp, response *core.DetailedResponse, err error) {
	backoff := wait.Backoff{
		Duration: 15 * time.Second,
		Factor:   1.1,
		Cap:      leftInContext(ctx),
		Steps:    math.MaxInt32,
	}

	err = wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
		var (
			err2 error
		)

		zoneList, response, err2 = zv1.ListZonesWithContext(ctx, listOpts)
		if err2 != nil {
			err2 = fmt.Errorf("ListZonesWithContext failed with: %v", err2)

			return false, err2
		}

		return true, nil
	})

	return
}

func listAllDnsRecords(ctx context.Context, dnsService *dnsrecordsv1.DnsRecordsV1, listOpts *dnsrecordsv1.ListAllDnsRecordsOptions) (result *dnsrecordsv1.ListDnsrecordsResp, response *core.DetailedResponse, err error) {
	backoff := wait.Backoff{
		Duration: 15 * time.Second,
		Factor:   1.1,
		Cap:      leftInContext(ctx),
		Steps:    math.MaxInt32,
	}

	err = wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
		var (
			err2 error
		)

		result, response, err2 = dnsService.ListAllDnsRecordsWithContext(ctx, listOpts)
		if err2 != nil {
			err2 = fmt.Errorf("ListAllDnsRecordsWithContext failed with: %v", err2)

			return false, err2
		}

		return true, nil
	})

	return
}

func deleteDnsRecord(ctx context.Context, dnsService *dnsrecordsv1.DnsRecordsV1, deleteOpts *dnsrecordsv1.DeleteDnsRecordOptions) (result *dnsrecordsv1.DeleteDnsrecordResp, response *core.DetailedResponse, err error) {
	backoff := wait.Backoff{
		Duration: 15 * time.Second,
		Factor:   1.1,
		Cap:      leftInContext(ctx),
		Steps:    math.MaxInt32,
	}

	err = wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
		var (
			err2 error
		)

		result, response, err2 = dnsService.DeleteDnsRecordWithContext(ctx, deleteOpts)
		if err2 != nil {
			err2 = fmt.Errorf("DeleteDnsRecordWithContext failed with: %v", err2)

			return false, err2
		}

		return true, nil
	})

	return
}

func createDnsRecord(ctx context.Context, dnsService *dnsrecordsv1.DnsRecordsV1, createOpts *dnsrecordsv1.CreateDnsRecordOptions) (result *dnsrecordsv1.DnsrecordResp, response *core.DetailedResponse, err error) {
	backoff := wait.Backoff{
		Duration: 15 * time.Second,
		Factor:   1.1,
		Cap:      leftInContext(ctx),
		Steps:    math.MaxInt32,
	}

	err = wait.ExponentialBackoffWithContext(ctx, backoff, func(context.Context) (bool, error) {
		var (
			err2 error
		)

		result, response, err2 = dnsService.CreateDnsRecordWithContext(ctx, createOpts)
		if err2 != nil {
			err2 = fmt.Errorf("CreateDnsRecordWithContext failed with: %v", err2)

			return false, err2
		}

		return true, nil
	})

	return
}
