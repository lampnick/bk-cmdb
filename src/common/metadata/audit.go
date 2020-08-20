/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package metadata

import (
	"encoding/json"

	"configcenter/src/common"
	"configcenter/src/common/errors"

	"go.mongodb.org/mongo-driver/bson"
)

type AuditQueryResult struct {
	BaseResp `json:",inline"`
	Data     struct {
		Count int64      `json:"count"`
		Info  []AuditLog `json:"info"`
	} `json:"data"`
}

type CreateAuditLogParam struct {
	Data []AuditLog `json:"data"`
}

type AuditQueryInput struct {
	Condition AuditQueryCondition `json:"condition"`
	Page      BasePage            `json:"page,omitempty"`
}

// Validate validates the input param
func (input *AuditQueryInput) Validate() errors.RawErrorInfo {
	if input.Page.Limit <= 0 {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrCommParamsInvalid,
			Args:    []interface{}{"limit"},
		}
	}

	if input.Page.Limit > common.BKAuditLogPageLimit {
		return errors.RawErrorInfo{
			ErrCode: common.CCErrCommPageLimitIsExceeded,
		}
	}

	return errors.RawErrorInfo{}
}

type AuditQueryCondition struct {
	AuditType    AuditType       `json:"audit_type"`
	User         string          `json:"user"`
	ResourceType ResourceType    `json:"resource_type" `
	Action       []ActionType    `json:"action"`
	OperateFrom  OperateFromType `json:"operate_from"`
	BizID        int64           `json:"bk_biz_id"`
	ResourceID   interface{}     `json:"resource_id"`
	// ResourceName filters audit logs by resource name, such as instance name, host ip etc., support fuzzy query
	ResourceName string `json:"resource_name"`
	// OperationTime is an array of start time and end time, filters audit logs between them
	OperationTime []string `json:"operation_time"`
	// Category is used by front end, filters audit logs as business(business resource and host operation related to business), resource(instance resource not related to business) or other category
	Category string `json:"category"`
}

type AuditLogBasicInfo struct {
	ID            int64        `json:"id"`
	User          string       `json:"user"`
	ResourceType  ResourceType `json:"resource_type"`
	Action        ActionType   `json:"action" bson:"action"`
	OperationTime Time         `json:"operation_time"`
	ResourceID    interface{}  `json:"resource_id"`
	ResourceName  string       `json:"resource_name"`
	BusinessID    int64        `json:"bk_biz_id"`
}

type AuditLog struct {
	ID int64 `json:"id" bson:"id"`
	// AuditType is a high level abstract of the resource managed by this cmdb.
	// Each kind of concept, resource must belongs to one of the resource type.
	AuditType AuditType `json:"audit_type" bson:"audit_type"`
	// the supplier account that this resource belongs to.
	SupplierAccount string `json:"bk_supplier_account" bson:"bk_supplier_account"`
	// name of the one who triggered this operation.
	User string `json:"user" bson:"user"`
	// the operated resource by the user
	ResourceType ResourceType `json:"resource_type" bson:"resource_type"`
	// ActionType represent the user's operation type, like CUD etc.
	Action ActionType `json:"action" bson:"action"`
	// OperateFrom describe which form does this audit come from.
	OperateFrom OperateFromType `json:"operate_from" bson:"operate_from"`
	// OperationDetail describe the details information by a user.
	// Note: when the ResourceType relevant to Business, then the business id field must
	// be bk_biz_id, otherwise the user can not search this operation log with business id.
	OperationDetail DetailFactory `json:"operation_detail" bson:"operation_detail"`
	// OperationTime is the time that user do the operation.
	OperationTime Time `json:"operation_time" bson:"operation_time"`
}

type bsonAuditLog struct {
	ID              int64           `json:"id" bson:"id"`
	AuditType       AuditType       `json:"audit_type" bson:"audit_type"`
	SupplierAccount string          `json:"bk_supplier_account" bson:"bk_supplier_account"`
	User            string          `json:"user" bson:"user"`
	ResourceType    ResourceType    `json:"resource_type" bson:"resource_type"`
	Action          ActionType      `json:"action" bson:"action"`
	OperateFrom     OperateFromType `json:"operate_from" bson:"operate_from"`
	OperationTime   Time            `json:"operation_time" bson:"operation_time"`
	OperationDetail bson.Raw        `json:"operation_detail" bson:"operation_detail"`
}

