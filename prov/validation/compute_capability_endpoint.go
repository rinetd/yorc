// Copyright 2018 Bull S.A.S. Atos Technologies - Bull, Rue Jean Jaures, B.P.68, 78340, Les Clayes-sous-Bois, France.
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

package validation

import (
	"context"

	"github.com/ystia/yorc/tasks"

	"github.com/hashicorp/consul/api"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/events"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/tasks/workflow"
)

func postComputeCreationHook(ctx context.Context, cfg config.Configuration, taskID, deploymentID, target string, activity workflow.Activity) {

	if activity.Type() != workflow.ActivityTypeDelegate && activity.Type() != workflow.ActivityTypeCallOperation {
		return
	}
	cc, err := cfg.GetConsulClient()
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).
			Registerf("Failed to retrieve consul client when ensuring that a compute will have it's endpoint ip set. Next operations will likely failed: %v", target, err)
		return
	}
	kv := cc.KV()
	status, err := tasks.GetTaskStatus(kv, taskID)
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).
			Registerf("Failed to retrieve task status when ensuring that a compute will have it's endpoint ip set. Next operations will likely failed: %v", err)
		return
	}
	if status == tasks.FAILED || status == tasks.CANCELED {
		return
	}

	isCompute, err := deployments.IsNodeDerivedFrom(kv, deploymentID, target, "yorc.nodes.Compute")
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).
			Registerf("Failed to retrieve node type for node %q when ensuring that a compute will have it's endpoint ip set. Next operations will likely failed: %v", target, err)
		return
	}
	if !isCompute {
		return
	}
	instances, err := deployments.GetNodeInstancesIds(kv, deploymentID, target)
	if err != nil {
		events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).
			Registerf("Failed to retrieve node instances for node %q when ensuring that a compute will have it's endpoint ip set. Next operations will likely failed: %v", target, err)
		return
	}
	checkAllInstances(ctx, kv, deploymentID, target, instances)
}

func checkAllInstances(ctx context.Context, kv *api.KV, deploymentID, target string, instances []string) {
	for _, instance := range instances {
		found, _, err := deployments.GetInstanceCapabilityAttribute(kv, deploymentID, target, instance, "endpoint", "ip_address")
		if err != nil {
			events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).
				Registerf("Failed to retrieve node attribute for node %q when ensuring that a compute will have it's endpoint ip set. Next operations will likely failed: %v", target, err)
			return
		}
		if !found {
			// Check those attributes in order. Stop at the first found.
			for _, attr := range []string{"public_ip_address", "public_address", "private_address", "ip_address"} {
				found, err := setEndpointIPFromAttribute(ctx, kv, deploymentID, target, instance, attr)
				if err != nil {
					events.WithContextOptionalFields(ctx).NewLogEntry(events.WARN, deploymentID).
						Registerf("Failed to retrieve node attribute for node %q when ensuring that a compute will have it's endpoint ip set. Next operations will likely failed: %v", target, err)
					return
				}
				if found {
					break
				}
			}
		}
	}
}

func setEndpointIPFromAttribute(ctx context.Context, kv *api.KV, deploymentID, nodeName, instance, attribute string) (bool, error) {
	found, ip, err := deployments.GetInstanceAttribute(kv, deploymentID, nodeName, instance, attribute)
	if err != nil {
		return false, err
	}
	if found {
		err = deployments.SetInstanceCapabilityAttribute(deploymentID, nodeName, instance, "endpoint", "ip_address", ip)
		if err != nil {
			return false, err
		}
	}
	return found, nil
}

func init() {
	workflow.RegisterPostActivityHook(postComputeCreationHook)
}
