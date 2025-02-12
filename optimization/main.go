package main

import (
	"context"

	"github.com/kade-chen/STSO/optimization/app"
	_ "github.com/kade-chen/STSO/optimization/app/impl"
	"github.com/kade-chen/library/ioc"
)

var (
	ctx  = context.Background()
	impl app.Service
)

func main() {
	impl.Receive(ctx)
}

func init() {
	req := ioc.NewLoadConfigRequest()
	req.ConfigFile.Enabled = true
	req.ConfigFile.Path = "etc/config.toml"
	ioc.DevelopmentSetup(req)
	impl = ioc.Controller().Get(app.AppName).(app.Service)
}
