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

package events

import (
	"fmt"

	"github.com/pkg/errors"
)

// StatusUpdateType is the status update type
type StatusUpdateType uint64

const (
	// InstanceStatusChangeType is the StatusUpdate type for an instance state change event
	InstanceStatusChangeType StatusUpdateType = iota
	// DeploymentStatusChangeType is the StatusUpdate type for an deployment status change event
	DeploymentStatusChangeType
	// CustomCommandStatusChangeType is the StatusUpdate type for an custom command status change event
	CustomCommandStatusChangeType
	// ScalingStatusChangeType is the StatusUpdate type for an scaling status change event
	ScalingStatusChangeType
	// WorkflowStatusChangeType is the StatusUpdate type for an workflow status change event
	WorkflowStatusChangeType
)

// StatusUpdate represents status change event
type StatusUpdate struct {
	Timestamp    string `json:"timestamp"`
	Type         string `json:"type"`
	Node         string `json:"node,omitempty"`
	Instance     string `json:"instance,omitempty"`
	TaskID       string `json:"task_id,omitempty"`
	DeploymentID string `json:"deployment_id"`
	Status       string `json:"status"`
}

const _StatusUpdateType_name = "instancedeploymentcustom-commandscalingworkflow"

var _StatusUpdateType_index = [...]uint8{0, 8, 18, 32, 39, 47}

func (i StatusUpdateType) String() string {
	if i >= StatusUpdateType(len(_StatusUpdateType_index)-1) {
		return fmt.Sprintf("StatusUpdateType(%d)", i)
	}
	return _StatusUpdateType_name[_StatusUpdateType_index[i]:_StatusUpdateType_index[i+1]]
}

var _StatusUpdateTypeNameToValue_map = map[string]StatusUpdateType{
	_StatusUpdateType_name[0:8]:   0,
	_StatusUpdateType_name[8:18]:  1,
	_StatusUpdateType_name[18:32]: 2,
	_StatusUpdateType_name[32:39]: 3,
	_StatusUpdateType_name[39:47]: 4,
}

// StatusUpdateTypeString returns a StatusUpdateType given its string representation
func StatusUpdateTypeString(s string) (StatusUpdateType, error) {
	if val, ok := _StatusUpdateTypeNameToValue_map[s]; ok {
		return val, nil
	}
	return 0, errors.Errorf("%s does not belong to StatusUpdateType values", s)
}