type DetailFactory interface {
	WithName() string
}

func (auditLog *AuditLog) UnmarshalJSON(data []byte) error {
	type jsonAuditLog struct {
		ID              int64           `json:"id" bson:"id"`
		AuditType       AuditType       `json:"audit_type" bson:"audit_type"`
		SupplierAccount string          `json:"bk_supplier_account" bson:"bk_supplier_account"`
		User            string          `json:"user" bson:"user"`
		ResourceType    ResourceType    `json:"resource_type" bson:"resource_type"`
		Action          ActionType      `json:"action" bson:"action"`
		OperateFrom     OperateFromType `json:"operate_from" bson:"operate_from"`
		OperationTime   Time            `json:"operation_time" bson:"operation_time"`
		OperationDetail json.RawMessage `json:"operation_detail" bson:"operation_detail"`
	}
	audit := jsonAuditLog{}
	if err := json.Unmarshal(data, &audit); err != nil {
		return err
	}
	auditLog.ID = audit.ID
	auditLog.AuditType = audit.AuditType
	auditLog.SupplierAccount = audit.SupplierAccount
	auditLog.User = audit.User
	auditLog.ResourceType = audit.ResourceType
	auditLog.Action = audit.Action
	auditLog.OperateFrom = audit.OperateFrom
	auditLog.OperationTime = audit.OperationTime
	if audit.Action == AuditTransferHostModule || audit.Action == AuditAssignHost || audit.Action == AuditUnassignHost {
		operationDetail := new(HostTransferOpDetail)
		if err := json.Unmarshal(audit.OperationDetail, &operationDetail); err != nil {
			return err
		}
		auditLog.OperationDetail = operationDetail
		return nil
	}
	switch audit.ResourceType {
	case BusinessRes, SetRes, ModuleRes, ProcessRes, HostRes, CloudAreaRes, ModelInstanceRes, MainlineInstanceRes:
		operationDetail := new(InstanceOpDetail)
		if err := json.Unmarshal(audit.OperationDetail, &operationDetail); err != nil {
			return err
		}
		auditLog.OperationDetail = operationDetail
	case InstanceAssociationRes:
		operationDetail := new(InstanceAssociationOpDetail)
		if err := json.Unmarshal(audit.OperationDetail, &operationDetail); err != nil {
			return err
		}
		auditLog.OperationDetail = operationDetail
	case ModelAssociationRes:
		operationDetail := new(ModelAssociationOpDetail)
		if err := json.Unmarshal(audit.OperationDetail, &operationDetail); err != nil {
			return err
		}
		auditLog.OperationDetail = operationDetail
	case ModelAttributeRes, ModelGroupRes:
		operationDetail := new(ModelAttrOpDetail)
		if err := json.Unmarshal(audit.OperationDetail, &operationDetail); err != nil {
			return err
		}
		auditLog.OperationDetail = operationDetail
	default:
		operationDetail := new(BasicOpDetail)
		if err := json.Unmarshal(audit.OperationDetail, &operationDetail); err != nil {
			return err
		}
		auditLog.OperationDetail = operationDetail
	}
	return nil
}

