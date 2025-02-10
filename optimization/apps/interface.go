package apps

import (
	"context"
)

const (
	AppName = "stso"
)

type Service interface {
	Receive(context.Context) error
	//first string is the projectID , second is the zone, third is the instanceName
	CreateSpotInstance(context.Context, string, string, string) error
}
