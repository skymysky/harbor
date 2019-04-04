// Copyright Project Harbor Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package manager

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/goharbor/harbor/src/replication/ng/dao"
	persist_models "github.com/goharbor/harbor/src/replication/ng/dao/models"
	"github.com/goharbor/harbor/src/replication/ng/model"
	"github.com/goharbor/harbor/src/replication/ng/policy"
)

var errNilPolicyModel = errors.New("nil policy model")

func convertFromPersistModel(policy *persist_models.RepPolicy) (*model.Policy, error) {
	if policy == nil {
		return nil, nil
	}

	ply := model.Policy{
		ID:            policy.ID,
		Name:          policy.Name,
		Description:   policy.Description,
		Creator:       policy.Creator,
		DestNamespace: policy.DestNamespace,
		Deletion:      policy.ReplicateDeletion,
		Override:      policy.Override,
		Enabled:       policy.Enabled,
		CreationTime:  policy.CreationTime,
		UpdateTime:    policy.UpdateTime,
	}
	if policy.SrcRegistryID > 0 {
		ply.SrcRegistry = &model.Registry{
			ID: policy.SrcRegistryID,
		}
	}
	if policy.DestRegistryID > 0 {
		ply.DestRegistry = &model.Registry{
			ID: policy.DestRegistryID,
		}
	}

	// 1. parse SrcNamespaces to array
	if len(policy.SrcNamespaces) > 0 {
		ply.SrcNamespaces = strings.Split(policy.SrcNamespaces, ",")
	}

	// 2. parse Filters
	if len(policy.Filters) > 0 {
		filters := []*model.Filter{}
		if err := json.Unmarshal([]byte(policy.Filters), &filters); err != nil {
			return nil, err
		}
		// convert the type of value from string to model.ResourceType if the filter
		// is a resource type filter
		for _, filter := range filters {
			if filter.Type == model.FilterTypeResource {
				filter.Value = (model.ResourceType)(filter.Value.(string))
			}
		}
		ply.Filters = filters
	}

	// 3. parse Trigger
	if len(policy.Trigger) > 0 {
		trigger := &model.Trigger{}
		if err := json.Unmarshal([]byte(policy.Trigger), trigger); err != nil {
			return nil, err
		}
		ply.Trigger = trigger
	}

	return &ply, nil
}

func convertToPersistModel(policy *model.Policy) (*persist_models.RepPolicy, error) {
	if policy == nil {
		return nil, errNilPolicyModel
	}

	ply := &persist_models.RepPolicy{
		ID:                policy.ID,
		Name:              policy.Name,
		Description:       policy.Description,
		Creator:           policy.Creator,
		SrcNamespaces:     strings.Join(policy.SrcNamespaces, ","),
		DestNamespace:     policy.DestNamespace,
		Override:          policy.Override,
		Enabled:           policy.Enabled,
		ReplicateDeletion: policy.Deletion,
		CreationTime:      policy.CreationTime,
		UpdateTime:        time.Now(),
	}
	if policy.SrcRegistry != nil {
		ply.SrcRegistryID = policy.SrcRegistry.ID
	}
	if policy.DestRegistry != nil {
		ply.DestRegistryID = policy.DestRegistry.ID
	}

	if policy.Trigger != nil {
		trigger, err := json.Marshal(policy.Trigger)
		if err != nil {
			return nil, err
		}
		ply.Trigger = string(trigger)
	}

	if len(policy.Filters) > 0 {
		filters, err := json.Marshal(policy.Filters)
		if err != nil {
			return nil, err
		}
		ply.Filters = string(filters)
	}

	return ply, nil
}

// DefaultManager provides replication policy CURD capabilities.
type DefaultManager struct{}

var _ policy.Controller = &DefaultManager{}

// NewDefaultManager is the constructor of DefaultManager.
func NewDefaultManager() *DefaultManager {
	return &DefaultManager{}
}

// Create creates a new policy with the provided data;
// If creating failed, error will be returned;
// If creating succeed, ID of the new created policy will be returned.
func (m *DefaultManager) Create(policy *model.Policy) (int64, error) {
	ply, err := convertToPersistModel(policy)
	if err != nil {
		return 0, err
	}
	return dao.AddRepPolicy(ply)
}

// List returns all the policies
func (m *DefaultManager) List(queries ...*model.PolicyQuery) (total int64, policies []*model.Policy, err error) {
	var persistPolicies []*persist_models.RepPolicy
	total, persistPolicies, err = dao.GetPolicies(queries...)
	if err != nil {
		return
	}

	for _, policy := range persistPolicies {
		ply, err := convertFromPersistModel(policy)
		if err != nil {
			return 0, nil, err
		}

		policies = append(policies, ply)
	}

	if policies == nil {
		policies = []*model.Policy{}
	}

	return
}

// Get returns the policy with the specified ID
func (m *DefaultManager) Get(policyID int64) (*model.Policy, error) {
	policy, err := dao.GetRepPolicy(policyID)
	if err != nil {
		return nil, err
	}

	return convertFromPersistModel(policy)
}

// GetByName returns the policy with the specified name
func (m *DefaultManager) GetByName(name string) (*model.Policy, error) {
	policy, err := dao.GetRepPolicyByName(name)
	if err != nil {
		return nil, err
	}

	return convertFromPersistModel(policy)
}

// Update Update the specified policy
func (m *DefaultManager) Update(policy *model.Policy, props ...string) error {
	updatePolicy, err := convertToPersistModel(policy)
	if err != nil {
		return err
	}

	return dao.UpdateRepPolicy(updatePolicy, props...)
}

// Remove Remove the specified policy
func (m *DefaultManager) Remove(policyID int64) error {
	return dao.DeleteRepPolicy(policyID)
}