func (auditLog *AuditLog) UnmarshalBSON(data []byte) error {
	audit := bsonAuditLog{}
	if err := bson.Unmarshal(data, &audit); err != nil {
		return err
	}
	auditLog.ID = audit.ID
	auditLog.AuditType = audit.AuditType
	auditLog.SupplierAccount = audit.SupplierAccount
	auditLog.User = audit.User
	auditLog.ResourceType = audit.ResourceType
	auditLog.Action = audit.Action
	auditLog.OperateFrom = audit.OperateFrom
	auditLog.OperationTime = audit.OperationTime
	if audit.Action == AuditTransferHostModule || audit.Action == AuditAssignHost || audit.Action == AuditUnassignHost {
		operationDetail := new(HostTransferOpDetail)
		if err := bson.Unmarshal(audit.OperationDetail, &operationDetail); err != nil {
			return err
		}
		auditLog.OperationDetail = operationDetail
		return nil
	}
	switch audit.ResourceType {
	case BusinessRes, SetRes, ModuleRes, ProcessRes, HostRes, CloudAreaRes, ModelInstanceRes, MainlineInstanceRes:
		operationDetail := new(InstanceOpDetail)
		if err := bson.Unmarshal(audit.OperationDetail, &operationDetail); err != nil {
			return err
		}
		auditLog.OperationDetail = operationDetail
	case InstanceAssociationRes:
		operationDetail := new(InstanceAssociationOpDetail)
		if err := bson.Unmarshal(audit.OperationDetail, &operationDetail); err != nil {
			return err
		}
		auditLog.OperationDetail = operationDetail
	case ModelAssociationRes:
		operationDetail := new(ModelAssociationOpDetail)
		if err := bson.Unmarshal(audit.OperationDetail, &operationDetail); err != nil {
			return err
		}
		auditLog.OperationDetail = operationDetail
	case ModelAttributeRes, ModelGroupRes:
		operationDetail := new(ModelAttrOpDetail)
		if err := bson.Unmarshal(audit.OperationDetail, &operationDetail); err != nil {
			return err
		}
		auditLog.OperationDetail = operationDetail
	default:
		operationDetail := new(BasicOpDetail)
		if err := bson.Unmarshal(audit.OperationDetail, &operationDetail); err != nil {
			return err
		}
		auditLog.OperationDetail = operationDetail
	}
	return nil
}

func (auditLog AuditLog) MarshalBSON() ([]byte, error) {
	audit := bsonAuditLog{}
	audit.ID = auditLog.ID
	audit.AuditType = auditLog.AuditType
	audit.SupplierAccount = auditLog.SupplierAccount
	audit.User = auditLog.User
	audit.ResourceType = auditLog.ResourceType
	audit.Action = auditLog.Action
	audit.OperateFrom = auditLog.OperateFrom
	audit.OperationTime = auditLog.OperationTime
	var err error
	switch val := auditLog.OperationDetail.(type) {
	default:
		audit.OperationDetail, err = bson.Marshal(val)
		if err != nil {
			return []byte{}, err
		}
	}
	return bson.Marshal(audit)
}

type BasicOpDetail struct {
	// the business id of the resource if it belongs to a business.
	BusinessID int64 `json:"bk_biz_id" bson:"bk_biz_id"`
	// ResourceID is the id of the resource instance. which is a unique id, dynamic grouping id is string type.
	ResourceID interface{} `json:"resource_id" bson:"resource_id"`
	// ResourceName is the name of the resource, such as a switch model has a name "switch"
	ResourceName string `json:"resource_name" bson:"resource_name"`
	// Details contains all the details information about a user's operation
	Details *BasicContent `json:"details" bson:"details"`
}

func (op *BasicOpDetail) WithName() string {
	return "BasicDetail"
}

type ModelAttrOpDetail struct {
	BasicOpDetail `bson:",inline"`
	BkObjID       string `json:"bk_obj_id" bson:"bk_obj_id"`
	BkObjName     string `json:"bk_obj_name" bson:"bk_obj_name"`
}

func (op *ModelAttrOpDetail) WithName() string {
	return "ModelAttrDetail"
}

type InstanceOpDetail struct {
	BasicOpDetail `bson:",inline"`
	ModelID       string `json:"bk_obj_id" bson:"bk_obj_id"`
}

func (op *InstanceOpDetail) WithName() string {
	return "InstanceOpDetail"
}

type HostTransferOpDetail struct {
	// the business id of the previous biz if the host transfers to the resource pool, otherwise is the current biz
	BusinessID  int64       `json:"bk_biz_id" bson:"bk_biz_id"`
	HostID      int64       `json:"resource_id" bson:"resource_id"`
	HostInnerIP string      `json:"resource_name" bson:"resource_name"`
	PreData     HostBizTopo `json:"pre_data" bson:"pre_data"`
	CurData     HostBizTopo `json:"cur_data" bson:"cur_data"`
}

