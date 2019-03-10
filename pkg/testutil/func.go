package testutil

import (
	"context"

	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	FailFuncName   = "FailFunc"
	WaitFuncName   = "WaitFunc"
	ArgFuncName    = "ArgFunc"
	OutputFuncName = "OutputFunc"
)

var (
	failFuncCh   chan error
	waitFuncCh   chan struct{}
	argFuncCh    chan map[string]interface{}
	outputFuncCh chan map[string]interface{}
)

func failFunc(context.Context, param.TemplateParams, map[string]interface{}) (map[string]interface{}, error) {
	err := errors.New("Kanister function failed")
	failFuncCh <- err
	return nil, err
}

func waitFunc(context.Context, param.TemplateParams, map[string]interface{}) (map[string]interface{}, error) {
	<-waitFuncCh
	return nil, nil
}
func argsFunc(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	argFuncCh <- args
	return nil, nil
}

func outputFunc(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	outputFuncCh <- args
	return args, nil
}

func init() {
	failFuncCh = make(chan error)
	waitFuncCh = make(chan struct{})
	argFuncCh = make(chan map[string]interface{})
	outputFuncCh = make(chan map[string]interface{})
	registerMockKanisterFunc(FailFuncName, failFunc)
	registerMockKanisterFunc(WaitFuncName, waitFunc)
	registerMockKanisterFunc(ArgFuncName, argsFunc)
	registerMockKanisterFunc(OutputFuncName, outputFunc)
}

func registerMockKanisterFunc(name string, f func(context.Context, param.TemplateParams, map[string]interface{}) (map[string]interface{}, error)) {
	kanister.Register(&mockKanisterFunc{name: name, f: f})
}

var _ kanister.Func = (*mockKanisterFunc)(nil)

type mockKanisterFunc struct {
	name string
	f    func(context.Context, param.TemplateParams, map[string]interface{}) (map[string]interface{}, error)
}

func (mf *mockKanisterFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	return mf.f(ctx, tp, args)
}

func (mf *mockKanisterFunc) Name() string {
	return mf.name
}

func FailFuncError() error {
	return <-failFuncCh
}

func ReleaseWaitFunc() {
	waitFuncCh <- struct{}{}
}

func ArgFuncArgs() map[string]interface{} {
	return <-argFuncCh
}

func OutputFuncOut() map[string]interface{} {
	return <-outputFuncCh
}

func (mf *mockKanisterFunc) RequiredArgs() []string {
	return nil
}
