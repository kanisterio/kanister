// Copyright 2023 The Kanister Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"github.com/prometheus/client_golang/prometheus"

	kanistermetrics "github.com/kanisterio/kanister/pkg/metrics"
)

// metrics encapsulates all the prometheus metrics that controller
// needs to own.
type metrics struct {
	actionSetResolutionCounterVec prometheus.CounterVec
}

const (
	ActionSetCounterVecLabelRes        = "resolution"
	ActionSetCounterVecLabelResSuccess = "success"
	ActionSetCounterVecLabelResFailure = "failure"
)

const (
	ActionTypeBackup            = "backup"
	ActionTypeRestore           = "restore"
	ActionTypeDelete            = "delete"
	ActionTypeBackupToServer    = "backupToServer"
	ActionTypeRestoreFromServer = "restoreFromServer"
	ActionTypeBeforeBackup      = "before-backup"
	ActionTypeOnSuccess         = "on-success"
	ActionTypeOnFailure         = "on-failure"
	ActionTypePreRestore        = "pre-restore"
	ActionTypePostRestore       = "post-restore"
	ActionTypePostRestoreFailed = "post-restore-failed"
	ActionTypeBackupPrehook     = "backupPrehook"
	ActionTypeBackupPosthook    = "backupPosthook"
	ActionTypeBackupOther       = "other"
)

var knownActionsList = map[string]bool{
	ActionTypeBackup:            true,
	ActionTypeRestore:           true,
	ActionTypeDelete:            true,
	ActionTypeBackupToServer:    true,
	ActionTypeRestoreFromServer: true,
	ActionTypeBeforeBackup:      true,
	ActionTypeOnSuccess:         true,
	ActionTypeOnFailure:         true,
	ActionTypePreRestore:        true,
	ActionTypePostRestore:       true,
	ActionTypePostRestoreFailed: true,
	ActionTypeBackupPrehook:     true,
	ActionTypeBackupPosthook:    true,
}

func getActionTypeBucket(aType string) string {
	actionTypeBucket := ActionTypeBackupOther
	if _, ok := knownActionsList[aType]; ok {
		actionTypeBucket = aType
	}
	return actionTypeBucket
}

// getActionSetCounterVecLabels builds a new BoundedLabel list to construct
// the labels permutations for the prometheus metric.
func getActionSetCounterVecLabels() []kanistermetrics.BoundedLabel {
	bl := make([]kanistermetrics.BoundedLabel, 1)
	bl[0] = kanistermetrics.BoundedLabel{
		LabelName: ActionSetCounterVecLabelRes,
		LabelValues: []string{
			ActionSetCounterVecLabelResSuccess,
			ActionSetCounterVecLabelResFailure,
		},
	}
	return bl
}

// newMetrics constructs a new metrics object that encapsulates all the
// prometheus metric objects that the controller package needs to own.
func newMetrics(reg prometheus.Registerer) *metrics {
	actionSetCounterOpts := prometheus.CounterOpts{
		Name: "kanister_action_set_resolutions_total",
		Help: "Total number of action set resolutions",
	}
	actionSetResolutionCounterVec := kanistermetrics.InitCounterVec(reg, actionSetCounterOpts, getActionSetCounterVecLabels())
	return &metrics{actionSetResolutionCounterVec: *actionSetResolutionCounterVec}
}