type HostBizTopo struct {
	BizID   int64  `json:"bk_biz_id" bson:"bk_biz_id"`
	BizName string `json:"bk_biz_name" bson:"bk_biz_name"`
	Set     []Topo `json:"set" bson:"set"`
}

func (op *HostTransferOpDetail) WithName() string {
	return "HostTransferOpDetail"
}

type AssociationOpDetail struct {
	AssociationID   string `json:"asst_id" bson:"asst_id"`
	AssociationKind string `json:"asst_kind" bson:"asst_kind"`
	SourceModelID   string `json:"src_model_id" bson:"src_model_id"`
	TargetModelID   string `json:"target_model_id" bson:"target_model_id"`
}

type InstanceAssociationOpDetail struct {
	AssociationOpDetail `bson:",inline"`
	SourceInstanceID    int64  `json:"src_instance_id" bson:"src_instance_id"`
	SourceInstanceName  string `json:"src_instance_name" bson:"src_instance_name"`
	TargetInstanceID    int64  `json:"target_instance_id" bson:"target_instance_id"`
	TargetInstanceName  string `json:"target_instance_name" bson:"target_instance_name"`
}

func (ao *InstanceAssociationOpDetail) WithName() string {
	return "InstanceAssociationOpDetail"
}

type ModelAssociationOpDetail struct {
	AssociationOpDetail `bson:",inline"`
	AssociationName     string                    `json:"asst_name" bson:"asst_name"`
	Mapping             AssociationMapping        `json:"mapping" bson:"mapping"`
	OnDelete            AssociationOnDeleteAction `json:"on_delete" bson:"on_delete"`
	IsPre               *bool                     `json:"is_pre" bson:"is_pre"`
	SourceModelName     string                    `json:"src_model_name" bson:"src_model_name"`
	TargetModelName     int64                     `json:"target_model_name" bson:"target_model_name"`
}

func (ao *ModelAssociationOpDetail) WithName() string {
	return "ModelAssociationOpDetail"
}

// Content contains the details information with in a user's operation.
// Generally, works for business, model, model instance etc.
type BasicContent struct {
	// PreData the previous data before the deletion or updating operation
	PreData map[string]interface{} `json:"pre_data" bson:"pre_data"`
	// CurData the current date after the creation operation
	CurData map[string]interface{} `json:"cur_data" bson:"cur_data"`
	// UpdateFields the data that user uses to update the pre data, might not be the actual changed data
	UpdateFields map[string]interface{} `json:"update_fields" bson:"update_fields"`
}

type AuditType string

const (
	// BusinessKind represent business itself's operation audit. such as you change a business maintainer, it's
	// audit belongs to this kind.
	BusinessType AuditType = "business"

	// Business resource include resources as follows:
	// - service template
	// - set template
	// - service category
	// - dynamic group
	// - main line instance, such as user-defined topology level, set, module etc.
	// - service instance.
	// - others.
	//
	// Note: host does not belong to business resource, it's a independent resource kind.
	BusinessResourceType AuditType = "business_resource"

	// HostType represent all the host related resource's operation audit.
	HostType AuditType = "host"

	// ModelType represent all the operation audit related with model in the system
	ModelType AuditType = "model"

	// ModelInstanceType represent all the operation audit related with model instance in the system,
	// and the instance association is included.
	ModelInstanceType AuditType = "model_instance"

	// AssociationKindType represent all the association kind operation audit.
	AssociationKindType AuditType = "association_kind"

	// EventType represent all the event related operation audit.
	EventPushType AuditType = "event"

	// CloudResource represent all the operation audit related with cloud, such as:
	// - cloud area
	// - cloud account
	// - cloud synchronize job
	// - others
	CloudResourceType AuditType = "cloud_resource"
)

type ResourceType string

