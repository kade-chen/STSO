package impl

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync/atomic"

	"cloud.google.com/go/pubsub"
	"github.com/kade-chen/STSO/optimization/app"
	"github.com/kade-chen/library/tools/format"
)

var receiveCounter int32

func (s *service) Receive(ctx context.Context) error {
	return s.Sub.Receive(ctx, func(_ context.Context, msg *pubsub.Message) {
		atomic.AddInt32(&receiveCounter, 1)
		s.log.Info().Msgf("------------------Start Processing, The message is received for the %d time-----------------------------", receiveCounter)
		var logData app.AuditLog
		if err := format.Unmarshal([]byte(msg.Data), &logData); err != nil {
			s.log.Error().Msgf("Unmarshal error: %v", err)
			msg.Nack()
			s.log.Info().Msgf("---------------Handling Exception, Parsing Failure, Nack message--------------------------------")
			return
		}

		if logData.ProtoPayload.AuthenticationInfo.PrincipalEmail == "kade.chen522@gmail.com" {
			s.log.Info().Msg("Ignoring message from kade.chen522@gmail.com")
			msg.Ack()
			s.log.Info().Msgf("---------------Processing Succeeded, Super Privilege, Ack message--------------------------------")
			return
		}

		instanceName := strings.Split(logData.ProtoPayload.ResourceName, "/")[len(strings.Split(logData.ProtoPayload.ResourceName, "/"))-1]
		//1.默认创建spot
		operation, err := s.createSpotInstance(ctx, logData.Resource.Labels.ProjectId, logData.Resource.Labels.Zone, instanceName, msg)
		if err != nil {
			//2.重试机制，如果创建失败，则重试
			operation, err = s.retryCreateInstance(ctx, logData.Resource.Labels.ProjectId, logData.Resource.Labels.Zone, instanceName, msg)
			if err != nil {
				msg.Nack()
				s.log.Info().Msg("nack message")
				s.log.Info().Msgf("---------------Handling Exception, Retry Failure, Nack message--------------------------------")
				return
			}
		}
		//3.检查是否创建成功
		instance, err := s.checkInstanceStatus(ctx, logData.Resource.Labels.ProjectId, path.Base(operation.Zone), instanceName, msg)
		if err != nil {
			msg.Nack()
			s.log.Info().Msg("nack message")
			s.log.Info().Msgf("---------------Handling Exception, Checking Instance Status failed, Nack message--------------------------------")
			return
		}

		fmt.Println(format.ToJSON(instance))
		fmt.Printf("Instance creation started: %v\n", strings.Split(operation.TargetLink, "/")[len(strings.Split(operation.TargetLink, "/"))-1])
		s.log.Info().Msgf("Receive counter: %d", receiveCounter)
		s.log.Info().Msgf("---------------The Process is Successful, The message was received for the %d time--------------------------------", receiveCounter)
	})
}
