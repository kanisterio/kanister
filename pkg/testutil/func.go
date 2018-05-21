package testutil

import (
	"context"

	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	FailFuncName = "FailFunc"
	WaitFuncName = "WaitFunc"
	ArgFuncName  = "ArgFunc"
)

var (
	waitFuncCh chan struct{}
	argFuncCh  chan map[string]interface{}
)

func failFunc(context.Context, param.TemplateParams, map[string]interface{}) error {
	return errors.New("Kanister Function Failed")
}

func waitFunc(context.Context, param.TemplateParams, map[string]interface{}) error {
	<-waitFuncCh
	return nil
}
func argsFunc(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) error {
	argFuncCh <- args
	return nil
}

func init() {
	waitFuncCh = make(chan struct{})
	argFuncCh = make(chan map[string]interface{})
	registerMockKanisterFunc(FailFuncName, failFunc)
	registerMockKanisterFunc(WaitFuncName, waitFunc)
	registerMockKanisterFunc(ArgFuncName, argsFunc)
}

func registerMockKanisterFunc(name string, f func(context.Context, param.TemplateParams, map[string]interface{}) error) {
	kanister.Register(&mockKanisterFunc{name: name, f: f})
}

var _ kanister.Func = (*mockKanisterFunc)(nil)

type mockKanisterFunc struct {
	name string
	f    func(context.Context, param.TemplateParams, map[string]interface{}) error
}

func (mf *mockKanisterFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) error {
	return mf.f(ctx, tp, args)
}

func (mf *mockKanisterFunc) Name() string {
	return mf.name
}

func ReleaseWaitFunc() {
	waitFuncCh <- struct{}{}
}

func ArgFuncArgs() map[string]interface{} {
	return <-argFuncCh
}

func (mf *mockKanisterFunc) RequiredArgs() []string {
	return nil
}