const (
	// business related operation type
	BusinessRes             ResourceType = "business"
	ServiceTemplateRes      ResourceType = "service_template"
	SetTemplateRes          ResourceType = "set_template"
	ServiceCategoryRes      ResourceType = "service_category"
	DynamicGroupRes         ResourceType = "dynamic_group"
	ServiceInstanceRes      ResourceType = "service_instance"
	ServiceInstanceLabelRes ResourceType = "service_instance_label"
	SetRes                  ResourceType = "set"
	ModuleRes               ResourceType = "module"
	ProcessRes              ResourceType = "process"
	HostApplyRes            ResourceType = "host_apply"
	CustomFieldRes          ResourceType = "custom_field"

	// model related operation type
	ModelRes               ResourceType = "model"
	ModelInstanceRes       ResourceType = "model_instance"
	MainlineInstanceRes    ResourceType = "mainline_instance"
	ModelAssociationRes    ResourceType = "model_association"
	ModelAttributeRes      ResourceType = "model_attribute"
	ModelAttributeGroupRes ResourceType = "model_attribute_group"
	ModelClassificationRes ResourceType = "model_classification"
	InstanceAssociationRes ResourceType = "instance_association"
	ModelGroupRes          ResourceType = "model_group"
	ModelUniqueRes         ResourceType = "model_unique"
	ResourceDirectoryRes   ResourceType = "resource_directory"

	AssociationKindRes ResourceType = "association_kind"
	EventPushRes       ResourceType = "event"
	CloudAreaRes       ResourceType = "cloud_area"
	CloudAccountRes    ResourceType = "cloud_account"
	CloudSyncTaskRes   ResourceType = "cloud_sync_task"

	// host related operation type
	HostRes ResourceType = "host"
)

type OperateFromType string

const (
	// FromCCSystem means this audit come from cc system operation, such as upgrader.
	FromCCSystem OperateFromType = "cc_system"
	// FromUser means this audit come from a user's operation, such as web.
	FromUser OperateFromType = "user"
	// FromDataCollection means this audit is created by data collection.
	FromDataCollection OperateFromType = "data_collection"
	// FromSynchronizer means this audit is created by the data synchronizer.
	FromSynchronizer OperateFromType = "synchronizer"
)

// ActionType defines all the user's operation type
type ActionType string

const (
	// create a resource
	AuditCreate ActionType = "create"
	// update a resource
	AuditUpdate ActionType = "update"
	// delete a resource
	AuditDelete ActionType = "delete"
	// transfer a host from resource pool to biz
	AuditAssignHost ActionType = "assign_host"
	// transfer a host from biz to resource pool
	AuditUnassignHost ActionType = "unassign_host"
	// transfer host to another module
	AuditTransferHostModule ActionType = "transfer_host_module"
	// archive a resource
	AuditArchive ActionType = "archive"
	// recover a resource
	AuditRecover ActionType = "recover"
	// pause an object
	AuditPause ActionType = "stop"
	// resume using an object
	AuditResume ActionType = "resume"
)

func GetAuditTypeByObjID(objID string, isMainline bool) AuditType {
	switch objID {
	case common.BKInnerObjIDApp:
		return BusinessType
	case common.BKInnerObjIDSet:
		return BusinessResourceType
	case common.BKInnerObjIDModule:
		return BusinessResourceType
	case common.BKInnerObjIDObject:
		return ModelInstanceType
	case common.BKInnerObjIDHost:
		return HostType
	case common.BKInnerObjIDProc:
		return BusinessResourceType
	case common.BKInnerObjIDPlat:
		return CloudResourceType
	default:
		if isMainline {
			return BusinessResourceType
		}
		return ModelInstanceType
	}
}

func GetResourceTypeByObjID(objID string, isMainline bool) ResourceType {
	switch objID {
	case common.BKInnerObjIDApp:
		return BusinessRes
	case common.BKInnerObjIDSet:
		return SetRes
	case common.BKInnerObjIDModule:
		return ModuleRes
	case common.BKInnerObjIDObject:
		return ModelInstanceRes
	case common.BKInnerObjIDHost:
		return HostRes
	case common.BKInnerObjIDProc:
		return ProcessRes
	case common.BKInnerObjIDPlat:
		return CloudAreaRes
	default:
		if isMainline {
			return MainlineInstanceRes
		}
		return ModelInstanceRes
	}
}

