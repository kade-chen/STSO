package impl

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/kade-chen/library/exception"
	"github.com/kade-chen/library/tools/generics"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

// first string is the projectID , second is the zone, third is the instanceName, fourth is the pubsub.Message
func (s *service) createSpotInstance(ctx context.Context, projectID, zone, instanceName string, msg *pubsub.Message) (*compute.Operation, error) {
	// zone = "northamerica-south1-b"
	return s.insertInstance(ctx, projectID, zone, instanceName)
}

func (s *service) insertInstance(ctx context.Context, projectID, zone, instanceName string) (*compute.Operation, error) {
	operation, err := s.Gce.Instances.Insert(projectID, zone, s.instance(projectID, zone, instanceName)).Context(ctx).Do()
	if err != nil {
		if apiErr, ok := err.(*googleapi.Error); ok {
			s.log.Error().Msgf("Insert instance failed in zone %s, API Error Code: %d, Message: %s\n", zone, apiErr.Code, apiErr.Message)
		}
		return nil, err
	}
	return operation, nil
}

func (s *service) retryCreateInstance(ctx context.Context, projectID, zone, instanceName string, msg *pubsub.Message) (*compute.Operation, error) {
	// zone = "northamerica-south1-b"
	zones, err := s.getAvabileZone(zone)
	if err != nil {
		s.log.Error().Msgf("Failed to get available zones: %v", err)
		return nil, err
	}

	for _, v := range zones {

		if v == zone {
			continue
		}

		s.log.Info().Msgf("Trying zone: %s", v)
		operation, err := s.insertInstance(ctx, projectID, v, instanceName)
		if err == nil {
			s.log.Info().Msgf("Successfully created instance in zone %s", v)
			return operation, nil
		}

		continue
	}
	return nil, exception.NewInternalServerError("Failed to create instance in all available zones")
}

func (s *service) checkInstanceStatus(ctx context.Context, projectID, zone, instanceName string, msg *pubsub.Message) (*compute.Instance, error) {
	for {
		instance, err := s.Gce.Instances.Get(projectID, zone, instanceName).Context(ctx).Do()
		if err != nil {
			s.log.Error().Msgf("Failed to get instance status: %v", err)
			return nil, exception.NewInternalServerError("Failed to get instance status, ERROR: %v", err)
		}

		switch instance.Status {
		case "RUNNING":
			s.log.Info().Msgf("Instance is running: %v", instance.Name)
			msg.Ack()
			return instance, nil
		case "STOPPING":
			s.log.Warn().Msg("Instance creation failed: STOPPING state detected")
			time.Sleep(8 * time.Second)
			return nil, exception.NewInternalServerError("Instance creation failed: STOPPING state")
		default:
			s.log.Info().Msgf("Instance not ready yet, status: %v", instance.Status)
			time.Sleep(3 * time.Second) // 避免 API 轮询过于频繁
			continue
		}
	}
}

func (s *service) instance(projectID, zone, instanceName string) *compute.Instance {
	return &compute.Instance{
		Name:        instanceName,
		MachineType: fmt.Sprintf("zones/%s/machineTypes/f1-micro", zone),
		NetworkInterfaces: []*compute.NetworkInterface{
			// {
			// 	Subnetwork: fmt.Sprintf("projects/%s/regions/%s/subnetworks/cc", projectID, strings.Join(strings.Split(zone, "-")[:len(strings.Split(zone, "-"))-1], "-")),
			// 	AccessConfigs: []*compute.AccessConfig{{
			// 		NetworkTier: "PREMIUM",
			// 	}},
			// },
			{
				Network:    "projects/kade-poc/global/networks/kade-vpc",
				Subnetwork: fmt.Sprintf("projects/%s/regions/us-central1/subnetworks/gke", projectID),
				AccessConfigs: []*compute.AccessConfig{{
					NetworkTier: "PREMIUM",
				}},
			},
			{
				Network:    "projects/kade-poc/global/networks/gcp-vpc",
				Subnetwork: fmt.Sprintf("projects/%s/regions/us-central1/subnetworks/kade-test01", projectID),
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
			AutomaticRestart:          generics.Generics(false),
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
}
