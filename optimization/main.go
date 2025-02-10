package main

import (
	"context"

	"github.com/kade-chen/STSO/optimization/apps"
	_ "github.com/kade-chen/STSO/optimization/apps/impl"
	"github.com/kade-chen/library/ioc"
)

var (
	ctx  = context.Background()
	impl apps.Service
)

func main() {
	impl.Receive(ctx)
}

func init() {
	req := ioc.NewLoadConfigRequest()
	req.ConfigFile.Enabled = true
	req.ConfigFile.Path = "etc/config.toml"
	ioc.DevelopmentSetup(req)
	impl = ioc.Controller().Get(apps.AppName).(apps.Service)
}