func GetAuditTypesByCategory(category string) []AuditType {
	switch category {
	case "business":
		return []AuditType{BusinessResourceType, HostType}
	case "resource":
		return []AuditType{BusinessType, ModelInstanceType, CloudResourceType}
	case "other":
		return []AuditType{ModelType, AssociationKindType, EventPushType}
	}
	return []AuditType{}
}

func GetAuditDict() []resourceTypeInfo {
	return auditDict
}

var auditDict = []resourceTypeInfo{
	{
		ID:   DynamicGroupRes,
		Name: "动态分组",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   SetTemplateRes,
		Name: "集群模板",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   ServiceTemplateRes,
		Name: "服务模板",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   ServiceCategoryRes,
		Name: "服务分类",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   ModuleRes,
		Name: "模块",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   SetRes,
		Name: "集群",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   MainlineInstanceRes,
		Name: "节点",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   ServiceInstanceRes,
		Name: "服务实例",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   ProcessRes,
		Name: "进程",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   ServiceInstanceLabelRes,
		Name: "服务实例标签",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditUpdate],
		},
	},
	{
		ID:   HostRes,
		Name: "主机",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
			actionInfoMap[AuditAssignHost],
			actionInfoMap[AuditUnassignHost],
			actionInfoMap[AuditTransferHostModule],
		},
	},
	{
		ID:   HostApplyRes,
		Name: "主机自动应用",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   CustomFieldRes,
		Name: "自定义字段",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   BusinessRes,
		Name: "业务",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditArchive],
			actionInfoMap[AuditRecover],
		},
	},
	{
		ID:   CloudAreaRes,
		Name: "云区域",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   ModelInstanceRes,
		Name: "模型实例",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   InstanceAssociationRes,
		Name: "实例关联",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   ResourceDirectoryRes,
		Name: "资源池目录",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   ModelGroupRes,
		Name: "模型分组",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
			actionInfoMap[AuditPause],
			actionInfoMap[AuditResume],
		},
	},
	{
		ID:   ModelRes,
		Name: "模型",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   ModelAttributeRes,
		Name: "模型字段",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   ModelUniqueRes,
		Name: "模型唯一校验",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   ModelAssociationRes,
		Name: "模型关联",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   ModelAttributeGroupRes,
		Name: "模型字段分组",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   EventPushRes,
		Name: "事件订阅",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   AssociationKindRes,
		Name: "关联类型",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   CloudAccountRes,
		Name: "云账户",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
	{
		ID:   CloudSyncTaskRes,
		Name: "云资源同步任务",
		Operations: []actionTypeInfo{
			actionInfoMap[AuditCreate],
			actionInfoMap[AuditUpdate],
			actionInfoMap[AuditDelete],
		},
	},
}

var actionInfoMap = map[ActionType]actionTypeInfo{
	AuditCreate:             {ID: AuditCreate, Name: "新增"},
	AuditUpdate:             {ID: AuditUpdate, Name: "修改"},
	AuditDelete:             {ID: AuditDelete, Name: "删除"},
	AuditAssignHost:         {ID: AuditAssignHost, Name: "分配到业务"},
	AuditUnassignHost:       {ID: AuditUnassignHost, Name: "归还到资源池"},
	AuditTransferHostModule: {ID: AuditTransferHostModule, Name: "转移模块"},
	AuditArchive:            {ID: AuditArchive, Name: "归档"},
	AuditRecover:            {ID: AuditRecover, Name: "恢复"},
	AuditPause:              {ID: AuditPause, Name: "停用"},
	AuditResume:             {ID: AuditResume, Name: "启用"},
}

type resourceTypeInfo struct {
	ID         ResourceType     `json:"id"`
	Name       string           `json:"name"`
	Operations []actionTypeInfo `json:"operations"`
}

type actionTypeInfo struct {
	ID   ActionType `json:"id"`
	Name string     `json:"name"`
}
