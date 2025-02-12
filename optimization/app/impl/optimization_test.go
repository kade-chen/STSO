package impl_test

import (
	"context"
	"testing"

	"github.com/kade-chen/STSO/optimization/app"
	_ "github.com/kade-chen/STSO/optimization/app/impl"
	"github.com/kade-chen/library/ioc"
)

var (
	ctx  = context.Background()
	impl app.Service
)

func TestMain(t *testing.T) {
	impl.Receive(ctx)
}

func init() {
	req := ioc.NewLoadConfigRequest()
	req.ConfigFile.Enabled = true
	req.ConfigFile.Path = "/Users/kade.chen/go-kade-project/github/STSO/etc/config.toml"
	ioc.DevelopmentSetup(req)
	impl = ioc.Controller().Get(app.AppName).(app.Service)
}
