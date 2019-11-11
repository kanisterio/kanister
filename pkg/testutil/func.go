// Copyright 2019 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutil

import (
	"context"

	"github.com/pkg/errors"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	FailFuncName            = "FailFunc"
	WaitFuncName            = "WaitFunc"
	ArgFuncName             = "ArgFunc"
	OutputFuncName          = "OutputFunc"
	CancelFuncName          = "CancelFunc"
	VersionMismatchFuncName = "VerMisFunc"
	TestVersion             = "v1.0.0"
)

var (
	failFuncCh   chan error
	waitFuncCh   chan struct{}
	argFuncCh    chan map[string]interface{}
	outputFuncCh chan map[string]interface{}
	cancelFuncCh chan error
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

func cancelFunc(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	<-ctx.Done()
	cancelFuncCh <- ctx.Err()
	return nil, ctx.Err()
}

func versionMismatchFunc(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}

func init() {
	failFuncCh = make(chan error)
	waitFuncCh = make(chan struct{})
	argFuncCh = make(chan map[string]interface{})
	outputFuncCh = make(chan map[string]interface{})
	cancelFuncCh = make(chan error)
	registerMockKanisterFunc(FailFuncName, failFunc)
	registerMockKanisterFunc(WaitFuncName, waitFunc)
	registerMockKanisterFunc(ArgFuncName, argsFunc)
	registerMockKanisterFunc(OutputFuncName, outputFunc)
	registerMockKanisterFunc(CancelFuncName, cancelFunc)
	registerMockKanisterFuncWithVersion(ArgFuncName, TestVersion, argsFunc)
	registerMockKanisterFuncWithVersion(VersionMismatchFuncName, TestVersion, versionMismatchFunc)
}

func registerMockKanisterFunc(name string, f func(context.Context, param.TemplateParams, map[string]interface{}) (map[string]interface{}, error)) {
	kanister.Register(&mockKanisterFunc{name: name, f: f})
}

func registerMockKanisterFuncWithVersion(name, version string, f func(context.Context, param.TemplateParams, map[string]interface{}) (map[string]interface{}, error)) {
	kanister.RegisterVersion(&mockKanisterFunc{name: name, f: f}, version)
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

func CancelFuncOut() error {
	return <-cancelFuncCh
}
