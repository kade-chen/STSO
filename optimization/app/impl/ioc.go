package impl

import (
	"context"

	"cloud.google.com/go/pubsub"
	"github.com/kade-chen/STSO/optimization/app"
	"github.com/kade-chen/library/exception"
	"github.com/kade-chen/library/ioc"
	logs "github.com/kade-chen/library/ioc/config/log"
	"github.com/rs/zerolog"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

var _ app.Service = (*service)(nil)

func init() {
	ioc.Controller().Registry(&service{})
}

type service struct {
	ioc.ObjectImpl
	Pubsub    *pubsub.Client
	Sub       *pubsub.Subscription
	Gce       *compute.Service
	log       *zerolog.Logger
	ProjectID string `toml:"project_id" json:"project_id" yaml:"project_id"  env:"project_id"`
	// service account directory path name
	Env                bool   `toml:"env" json:"env" yaml:"env" env:"env"`
	ServiceAccountDev  string `toml:"service_account_dev" json:"service_account_dev" yaml:"service_account_dev" env:"service_account_dev"`
	ServiceAccountProd string `toml:"service_account_prod" json:"service_account_prod" yaml:"service_account_prod" env:"service_account_prod"`
	SubscriptionId     string `toml:"subscription_id" json:"subscription_id" yaml:"subscription_id" env:"subscription_id"`
}

func (s *service) Init() error {
	s.log = logs.Sub(app.AppName)

	credFile := s.ServiceAccountDev
	if s.Env {
		credFile = s.ServiceAccountProd
	}

	client, err := pubsub.NewClient(context.Background(), s.ProjectID, option.WithCredentialsFile(credFile))
	if err != nil {
		return exception.NewIocRegisterFailed("pubsub.NewClient error: %v\n", err)
	}
	s.Pubsub = client

	s.Sub = s.Pubsub.Subscription(s.SubscriptionId)

	computeClient, err := compute.NewService(context.Background(), option.WithCredentialsFile(credFile))
	if err != nil {
		return exception.NewIocRegisterFailed("compute.NewService error: %v\n", err)
	}
	s.Gce = computeClient

	return nil
}
func (s *service) Name() string {
	return app.AppName
}
