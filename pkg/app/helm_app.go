package app

import "github.com/kanisterio/kanister/pkg/helm"

type HelmApp interface {
	App
	// Chart returns the chart of this Helm app.
	Chart() *helm.ChartInfo
	// SetChart sets the chart of this Helm app.
	SetChart(chart helm.ChartInfo)
}
