package app

import (
	"context"
)

const (
	AppName = "stso"
)

type Service interface {
	Receive(context.Context) error
}

type AuditLog struct {
	ProtoPayload struct {
		Type               string `json:"@type"`
		AuthenticationInfo struct {
			PrincipalEmail string `json:"principalEmail"`
		} `json:"authenticationInfo"`
		RequestMetadata struct {
			CallerIp                string `json:"callerIp"`
			CallerSuppliedUserAgent string `json:"callerSuppliedUserAgent"`
		} `json:"requestMetadata"`
		ServiceName  string `json:"serviceName"`
		MethodName   string `json:"methodName"`
		ResourceName string `json:"resourceName"`
		Request      struct {
			Type string `json:"@type"`
		} `json:"request"`
	} `json:"protoPayload"`
	InsertId string `json:"insertId"`
	Resource struct {
		Type   string `json:"type"`
		Labels struct {
			ProjectId  string `json:"project_id"`
			InstanceId string `json:"instance_id"`
			Zone       string `json:"zone"`
		} `json:"labels"`
	} `json:"resource"`
	Timestamp string `json:"timestamp"`
	Severity  string `json:"severity"`
	Labels    struct {
		RootTriggerId string `json:"compute.googleapis.com/root_trigger_id"`
	} `json:"labels"`
	LogName   string `json:"logName"`
	Operation struct {
		Id       string `json:"id"`
		Producer string `json:"producer"`
		Last     bool   `json:"last"`
	} `json:"operation"`
	ReceiveTimestamp string `json:"receiveTimestamp"`
}
