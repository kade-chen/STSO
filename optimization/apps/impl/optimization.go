package impl

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"cloud.google.com/go/pubsub"
	"github.com/kade-chen/library/exception"
	"github.com/kade-chen/library/tools/format"
	"github.com/kade-chen/library/tools/generics"
	"google.golang.org/api/compute/v1"
)

var receiveCounter int32

func (s *service) Receive(ctx context.Context) error {
	return s.Sub.Receive(ctx, func(_ context.Context, msg *pubsub.Message) {

		atomic.AddInt32(&receiveCounter, 1)

		var logData AuditLog
		err := format.Unmarshal([]byte(msg.Data), &logData)
		if err != nil {
			fmt.Println("unmarshal error:", err) // 记录日
			msg.Nack()
		}

		if logData.ProtoPayload.AuthenticationInfo.PrincipalEmail == "kade.chen522@gmail.com" {
			fmt.Println("ignore kade.chen522@gmail.com")
			msg.Ack()
			return
		}
		// fmt.Println(format.ToJSON(logData))
		s.CreateSpotInstance(ctx, logData.Resource.Labels.ProjectId, logData.Resource.Labels.Zone, strings.Split(logData.ProtoPayload.ResourceName, "/")[(len(strings.Split(logData.ProtoPayload.ResourceName, "/"))-1)])
		// fmt.Println("a:", logData)
		// msg.Ack()
		msg.Ack()
		fmt.Println("-0---------111-", receiveCounter)
	})
}

func (s *service) CreateSpotInstance(ctx context.Context, projectID string, zone string, instanceName string) error {

	instance := &compute.Instance{
		Name:        instanceName,
		MachineType: fmt.Sprintf("zones/%s/machineTypes/f1-micro", zone),
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				Subnetwork: fmt.Sprintf("projects/%s/regions/us-central1/subnetworks/gke", projectID),
				AccessConfigs: []*compute.AccessConfig{{
					NetworkTier: "PREMIUM",
				}},
			},
		},
		Disks: []*compute.AttachedDisk{
			{
				Boot:       true,
				AutoDelete: true,
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: "projects/debian-cloud/global/images/debian-11-bullseye-v20250123",
					DiskSizeGb:  10,
					DiskType:    fmt.Sprintf("zones/%s/diskTypes/pd-balanced", zone),
				},
			},
		},
		Scheduling: &compute.Scheduling{
			ProvisioningModel:         "SPOT",
			InstanceTerminationAction: "STOP",
			OnHostMaintenance:         "TERMINATE",
			AutomaticRestart:          generics.Generics[bool](false),
		},
		ServiceAccounts: []*compute.ServiceAccount{
			{
				Email: "499036589398-compute@developer.gserviceaccount.com",
				Scopes: []string{
					"https://www.googleapis.com/auth/devstorage.read_only",
					"https://www.googleapis.com/auth/logging.write",
					"https://www.googleapis.com/auth/monitoring.write",
					"https://www.googleapis.com/auth/service.management.readonly",
					"https://www.googleapis.com/auth/servicecontrol",
					"https://www.googleapis.com/auth/trace.append",
				},
			},
		},
		Labels: map[string]string{
			"goog-ec-src": "vm_add-gcloud",
		},
		ShieldedInstanceConfig: &compute.ShieldedInstanceConfig{
			EnableVtpm:                true,
			EnableIntegrityMonitoring: true,
		},
		ReservationAffinity: &compute.ReservationAffinity{
			ConsumeReservationType: "ANY_RESERVATION",
		},
	}

	// 发送创建实例请求
	operation, err := s.Gce.Instances.Insert(projectID, zone, instance).Context(ctx).Do()
	if err != nil {
		return exception.NewInternalServerError("Failed to create instance, ERROR: %v", err)
	}
	// fmt.Printf("%+v", format.ToJSON(operation))
	fmt.Printf("Instance creation started: %v\n", strings.Split(operation.TargetLink, "/")[(len(strings.Split(operation.TargetLink, "/"))-1)])
	return nil
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
