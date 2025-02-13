package impl

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/kade-chen/STSO/optimization/app"
	"github.com/kade-chen/library/exception"
	"github.com/kade-chen/library/tools/format"
	"github.com/kade-chen/library/tools/generics"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

var receiveCounter int32

func (s *service) Receive(ctx context.Context) error {
	return s.Sub.Receive(ctx, func(_ context.Context, msg *pubsub.Message) {

		atomic.AddInt32(&receiveCounter, 1)

		var logData app.AuditLog
		err := format.Unmarshal([]byte(msg.Data), &logData)
		if err != nil {
			s.log.Error().Msgf("unmarshal error: %v", err)
			msg.Nack()
			s.log.Info().Msg("nack message")
			return
		}

		if logData.ProtoPayload.AuthenticationInfo.PrincipalEmail == "kade.chen522@gmail.com" {
			s.log.Info().Msgf("ignore kade.chen522@gmail.com")
			msg.Ack()
			s.log.Info().Msg("ack message")
			return
		}
		// fmt.Println(format.ToJSON(logData))
		err = s.CreateSpotInstance(ctx, logData.Resource.Labels.ProjectId, logData.Resource.Labels.Zone, strings.Split(logData.ProtoPayload.ResourceName, "/")[(len(strings.Split(logData.ProtoPayload.ResourceName, "/"))-1)], msg)
		if err != nil {
			msg.Nack()
			s.log.Info().Msg("nack message")
			return
		}
		s.log.Info().Msgf("receiveCounter: %d", receiveCounter)
	})
}

func (s *service) CreateSpotInstance(ctx context.Context, projectID string, zone string, instanceName string, msg *pubsub.Message) error {
	zone = "northamerica-south1-b"
	instance := &compute.Instance{
		Name:        instanceName,
		MachineType: fmt.Sprintf("zones/%s/machineTypes/f1-micro", zone),
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				Subnetwork: fmt.Sprintf("projects/%s/regions/northamerica-south1/subnetworks/cc", projectID),
				AccessConfigs: []*compute.AccessConfig{{
					NetworkTier: "PREMIUM",
				}},
			},
			// {
			// 	Network:    "projects/kade-poc/global/networks/kade-vpc",
			// 	Subnetwork: fmt.Sprintf("projects/%s/regions/us-central1/subnetworks/gke", projectID),
			// 	AccessConfigs: []*compute.AccessConfig{{
			// 		NetworkTier: "PREMIUM",
			// 	}},
			// },
			// {
			// 	Network:    "projects/kade-poc/global/networks/gcp-vpc",
			// 	Subnetwork: fmt.Sprintf("projects/%s/regions/us-central1/subnetworks/kade-test01", projectID),
			// 	AccessConfigs: []*compute.AccessConfig{{
			// 		NetworkTier: "PREMIUM",
			// 	}},
			// },
		},
		Disks: []*compute.AttachedDisk{
			{
				Boot:       true,
				AutoDelete: true,
				InitializeParams: &compute.AttachedDiskInitializeParams{
					// SourceImage: "projects/debian-cloud/global/images/debian-11-bullseye-v20250123",
					SourceImage: "projects/kade-poc/global/images/kade-test",
					DiskSizeGb:  50,
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
		if apiErr, ok := err.(*googleapi.Error); ok {
			fmt.Printf("API Error Code: %d\n", apiErr.Code)
			fmt.Printf("API Error Message: %s\n", apiErr.Message)
			// 检查错误代码和消息
			if apiErr.Code == 400 || apiErr.Code == 403 {
				if strings.Contains(apiErr.Message, "ZONE_RESOURCE_POOL_EXHAUSTED") {
					fmt.Println("Spot instances unavailable in this zone.")
				}
			}
		}
		s.log.Error().Msgf("Failed to create instance: %v", err)
		return exception.NewInternalServerError("Failed to create instance, ERROR: %v", err)
	}

	c, err := s.checkInstanceStatus(ctx, s.Gce, projectID, zone, instanceName, msg)
	if err != nil {
		s.log.Error().Msgf("Failed to create instance: %v", err)
		return err
	}

	fmt.Println(format.ToJSON(c))
	// fmt.Printf("%+v", format.ToJSON(operation))
	fmt.Printf("Instance creation started: %v\n", strings.Split(operation.TargetLink, "/")[(len(strings.Split(operation.TargetLink, "/"))-1)])
	return nil
}

func (s *service) checkInstanceStatus(ctx context.Context, computeService *compute.Service, projectID, zone, instanceName string, msg *pubsub.Message) (cc *compute.Instance, err error) {
	for {
		// 获取实例信息
		cc, err = computeService.Instances.Get(projectID, zone, instanceName).Context(ctx).Do()
		if err != nil {
			s.log.Error().Msgf("Failed to get instance status: %v", err)
			return nil, exception.NewInternalServerError("Failed to get instance status, ERROR: %v", err) // 直接返回错误，而不是 `log.Fatalf`
		}

		// 处理不同状态
		switch cc.Status {
		case "RUNNING":
			s.log.Info().Msgf("Instance is running: %v\n", cc.Name)
			msg.Ack()
			return cc, nil // 返回实例信息
		case "STOPPING":
			time.Sleep(8 * time.Second)
			return nil, exception.NewInternalServerError("Instance creation failed: %v\n", cc.Status)
		default:
			// s.log.Debug().Msgf("Instance not ready yet, status: %v\n", cc.Status)
			continue
		}
	}
}
