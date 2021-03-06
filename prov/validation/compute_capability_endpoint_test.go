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
	"fmt"
	"path"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/satori/go.uuid"
	"github.com/ystia/yorc/helper/consulutil"
	"github.com/ystia/yorc/tasks"

	"github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/require"

	"github.com/ystia/yorc/config"
	"github.com/ystia/yorc/deployments"
	"github.com/ystia/yorc/tasks/workflow"
)

type mockActivity struct {
	t workflow.ActivityType
	v string
}

func (m *mockActivity) Type() workflow.ActivityType {
	return m.t
}

func (m *mockActivity) Value() string {
	return m.v
}

func testPostComputeCreationHook(t *testing.T, kv *api.KV, cfg config.Configuration) {
	ctx := context.Background()

	target := "Compute"
	type args struct {
		taskStatus   tasks.TaskStatus
		activityType workflow.ActivityType
		endpointIPs  []string
		attributes   []map[string]string
	}
	tests := []struct {
		name   string
		args   args
		checks []string
	}{
		{"TestEndpointFromPublicIP", args{tasks.RUNNING, workflow.ActivityTypeDelegate, nil, []map[string]string{
			map[string]string{"public_ip_address": "10.0.0.1", "public_address": "10.0.0.2", "private_address": "10.0.0.3", "ip_address": "10.0.0.4"},
			map[string]string{"public_ip_address": "10.0.1.1", "public_address": "10.0.1.2", "private_address": "10.0.1.3", "ip_address": "10.0.1.4"},
			map[string]string{"public_ip_address": "10.0.2.1", "public_address": "10.0.2.2", "private_address": "10.0.2.3", "ip_address": "10.0.2.4"},
			map[string]string{"public_ip_address": "10.0.3.1", "public_address": "10.0.3.2", "private_address": "10.0.3.3", "ip_address": "10.0.3.4"},
			map[string]string{"public_ip_address": "10.0.4.1", "public_address": "10.0.4.2", "private_address": "10.0.4.3", "ip_address": "10.0.4.4"},
		}},
			[]string{"10.0.0.1", "10.0.1.1", "10.0.2.1", "10.0.3.1", "10.0.4.1"},
		},
		{"TestEndpointFromPublicAddr", args{tasks.RUNNING, workflow.ActivityTypeCallOperation, nil, []map[string]string{
			map[string]string{"public_address": "10.0.0.2", "private_address": "10.0.0.3", "ip_address": "10.0.0.4"},
			map[string]string{"public_address": "10.0.1.2", "private_address": "10.0.1.3", "ip_address": "10.0.1.4"},
			map[string]string{"public_address": "10.0.2.2", "private_address": "10.0.2.3", "ip_address": "10.0.2.4"},
			map[string]string{"public_address": "10.0.3.2", "private_address": "10.0.3.3", "ip_address": "10.0.3.4"},
			map[string]string{"public_address": "10.0.4.2", "private_address": "10.0.4.3", "ip_address": "10.0.4.4"},
		}},
			[]string{"10.0.0.2", "10.0.1.2", "10.0.2.2", "10.0.3.2", "10.0.4.2"},
		},
		{"TestEndpointFromPrivateAdd", args{tasks.RUNNING, workflow.ActivityTypeDelegate, nil, []map[string]string{
			map[string]string{"private_address": "10.0.0.3", "ip_address": "10.0.0.4"},
			map[string]string{"private_address": "10.0.1.3", "ip_address": "10.0.1.4"},
			map[string]string{"private_address": "10.0.2.3", "ip_address": "10.0.2.4"},
			map[string]string{"private_address": "10.0.3.3", "ip_address": "10.0.3.4"},
			map[string]string{"private_address": "10.0.4.3", "ip_address": "10.0.4.4"},
		}},
			[]string{"10.0.0.3", "10.0.1.3", "10.0.2.3", "10.0.3.3", "10.0.4.3"},
		},
		{"TestEndpointFromIPAdd", args{tasks.RUNNING, workflow.ActivityTypeCallOperation, nil, []map[string]string{
			map[string]string{"ip_address": "10.0.0.4"},
			map[string]string{"ip_address": "10.0.1.4"},
			map[string]string{"ip_address": "10.0.2.4"},
			map[string]string{"ip_address": "10.0.3.4"},
			map[string]string{"ip_address": "10.0.4.4"},
		}},
			[]string{"10.0.0.4", "10.0.1.4", "10.0.2.4", "10.0.3.4", "10.0.4.4"},
		},
		{"TestEndpointAlreadySet", args{tasks.RUNNING, workflow.ActivityTypeDelegate, []string{"10.1.0.4", "10.1.1.4", "10.1.2.4", "10.1.3.4", "10.1.4.4"}, []map[string]string{
			map[string]string{"ip_address": "10.0.0.4"},
			map[string]string{"ip_address": "10.0.1.4"},
			map[string]string{"ip_address": "10.0.2.4"},
			map[string]string{"ip_address": "10.0.3.4"},
			map[string]string{"ip_address": "10.0.4.4"},
		}},
			[]string{"10.1.0.4", "10.1.1.4", "10.1.2.4", "10.1.3.4", "10.1.4.4"},
		},
		{"TestEndpointTaskFailed", args{tasks.FAILED, workflow.ActivityTypeDelegate, nil, []map[string]string{
			map[string]string{"ip_address": "10.0.0.4"},
			map[string]string{"ip_address": "10.0.1.4"},
			map[string]string{"ip_address": "10.0.2.4"},
			map[string]string{"ip_address": "10.0.3.4"},
			map[string]string{"ip_address": "10.0.4.4"},
		}},
			[]string{"", "", "", "", ""},
		},
		{"TestEndpointTaskCancelled", args{tasks.CANCELED, workflow.ActivityTypeDelegate, nil, []map[string]string{
			map[string]string{"ip_address": "10.0.0.4"},
			map[string]string{"ip_address": "10.0.1.4"},
			map[string]string{"ip_address": "10.0.2.4"},
			map[string]string{"ip_address": "10.0.3.4"},
			map[string]string{"ip_address": "10.0.4.4"},
		}},
			[]string{"", "", "", "", ""},
		},
		{"TestEndpointInlineActivity", args{tasks.RUNNING, workflow.ActivityTypeInline, nil, []map[string]string{
			map[string]string{"ip_address": "10.0.0.4"},
			map[string]string{"ip_address": "10.0.1.4"},
			map[string]string{"ip_address": "10.0.2.4"},
			map[string]string{"ip_address": "10.0.3.4"},
			map[string]string{"ip_address": "10.0.4.4"},
		}},
			[]string{"", "", "", "", ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deploymentID := strings.Replace(t.Name(), "/", "_", -1)

			err := deployments.StoreDeploymentDefinition(ctx, kv, deploymentID, "testdata/compute.yaml")
			require.Nil(t, err)

			taskID := uuid.NewV4().String()

			p := &api.KVPair{Key: path.Join(consulutil.TasksPrefix, taskID, "status"), Value: []byte(strconv.Itoa(int(tt.args.taskStatus)))}
			_, err = kv.Put(p, nil)
			require.NoError(t, err)

			activity := &mockActivity{
				t: tt.args.activityType,
				v: target,
			}

			for i, eip := range tt.args.endpointIPs {
				err = deployments.SetInstanceCapabilityAttribute(deploymentID, target, fmt.Sprint(i), "endpoint", "ip_address", eip)
			}
			for i, attrs := range tt.args.attributes {
				for k, v := range attrs {
					err = deployments.SetInstanceAttribute(deploymentID, target, fmt.Sprint(i), k, v)
					require.NoError(t, err)
				}
			}
			postComputeCreationHook(ctx, cfg, taskID, deploymentID, target, activity)

			for i, check := range tt.checks {
				_, actualIP, err := deployments.GetInstanceCapabilityAttribute(kv, deploymentID, target, fmt.Sprint(i), "endpoint", "ip_address")
				require.NoError(t, err)
				assert.Equal(t, check, actualIP, "postComputeCreationHook: Unexpected value for endpoint.ip_address attribute")
			}
		})
	}
}
