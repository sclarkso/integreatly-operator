package monitoringcommon

import (
	"github.com/integr8ly/integreatly-operator/apis/v1alpha1"
	"github.com/integr8ly/integreatly-operator/pkg/resources"
)

// This dashboard json is dynamically configured based on installation type (rhmi or rhoam)
// The installation name taken from the v1alpha1.RHMI.ObjectMeta.Name
func GetMonitoringGrafanaDBClusterResourcesJSON(nsPrefix, installationName string, containerCpuMetric string) string {
	quota := ``
	if installationName == resources.InstallationNames[string(v1alpha1.InstallationTypeManagedApi)] || installationName == resources.InstallationNames[string(v1alpha1.InstallationTypeMultitenantManagedApi)] {
		quota = `, 
			{
				"datasource": "Prometheus",
				"enable": true,
				"expr": "count by (quota,toQuota)(rhoam_quota{toQuota!=\"\"})",
				"hide": false,
				"iconColor": "#FADE2A",
				"limit": 100,
				"name": "Quota",
				"showIn": 0,
				"step": "",
				"tagKeys": "stage,quota,toQuota",
				"tags": "",
				"titleFormat": "Quota Change (million per day)",
				"type": "tags",
				"useValueForTime": false
			}`
	}
	return `{
	"annotations": {
		"list": [{
				"builtIn": 1,
				"datasource": "-- Grafana --",
				"enable": true,
				"hide": true,
				"iconColor": "rgba(0, 211, 255, 1)",
				"name": "Annotations & Alerts",
				"type": "dashboard"
			},
			{
				"datasource": "Prometheus",
				"enable": true,
				"expr": "count by (stage,version,to_version)(` + installationName + `_version{to_version!=\"\"})",
				"hide": false,
				"iconColor": "#FADE2A",
				"limit": 100,
				"name": "Upgrade",
				"showIn": 0,
				"step": "",
				"tagKeys": "stage,version,to_version",
				"tags": "",
				"titleFormat": "Upgrade",
				"type": "tags",
				"useValueForTime": false
			}` + quota + `
		]
	},
	"editable": true,
	"gnetId": null,
	"graphTooltip": 0,
	"links": [],
	"panels": [{
			"collapsed": false,
			"gridPos": {
				"h": 1,
				"w": 24,
				"x": 0,
				"y": 0
			},
			"id": 10,
			"panels": [],
			"repeat": null,
			"title": "CPU Overview",
			"type": "row"
		},
		{
			"aliasColors": {},
			"bars": false,
			"cacheTimeout": null,
			"colorBackground": false,
			"colorValue": false,
			"colors": [
				"#299c46",
				"rgba(237, 129, 40, 0.89)",
				"#d44a3a"
			],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"decimals": 2,
			"description": "CPU Usage of all middleware namespaces",
			"fill": 1,
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 3,
				"w": 3,
				"x": 0,
				"y": 1
			},
			"id": 0,
			"interval": null,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 1,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
					"name": "value to text",
					"value": 1
				},
				{
					"name": "range to text",
					"value": 2
				}
			],
			"maxDataPoints": 100,
			"nullPointMode": "null as zero",
			"nullText": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "N/A",
				"to": "null"
			}],
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"stack": false,
			"steppedLine": false,
			"tableColumn": "",
			"targets": [{
				"expr": "sum(` + containerCpuMetric + `{namespace=~'` + nsPrefix + `.*'}) / sum(kube_node_role{role=\"worker\"} * on(node) group_left (instance) kube_node_status_allocatable{resource='cpu'})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 2,
				"refId": "A"
			}],
			"thresholds": "70,80",
			"timeFrom": null,
			"timeShift": null,
			"title": "CPU Utilisation (Middleware)",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "N/A",
				"value": "null"
			}],
			"valueName": "avg",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			]
		},
		{
			"aliasColors": {},
			"bars": false,
			"cacheTimeout": null,
			"colorBackground": false,
			"colorValue": false,
			"colors": [
				"#299c46",
				"rgba(237, 129, 40, 0.89)",
				"#d44a3a"
			],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"description": "CPU idle percentage across all compute nodes",
			"fill": 1,
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 6,
				"w": 3,
				"x": 3,
				"y": 1
			},
			"id": 3,
			"interval": null,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 1,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
					"name": "value to text",
					"value": 1
				},
				{
					"name": "range to text",
					"value": 2
				}
			],
			"maxDataPoints": 100,
			"nullPointMode": "null as zero",
			"nullText": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "N/A",
				"to": "null"
			}],
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"stack": false,
			"steppedLine": false,
			"tableColumn": "",
			"targets": [{
				"expr": "1-avg(label_replace(instance:node_cpu_utilisation:rate1m, \"node\", \"$1\", \"instance\", \"(.*)\") * on(node) kube_node_role{role=\"worker\"})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 2,
				"refId": "A"
			}],
			"thresholds": "70,80",
			"timeFrom": null,
			"timeShift": null,
			"title": "CPU Unused %",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "N/A",
				"value": "null"
			}],
			"valueName": "avg",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			]
		},
		{
			"aliasColors": {},
			"bars": false,
			"cacheTimeout": null,
			"colorBackground": false,
			"colorValue": false,
			"colors": [
				"#299c46",
				"rgba(237, 129, 40, 0.89)",
				"#d44a3a"
			],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"description": "Sum of all CPU requests across middleware namespaces",
			"fill": 1,
			"format": "none",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 3,
				"w": 5,
				"x": 6,
				"y": 1
			},
			"id": 1,
			"interval": null,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 1,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
					"name": "value to text",
					"value": 1
				},
				{
					"name": "range to text",
					"value": 2
				}
			],
			"maxDataPoints": 100,
			"nullPointMode": "null as zero",
			"nullText": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "N/A",
				"to": "null"
			}],
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"stack": false,
			"steppedLine": false,
			"tableColumn": "",
			"targets": [{
				"expr": "sum(kube_pod_container_resource_requests{namespace=~'` + nsPrefix + `.*',resource='cpu'})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 2,
				"refId": "A"
			}],
			"thresholds": "70,80",
			"timeFrom": null,
			"timeShift": null,
			"title": "CPU Requests (Middleware)",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "N/A",
				"value": "null"
			}],
			"valueName": "avg",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			]
		},
		{
			"aliasColors": {},
			"bars": false,
			"cacheTimeout": null,
			"colorBackground": false,
			"colorValue": false,
			"colors": [
				"#299c46",
				"rgba(237, 129, 40, 0.89)",
				"#d44a3a"
			],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"description": "Total available CPU requests across all compute nodes",
			"fill": 1,
			"format": "none",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 6,
				"w": 3,
				"x": 11,
				"y": 1
			},
			"id": 19,
			"interval": null,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 1,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
					"name": "value to text",
					"value": 1
				},
				{
					"name": "range to text",
					"value": 2
				}
			],
			"maxDataPoints": 100,
			"nullPointMode": "null as zero",
			"nullText": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "N/A",
				"to": "null"
			}],
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"stack": false,
			"steppedLine": false,
			"tableColumn": "",
			"targets": [{
				"expr": "sum(kube_node_role{role=\"worker\"} * on(node) group_left (instance) kube_node_status_allocatable{resource='cpu'})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 2,
				"refId": "A"
			}],
			"thresholds": "70,80",
			"timeFrom": null,
			"timeShift": null,
			"title": "CPU Requests Available",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "N/A",
				"value": "null"
			}],
			"valueName": "avg",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			]
		},
		{
			"aliasColors": {},
			"bars": false,
			"cacheTimeout": null,
			"colorBackground": false,
			"colorValue": false,
			"colors": [
				"#299c46",
				"rgba(237, 129, 40, 0.89)",
				"#d44a3a"
			],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"description": "Percentage of CPU requests allocated by middleware namespaces",
			"fill": 1,
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 3,
				"w": 5,
				"x": 14,
				"y": 1
			},
			"id": 2,
			"interval": null,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 1,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
					"name": "value to text",
					"value": 1
				},
				{
					"name": "range to text",
					"value": 2
				}
			],
			"maxDataPoints": 100,
			"nullPointMode": "null as zero",
			"nullText": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "N/A",
				"to": "null"
			}],
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"stack": false,
			"steppedLine": false,
			"tableColumn": "",
			"targets": [{
				"expr": "sum(kube_pod_container_resource_requests{namespace=~'` + nsPrefix + `.*', resource='cpu'}) / sum(kube_node_role{role=\"worker\"} * on(node) group_left (instance) kube_node_status_allocatable{resource='cpu'})\n",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 2,
				"refId": "A"
			}],
			"thresholds": "70,80",
			"timeFrom": null,
			"timeShift": null,
			"title": "CPU Requests % (Middleware)",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "N/A",
				"value": "null"
			}],
			"valueName": "avg",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			]
		},
		{
			"aliasColors": {},
			"bars": false,
			"cacheTimeout": null,
			"colorBackground": false,
			"colorValue": false,
			"colors": [
				"#299c46",
				"rgba(237, 129, 40, 0.89)",
				"#d44a3a"
			],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"description": "Percentage of CPU requests still available for allocation across all compute nodes",
			"fill": 1,
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 6,
				"w": 5,
				"x": 19,
				"y": 1
			},
			"id": 4,
			"interval": null,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 1,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
					"name": "value to text",
					"value": 1
				},
				{
					"name": "range to text",
					"value": 2
				}
			],
			"maxDataPoints": 100,
			"nullPointMode": "null as zero",
			"nullText": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "N/A",
				"to": "null"
			}],
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"stack": false,
			"steppedLine": false,
			"tableColumn": "",
			"targets": [{
				"expr": "(1 - sum(kube_pod_container_resource_requests{resource='cpu'} * on(node) group_left(instance) kube_node_role{role=\"worker\"}) / sum(kube_node_role{role=\"worker\"} * on(node) group_left (instance) kube_node_status_allocatable{resource='cpu'}))\n",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 2,
				"refId": "A"
			}],
			"thresholds": "70,80",
			"timeFrom": null,
			"timeShift": null,
			"title": "CPU Uncommitted %",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "N/A",
				"value": "null"
			}],
			"valueName": "avg",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			]
		},
		{
			"cacheTimeout": null,
			"colorBackground": false,
			"colorValue": false,
			"colors": [
				"#299c46",
				"rgba(237, 129, 40, 0.89)",
				"#d44a3a"
			],
			"datasource": "Prometheus",
			"description": "CPU usage across all compute nodes",
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 3,
				"w": 3,
				"x": 0,
				"y": 4
			},
			"id": 16,
			"interval": null,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
					"name": "value to text",
					"value": 1
				},
				{
					"name": "range to text",
					"value": 2
				}
			],
			"maxDataPoints": 100,
			"nullPointMode": "connected",
			"nullText": null,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "N/A",
				"to": "null"
			}],
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"tableColumn": "",
			"targets": [{
				"expr": "avg(label_replace(instance:node_cpu_utilisation:rate1m, \"node\", \"$1\", \"instance\", \"(.*)\") * on(node) kube_node_role{role=\"worker\"})",
				"format": "time_series",
				"intervalFactor": 1,
				"refId": "A"
			}],
			"thresholds": "",
			"title": "CPU Utilisation (all on compute nodes)",
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "N/A",
				"value": "null"
			}],
			"valueName": "current"
		},
		{
			"aliasColors": {},
			"bars": false,
			"cacheTimeout": null,
			"colorBackground": false,
			"colorValue": false,
			"colors": [
				"#299c46",
				"rgba(237, 129, 40, 0.89)",
				"#d44a3a"
			],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"description": "Sum of all CPU requests across all compute nodes",
			"fill": 1,
			"format": "none",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 3,
				"w": 5,
				"x": 6,
				"y": 4
			},
			"id": 17,
			"interval": null,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 1,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
					"name": "value to text",
					"value": 1
				},
				{
					"name": "range to text",
					"value": 2
				}
			],
			"maxDataPoints": 100,
			"nullPointMode": "null as zero",
			"nullText": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "N/A",
				"to": "null"
			}],
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"stack": false,
			"steppedLine": false,
			"tableColumn": "",
			"targets": [{
				"expr": "sum(kube_pod_container_resource_requests{resource='cpu'} * on(node) group_left(instance) kube_node_role{role=\"worker\"})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 2,
				"refId": "A"
			}],
			"thresholds": "70,80",
			"timeFrom": null,
			"timeShift": null,
			"title": "CPU Requests (Total)",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "N/A",
				"value": "null"
			}],
			"valueName": "avg",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			]
		},
		{
			"aliasColors": {},
			"bars": false,
			"cacheTimeout": null,
			"colorBackground": false,
			"colorValue": false,
			"colors": [
				"#299c46",
				"rgba(237, 129, 40, 0.89)",
				"#d44a3a"
			],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"description": "Percentage of CPU requests allocated across all compute nodes",
			"fill": 1,
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 3,
				"w": 5,
				"x": 14,
				"y": 4
			},
			"id": 18,
			"interval": null,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 1,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
					"name": "value to text",
					"value": 1
				},
				{
					"name": "range to text",
					"value": 2
				}
			],
			"maxDataPoints": 100,
			"nullPointMode": "null as zero",
			"nullText": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "N/A",
				"to": "null"
			}],
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"stack": false,
			"steppedLine": false,
			"tableColumn": "",
			"targets": [{
				"expr": "sum(kube_pod_container_resource_requests{resource='cpu'} * on(node) group_left(instance) kube_node_role{role=\"worker\"}) / sum(kube_node_role{role=\"worker\"} * on(node) group_left (instance) kube_node_status_allocatable{resource='cpu'})\n",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 2,
				"refId": "A"
			}],
			"thresholds": "70,80",
			"timeFrom": null,
			"timeShift": null,
			"title": "CPU Requests % (Total)",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "N/A",
				"value": "null"
			}],
			"valueName": "avg",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			]
		},
		{
			"collapsed": false,
			"gridPos": {
				"h": 1,
				"w": 24,
				"x": 0,
				"y": 7
			},
			"id": 11,
			"panels": [],
			"repeat": null,
			"title": "CPU",
			"type": "row"
		},
		{
			"aliasColors": {},
			"bars": false,
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"fill": 10,
			"gridPos": {
				"h": 7,
				"w": 24,
				"x": 0,
				"y": 8
			},
			"id": 6,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 0,
			"links": [],
			"nullPointMode": "null as zero",
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"stack": true,
			"steppedLine": false,
			"targets": [{
				"expr": "sum(` + containerCpuMetric + `{namespace=~'` + nsPrefix + `.*'}) by (namespace)",
				"format": "time_series",
				"intervalFactor": 2,
				"legendFormat": "{{namespace}}",
				"legendLink": null,
				"refId": "A",
				"step": 10
			}],
			"thresholds": [],
			"timeFrom": null,
			"timeRegions": [],
			"timeShift": null,
			"title": "CPU Utilisation (in number of CPUs)",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"type": "graph",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			],
			"yaxis": {
				"align": false,
				"alignLevel": null
			}
		},
		{
			"collapsed": false,
			"gridPos": {
				"h": 1,
				"w": 24,
				"x": 0,
				"y": 15
			},
			"id": 12,
			"panels": [],
			"repeat": null,
			"title": "CPU Quota",
			"type": "row"
		},
		{
			"aliasColors": {},
			"bars": false,
			"columns": [],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"fill": 1,
			"fontSize": "100%",
			"gridPos": {
				"h": 7,
				"w": 24,
				"x": 0,
				"y": 16
			},
			"id": 7,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 1,
			"links": [],
			"nullPointMode": "null as zero",
			"pageSize": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"renderer": "flot",
			"scroll": true,
			"seriesOverrides": [],
			"showHeader": true,
			"sort": {
				"col": 1,
				"desc": false
			},
			"spaceLength": 10,
			"stack": false,
			"steppedLine": false,
			"styles": [{
					"alias": "Time",
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"pattern": "Time",
					"type": "hidden"
				},
				{
					"alias": "CPU Usage",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "Value #A",
					"thresholds": [],
					"type": "number",
					"unit": "short"
				},
				{
					"alias": "CPU Requests",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "Value #B",
					"thresholds": [],
					"type": "number",
					"unit": "short"
				},
				{
					"alias": "CPU Requests %",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "Value #C",
					"thresholds": [],
					"type": "number",
					"unit": "percentunit"
				},
				{
					"alias": "CPU Limits",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "Value #D",
					"thresholds": [],
					"type": "number",
					"unit": "short"
				},
				{
					"alias": "CPU Limits %",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "Value #E",
					"thresholds": [],
					"type": "number",
					"unit": "percentunit"
				},
				{
					"alias": "Namespace",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": true,
					"linkTooltip": "Drill down",
					"linkUrl": "/d/a9ce5290ba1d485ca67e05c0a63aa2d8/resources-by-namespace?var-namespace=$__cell",
					"pattern": "namespace",
					"thresholds": [],
					"type": "number",
					"unit": "short"
				},
				{
					"alias": "",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"pattern": "/.*/",
					"thresholds": [],
					"type": "string",
					"unit": "short"
				}
			],
			"targets": [{
					"expr": "sum(` + containerCpuMetric + `{namespace=~'` + nsPrefix + `.*'}) by (namespace)",
					"format": "table",
					"instant": true,
					"intervalFactor": 2,
					"legendFormat": "",
					"refId": "A",
					"step": 10
				},
				{
					"expr": "sum(kube_pod_container_resource_requests{namespace=~'` + nsPrefix + `.*',resource='cpu'}) by (namespace)",
					"format": "table",
					"instant": true,
					"intervalFactor": 2,
					"legendFormat": "",
					"refId": "B",
					"step": 10
				},
				{
					"expr": "(sum(` + containerCpuMetric + `{namespace=~'` + nsPrefix + `.*'}) by (namespace) / sum(kube_pod_container_resource_requests{resource='cpu'}) by (namespace))",
					"format": "table",
					"instant": true,
					"intervalFactor": 2,
					"legendFormat": "",
					"refId": "C",
					"step": 10
				},
				{
					"expr": "sum(kube_pod_container_resource_limits{namespace=~'` + nsPrefix + `.*',resource='cpu'}) by (namespace)",
					"format": "table",
					"instant": true,
					"intervalFactor": 2,
					"legendFormat": "",
					"refId": "D",
					"step": 10
				},
				{
					"expr": "(sum(` + containerCpuMetric + `{namespace=~'` + nsPrefix + `.*'}) by (namespace) / sum(kube_pod_container_resource_limits{resource='cpu'}) by (namespace))",
					"format": "table",
					"instant": true,
					"intervalFactor": 2,
					"legendFormat": "",
					"refId": "E",
					"step": 10
				}
			],
			"thresholds": [],
			"timeFrom": null,
			"timeShift": null,
			"title": "CPU Quota",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"transform": "table",
			"type": "table",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "percentunit",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			]
		},
		{
			"collapsed": false,
			"gridPos": {
				"h": 1,
				"w": 24,
				"x": 0,
				"y": 23
			},
			"id": 21,
			"panels": [],
			"title": "Memory Overview",
			"type": "row"
		},
		{
			"aliasColors": {},
			"bars": false,
			"cacheTimeout": null,
			"colorBackground": false,
			"colorValue": false,
			"colors": [
				"#299c46",
				"rgba(237, 129, 40, 0.89)",
				"#d44a3a"
			],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"description": "Memory usage by middleware namespaces",
			"fill": 1,
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 3,
				"w": 3,
				"x": 0,
				"y": 24
			},
			"id": 22,
			"interval": null,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 1,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
					"name": "value to text",
					"value": 1
				},
				{
					"name": "range to text",
					"value": 2
				}
			],
			"maxDataPoints": 100,
			"nullPointMode": "null as zero",
			"nullText": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "N/A",
				"to": "null"
			}],
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"stack": false,
			"steppedLine": false,
			"tableColumn": "",
			"targets": [{
				"expr": "sum(container_memory_rss{container!='',container!='POD',namespace=~'` + nsPrefix + `.*'}) / sum(kube_node_status_capacity{resource='memory'}  * on(node) group_left(instance) kube_node_role{role=\"worker\"})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 2,
				"refId": "A"
			}],
			"thresholds": "70,80",
			"timeFrom": null,
			"timeShift": null,
			"title": "Memory Utilisation (Middleware)",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "N/A",
				"value": "null"
			}],
			"valueName": "avg",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			]
		},
		{
			"aliasColors": {},
			"bars": false,
			"cacheTimeout": null,
			"colorBackground": false,
			"colorValue": false,
			"colors": [
				"#299c46",
				"rgba(237, 129, 40, 0.89)",
				"#d44a3a"
			],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"description": "Percentage of unused memory across all compute nodes",
			"fill": 1,
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 6,
				"w": 3,
				"x": 3,
				"y": 24
			},
			"id": 24,
			"interval": null,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 1,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
					"name": "value to text",
					"value": 1
				},
				{
					"name": "range to text",
					"value": 2
				}
			],
			"maxDataPoints": 100,
			"nullPointMode": "null as zero",
			"nullText": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "N/A",
				"to": "null"
			}],
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"stack": false,
			"steppedLine": false,
			"tableColumn": "",
			"targets": [{
				"expr": "1-avg(label_replace(instance:node_memory_utilisation:ratio, \"node\", \"$1\", \"instance\", \"(.*)\")  * on(node) kube_node_role{role=\"worker\"})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 2,
				"refId": "A"
			}],
			"thresholds": "70,80",
			"timeFrom": null,
			"timeShift": null,
			"title": "Memory Unused %",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "N/A",
				"value": "null"
			}],
			"valueName": "avg",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			]
		},
		{
			"aliasColors": {},
			"bars": false,
			"cacheTimeout": null,
			"colorBackground": false,
			"colorValue": false,
			"colors": [
				"#299c46",
				"rgba(237, 129, 40, 0.89)",
				"#d44a3a"
			],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"description": "Memory requests by middleware namespaces",
			"fill": 1,
			"format": "bytes",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 3,
				"w": 5,
				"x": 6,
				"y": 24
			},
			"id": 25,
			"interval": null,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 1,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
					"name": "value to text",
					"value": 1
				},
				{
					"name": "range to text",
					"value": 2
				}
			],
			"maxDataPoints": 100,
			"nullPointMode": "null as zero",
			"nullText": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "N/A",
				"to": "null"
			}],
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"stack": false,
			"steppedLine": false,
			"tableColumn": "",
			"targets": [{
				"expr": "sum(kube_pod_container_resource_requests{namespace=~'` + nsPrefix + `.*',resource='memory'})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 2,
				"refId": "A"
			}],
			"thresholds": "70,80",
			"timeFrom": null,
			"timeShift": null,
			"title": "Memory Requests (Middleware)",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "N/A",
				"value": "null"
			}],
			"valueName": "avg",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			]
		},
		{
			"aliasColors": {},
			"bars": false,
			"cacheTimeout": null,
			"colorBackground": false,
			"colorValue": false,
			"colors": [
				"#299c46",
				"rgba(237, 129, 40, 0.89)",
				"#d44a3a"
			],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"description": "Total amount of requestable memory across all compute nodes",
			"fill": 1,
			"format": "bytes",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 6,
				"w": 3,
				"x": 11,
				"y": 24
			},
			"id": 27,
			"interval": null,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 1,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
					"name": "value to text",
					"value": 1
				},
				{
					"name": "range to text",
					"value": 2
				}
			],
			"maxDataPoints": 100,
			"nullPointMode": "null as zero",
			"nullText": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "N/A",
				"to": "null"
			}],
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"stack": false,
			"steppedLine": false,
			"tableColumn": "",
			"targets": [{
				"expr": "sum(kube_node_status_allocatable{resource='memory'} * on(node) group_left(instance) kube_node_role{role=\"worker\"})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 2,
				"legendFormat": "",
				"refId": "A"
			}],
			"thresholds": "70,80",
			"timeFrom": null,
			"timeShift": null,
			"title": "Memory Requests (Available)",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "N/A",
				"value": "null"
			}],
			"valueName": "avg",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			]
		},
		{
			"aliasColors": {},
			"bars": false,
			"cacheTimeout": null,
			"colorBackground": false,
			"colorValue": false,
			"colors": [
				"#299c46",
				"rgba(237, 129, 40, 0.89)",
				"#d44a3a"
			],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"description": "Percentage of memory requested by middleware namespaces",
			"fill": 1,
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 3,
				"w": 5,
				"x": 14,
				"y": 24
			},
			"id": 28,
			"interval": null,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 1,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
					"name": "value to text",
					"value": 1
				},
				{
					"name": "range to text",
					"value": 2
				}
			],
			"maxDataPoints": 100,
			"nullPointMode": "null as zero",
			"nullText": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "N/A",
				"to": "null"
			}],
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"stack": false,
			"steppedLine": false,
			"tableColumn": "",
			"targets": [{
				"expr": "sum(kube_pod_container_resource_requests{namespace=~'` + nsPrefix + `.*',resource='memory'}) / sum(kube_node_status_allocatable{resource='memory'} * on(node) group_left(instance) kube_node_role{role=\"worker\"})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 2,
				"refId": "A"
			}],
			"thresholds": "70,80",
			"timeFrom": null,
			"timeShift": null,
			"title": "Memory Requests % (Middleware)",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "N/A",
				"value": "null"
			}],
			"valueName": "avg",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			]
		},
		{
			"aliasColors": {},
			"bars": false,
			"cacheTimeout": null,
			"colorBackground": false,
			"colorValue": false,
			"colors": [
				"#299c46",
				"rgba(237, 129, 40, 0.89)",
				"#d44a3a"
			],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"description": "Percentage of memory available for allocation across all compute nodes",
			"fill": 1,
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 6,
				"w": 5,
				"x": 19,
				"y": 24
			},
			"id": 30,
			"interval": null,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 1,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
					"name": "value to text",
					"value": 1
				},
				{
					"name": "range to text",
					"value": 2
				}
			],
			"maxDataPoints": 100,
			"nullPointMode": "null as zero",
			"nullText": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "N/A",
				"to": "null"
			}],
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"stack": false,
			"steppedLine": false,
			"tableColumn": "",
			"targets": [{
				"expr": "1 - sum(kube_pod_container_resource_requests{resource='memory'} * on(node) group_left(instance) kube_node_role{role=\"worker\"}) / sum(kube_node_status_allocatable{resource='memory'} * on(node) group_left(instance) kube_node_role{role=\"worker\"})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 2,
				"refId": "A"
			}],
			"thresholds": "70,80",
			"timeFrom": null,
			"timeShift": null,
			"title": "Memory Uncommited %",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "N/A",
				"value": "null"
			}],
			"valueName": "avg",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			]
		},
		{
			"aliasColors": {},
			"bars": false,
			"cacheTimeout": null,
			"colorBackground": false,
			"colorValue": false,
			"colors": [
				"#299c46",
				"rgba(237, 129, 40, 0.89)",
				"#d44a3a"
			],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"description": "Memory usage across all compute nodes (this is an accumulated average that does not take into account sudden spikes)",
			"fill": 1,
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 3,
				"w": 3,
				"x": 0,
				"y": 27
			},
			"id": 23,
			"interval": null,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 1,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
					"name": "value to text",
					"value": 1
				},
				{
					"name": "range to text",
					"value": 2
				}
			],
			"maxDataPoints": 100,
			"nullPointMode": "null as zero",
			"nullText": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "N/A",
				"to": "null"
			}],
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"stack": false,
			"steppedLine": false,
			"tableColumn": "",
			"targets": [{
				"expr": "avg(label_replace(instance:node_memory_utilisation:ratio, \"node\", \"$1\", \"instance\", \"(.*)\")  * on(node) kube_node_role{role=\"worker\"})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 2,
				"refId": "A"
			}],
			"thresholds": "70,80",
			"timeFrom": null,
			"timeShift": null,
			"title": "Memory Utilisation (Total)",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "N/A",
				"value": "null"
			}],
			"valueName": "avg",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			]
		},
		{
			"aliasColors": {},
			"bars": false,
			"cacheTimeout": null,
			"colorBackground": false,
			"colorValue": false,
			"colors": [
				"#299c46",
				"rgba(237, 129, 40, 0.89)",
				"#d44a3a"
			],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"description": "Memory requests across all compute nodes",
			"fill": 1,
			"format": "bytes",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 3,
				"w": 5,
				"x": 6,
				"y": 27
			},
			"id": 26,
			"interval": null,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 1,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
					"name": "value to text",
					"value": 1
				},
				{
					"name": "range to text",
					"value": 2
				}
			],
			"maxDataPoints": 100,
			"nullPointMode": "null as zero",
			"nullText": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "N/A",
				"to": "null"
			}],
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"stack": false,
			"steppedLine": false,
			"tableColumn": "",
			"targets": [{
				"expr": "sum(kube_pod_container_resource_requests{resource='memory'} * on(node) group_left(instance) kube_node_role{role=\"worker\"})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 2,
				"refId": "A"
			}],
			"thresholds": "70,80",
			"timeFrom": null,
			"timeShift": null,
			"title": "Memory Requests (Total)",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "N/A",
				"value": "null"
			}],
			"valueName": "avg",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			]
		},
		{
			"aliasColors": {},
			"bars": false,
			"cacheTimeout": null,
			"colorBackground": false,
			"colorValue": false,
			"colors": [
				"#299c46",
				"rgba(237, 129, 40, 0.89)",
				"#d44a3a"
			],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"description": "Percentage of memory requested across all compute nodes",
			"fill": 1,
			"format": "percentunit",
			"gauge": {
				"maxValue": 100,
				"minValue": 0,
				"show": false,
				"thresholdLabels": false,
				"thresholdMarkers": true
			},
			"gridPos": {
				"h": 3,
				"w": 5,
				"x": 14,
				"y": 27
			},
			"id": 29,
			"interval": null,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 1,
			"links": [],
			"mappingType": 1,
			"mappingTypes": [{
					"name": "value to text",
					"value": 1
				},
				{
					"name": "range to text",
					"value": 2
				}
			],
			"maxDataPoints": 100,
			"nullPointMode": "null as zero",
			"nullText": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"postfix": "",
			"postfixFontSize": "50%",
			"prefix": "",
			"prefixFontSize": "50%",
			"rangeMaps": [{
				"from": "null",
				"text": "N/A",
				"to": "null"
			}],
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"sparkline": {
				"fillColor": "rgba(31, 118, 189, 0.18)",
				"full": false,
				"lineColor": "rgb(31, 120, 193)",
				"show": false
			},
			"stack": false,
			"steppedLine": false,
			"tableColumn": "",
			"targets": [{
				"expr": "sum(kube_pod_container_resource_requests{resource='memory'} * on(node) group_left(instance) kube_node_role{role=\"worker\"}) / sum(kube_node_status_allocatable{resource='memory'} * on(node) group_left(instance) kube_node_role{role=\"worker\"})",
				"format": "time_series",
				"instant": true,
				"intervalFactor": 2,
				"refId": "A"
			}],
			"thresholds": "70,80",
			"timeFrom": null,
			"timeShift": null,
			"title": "Memory Requests % (Total)",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"type": "singlestat",
			"valueFontSize": "80%",
			"valueMaps": [{
				"op": "=",
				"text": "N/A",
				"value": "null"
			}],
			"valueName": "avg",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			]
		},
		{
			"collapsed": false,
			"gridPos": {
				"h": 1,
				"w": 24,
				"x": 0,
				"y": 30
			},
			"id": 13,
			"panels": [],
			"repeat": null,
			"title": "Memory",
			"type": "row"
		},
		{
			"aliasColors": {},
			"bars": false,
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"fill": 10,
			"gridPos": {
				"h": 7,
				"w": 24,
				"x": 0,
				"y": 31
			},
			"id": 8,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 0,
			"links": [],
			"nullPointMode": "null as zero",
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"renderer": "flot",
			"seriesOverrides": [],
			"spaceLength": 10,
			"stack": true,
			"steppedLine": false,
			"targets": [{
				"expr": "sum(container_memory_rss{container!='',namespace=~'` + nsPrefix + `.*'}) by (namespace)",
				"format": "time_series",
				"intervalFactor": 2,
				"legendFormat": "{{namespace}}",
				"legendLink": null,
				"step": 10
			}],
			"thresholds": [],
			"timeFrom": null,
			"timeRegions": [],
			"timeShift": null,
			"title": "Memory Usage (w/o cache)",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"type": "graph",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "bytes",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			],
			"yaxis": {
				"align": false,
				"alignLevel": null
			}
		},
		{
			"collapsed": false,
			"gridPos": {
				"h": 1,
				"w": 24,
				"x": 0,
				"y": 38
			},
			"id": 14,
			"panels": [],
			"repeat": null,
			"title": "Memory Quota",
			"type": "row"
		},
		{
			"aliasColors": {},
			"bars": false,
			"columns": [],
			"dashLength": 10,
			"dashes": false,
			"datasource": "Prometheus",
			"fill": 1,
			"fontSize": "100%",
			"gridPos": {
				"h": 7,
				"w": 24,
				"x": 0,
				"y": 39
			},
			"id": 9,
			"legend": {
				"avg": false,
				"current": false,
				"max": false,
				"min": false,
				"show": true,
				"total": false,
				"values": false
			},
			"lines": true,
			"linewidth": 1,
			"links": [],
			"nullPointMode": "null as zero",
			"pageSize": null,
			"percentage": false,
			"pointradius": 5,
			"points": false,
			"renderer": "flot",
			"scroll": true,
			"seriesOverrides": [],
			"showHeader": true,
			"sort": {
				"col": 1,
				"desc": false
			},
			"spaceLength": 10,
			"stack": false,
			"steppedLine": false,
			"styles": [{
					"alias": "Time",
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"pattern": "Time",
					"type": "hidden"
				},
				{
					"alias": "Memory Usage",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "Value #A",
					"thresholds": [],
					"type": "number",
					"unit": "bytes"
				},
				{
					"alias": "Memory Requests",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "Value #B",
					"thresholds": [],
					"type": "number",
					"unit": "bytes"
				},
				{
					"alias": "Memory Requests %",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "Value #C",
					"thresholds": [],
					"type": "number",
					"unit": "percentunit"
				},
				{
					"alias": "Memory Limits",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "Value #D",
					"thresholds": [],
					"type": "number",
					"unit": "bytes"
				},
				{
					"alias": "Memory Limits %",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": false,
					"linkTooltip": "Drill down",
					"linkUrl": "",
					"pattern": "Value #E",
					"thresholds": [],
					"type": "number",
					"unit": "percentunit"
				},
				{
					"alias": "Namespace",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"link": true,
					"linkTooltip": "Drill down",
					"linkUrl": "/d/a9ce5290ba1d485ca67e05c0a63aa2d8/resources-by-namespace?var-namespace=$__cell",
					"pattern": "namespace",
					"thresholds": [],
					"type": "number",
					"unit": "short"
				},
				{
					"alias": "",
					"colorMode": null,
					"colors": [],
					"dateFormat": "YYYY-MM-DD HH:mm:ss",
					"decimals": 2,
					"pattern": "/.*/",
					"thresholds": [],
					"type": "string",
					"unit": "short"
				}
			],
			"targets": [{
					"expr": "sum(container_memory_rss{container!='', namespace=~'` + nsPrefix + `.*'}) by (namespace)",
					"format": "table",
					"instant": true,
					"intervalFactor": 2,
					"legendFormat": "",
					"refId": "A",
					"step": 10
				},
				{
					"expr": "sum(kube_pod_container_resource_requests{namespace=~'` + nsPrefix + `.*',resource='memory'}) by (namespace)",
					"format": "table",
					"instant": true,
					"intervalFactor": 2,
					"legendFormat": "",
					"refId": "B",
					"step": 10
				},
				{
					"expr": "(sum(container_memory_rss{container!='',namespace=~'` + nsPrefix + `.*'}) by (namespace) / sum(kube_pod_container_resource_requests{resource='memory'}) by (namespace))",
					"format": "table",
					"instant": true,
					"intervalFactor": 2,
					"legendFormat": "",
					"refId": "C",
					"step": 10
				},
				{
					"expr": "sum(kube_pod_container_resource_limits{namespace=~'` + nsPrefix + `.*',resource='memory'}) by (namespace)",
					"format": "table",
					"instant": true,
					"intervalFactor": 2,
					"legendFormat": "",
					"refId": "D",
					"step": 10
				},
				{
					"expr": "(sum(container_memory_rss{container!='', namespace=~'` + nsPrefix + `.*'}) by (namespace) / sum(kube_pod_container_resource_limits{resource='memory'}) by (namespace))",
					"format": "table",
					"instant": true,
					"intervalFactor": 2,
					"legendFormat": "",
					"refId": "E",
					"step": 10
				}
			],
			"thresholds": [],
			"timeFrom": null,
			"timeShift": null,
			"title": "Memory Quota",
			"tooltip": {
				"shared": true,
				"sort": 0,
				"value_type": "individual"
			},
			"transform": "table",
			"type": "table",
			"xaxis": {
				"buckets": null,
				"mode": "time",
				"name": null,
				"show": true,
				"values": []
			},
			"yaxes": [{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": 0,
					"show": true
				},
				{
					"format": "short",
					"label": null,
					"logBase": 1,
					"max": null,
					"min": null,
					"show": false
				}
			]
		}
	],
	"refresh": "10s",
	"schemaVersion": 16,
	"style": "dark",
	"tags": [],
	"templating": {
		"list": []
	},
	"time": {
		"from": "now-1h",
		"to": "now"
	},
	"timepicker": {
		"refresh_intervals": [
			"5s",
			"10s",
			"30s",
			"1m",
			"5m",
			"15m",
			"30m",
			"1h",
			"2h",
			"1d"
		],
		"time_options": [
			"5m",
			"15m",
			"1h",
			"6h",
			"12h",
			"24h",
			"2d",
			"7d",
			"30d"
		]
	},
	"timezone": "",
	"title": "Resource Usage for Cluster",
	"version": 9
}`
}
