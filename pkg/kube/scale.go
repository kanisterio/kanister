package kube

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var emptyGetOptions = v1.GetOptions{}

const scaleTimeout = 10 * time.Minute

// ScaleStatefulSet sets the number of replicas for a StatefulSet.
func ScaleStatefulSet(ctx context.Context, cli kubernetes.Interface, namespace, name string, replicas int32) error {
	ss, err := cli.AppsV1beta1().StatefulSets(namespace).Get(name, emptyGetOptions)
	if err != nil {
		return err
	}
	ss.Spec.Replicas = &replicas
	if _, err = cli.AppsV1beta1().StatefulSets(namespace).Update(ss); err != nil {
		return err
	}
	return WaitForStatefulSetReady(ctx, cli, namespace, name)
}

// WaitForStatefulSetReady waits for the stateful set's replicas to all have status Ready
func WaitForStatefulSetReady(ctx context.Context, cli kubernetes.Interface, namespace, name string) error {
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, scaleTimeout)
	defer cancel()

	return wait(ctx, func(ctx context.Context) (bool, error) {
		ss, ferr := cli.AppsV1beta1().StatefulSets(namespace).Get(name, emptyGetOptions)
		if ferr != nil {
			return false, ferr
		}
		ok := *ss.Spec.Replicas == ss.Status.ReadyReplicas
		return ok, nil
	})
}

const (
	minBackoff = time.Millisecond
	maxBackoff = 5 * time.Second
)

func wait(ctx context.Context, f func(context.Context) (bool, error)) error {
	backoff := time.Millisecond
	for {
		if ok, err := f(ctx); ok || err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("Timeout while waiting")
		default:
		}
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
		time.Sleep(backoff)
	}
}
