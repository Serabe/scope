package detailed_test

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/weaveworks/scope/probe/docker"
	"github.com/weaveworks/scope/probe/host"
	"github.com/weaveworks/scope/probe/process"
	"github.com/weaveworks/scope/render"
	"github.com/weaveworks/scope/render/detailed"
	"github.com/weaveworks/scope/render/expected"
	"github.com/weaveworks/scope/test"
	"github.com/weaveworks/scope/test/fixture"
)

func TestMakeDetailedHostNode(t *testing.T) {
	renderableNodes := render.HostRenderer.Render(fixture.Report)
	renderableNode := renderableNodes[render.MakeHostID(fixture.ClientHostID)]
	have := detailed.MakeNode(fixture.Report, renderableNodes, renderableNode)

	containerImageNodeSummary, _ := detailed.MakeNodeSummary(
		render.ContainerImageRenderer.Render(fixture.Report)[expected.ClientContainerImageID],
	)
	containerNodeSummary, _ := detailed.MakeNodeSummary(
		render.ContainerRenderer.Render(fixture.Report)[expected.ClientContainerID],
	)
	process1NodeSummary, _ := detailed.MakeNodeSummary(
		render.ProcessRenderer.Render(fixture.Report)[expected.ClientProcess1ID],
	)
	process1NodeSummary.Linkable = true
	process2NodeSummary, _ := detailed.MakeNodeSummary(
		render.ProcessRenderer.Render(fixture.Report)[expected.ClientProcess2ID],
	)
	process2NodeSummary.Linkable = true
	want := detailed.Node{
		NodeSummary: detailed.NodeSummary{
			ID:       render.MakeHostID(fixture.ClientHostID),
			Label:    "client",
			Linkable: true,
			Metadata: []detailed.MetadataRow{
				{
					ID:    "host_name",
					Value: "client.hostname.com",
				},
				{
					ID:    "os",
					Value: "Linux",
				},
				{
					ID:    "local_networks",
					Value: "10.10.10.0/24",
				},
			},
			Metrics: []detailed.MetricRow{
				{
					ID:     host.CPUUsage,
					Format: "percent",
					Value:  0.07,
					Metric: &fixture.ClientHostCPUMetric,
				},
				{
					ID:     host.MemoryUsage,
					Format: "filesize",
					Value:  0.08,
					Metric: &fixture.ClientHostMemoryMetric,
				},
				{
					ID:     host.Load1,
					Group:  "load",
					Value:  0.09,
					Metric: &fixture.ClientHostLoad1Metric,
				},
				{
					ID:     host.Load5,
					Group:  "load",
					Value:  0.10,
					Metric: &fixture.ClientHostLoad5Metric,
				},
				{
					ID:     host.Load15,
					Group:  "load",
					Value:  0.11,
					Metric: &fixture.ClientHostLoad15Metric,
				},
			},
		},
		Rank:     "hostname.com",
		Pseudo:   false,
		Controls: []detailed.ControlInstance{},
		Children: []detailed.NodeSummaryGroup{
			{
				Label:      "Containers",
				TopologyID: "containers",
				Columns:    []detailed.Column{detailed.MakeColumn(docker.CPUTotalUsage), detailed.MakeColumn(docker.MemoryUsage)},
				Nodes:      []detailed.NodeSummary{containerNodeSummary},
			},
			{
				Label:      "Processes",
				TopologyID: "processes",
				Columns:    []detailed.Column{detailed.MakeColumn(process.PID), detailed.MakeColumn(process.CPUUsage), detailed.MakeColumn(process.MemoryUsage)},
				Nodes:      []detailed.NodeSummary{process1NodeSummary, process2NodeSummary},
			},
			{
				Label:      "Container Images",
				TopologyID: "containers-by-image",
				Columns:    []detailed.Column{detailed.MakeColumn(render.ContainersKey)},
				Nodes:      []detailed.NodeSummary{containerImageNodeSummary},
			},
		},
		Connections: []detailed.NodeSummaryGroup{
			{
				ID:    "incoming-connections",
				Label: "Inbound",
				Columns: []detailed.Column{
					{ID: "local_port", Label: "Port"},
					{ID: "count", Label: "Count", DefaultSort: true},
				},
				Nodes: []detailed.NodeSummary{},
			},
			{
				ID:    "outgoing-connections",
				Label: "Outbound",
				Columns: []detailed.Column{
					{ID: "remote_port", Label: "Port"},
					{ID: "count", Label: "Count", DefaultSort: true},
				},
				Nodes: []detailed.NodeSummary{},
			},
		},
	}
	if !reflect.DeepEqual(want, have) {
		t.Errorf("%s", test.Diff(want, have))
	}
}

func TestMakeDetailedContainerNode(t *testing.T) {
	id := render.MakeContainerID(fixture.ServerContainerID)
	renderableNodes := render.ContainerRenderer.Render(fixture.Report)
	renderableNode, ok := renderableNodes[id]
	if !ok {
		t.Fatalf("Node not found: %s", id)
	}
	have := detailed.MakeNode(fixture.Report, renderableNodes, renderableNode)
	want := detailed.Node{
		NodeSummary: detailed.NodeSummary{
			ID:       id,
			Label:    "server",
			Linkable: true,
			Metadata: []detailed.MetadataRow{
				{ID: "docker_container_id", Value: fixture.ServerContainerID, Prime: true},
				{ID: "docker_container_state", Value: "running", Prime: true},
				{ID: "docker_image_id", Value: fixture.ServerContainerImageID},
			},
			DockerLabels: []detailed.MetadataRow{
				{ID: "label_" + render.AmazonECSContainerNameLabel, Value: `server`},
				{ID: "label_foo1", Value: `bar1`},
				{ID: "label_foo2", Value: `bar2`},
				{ID: "label_io.kubernetes.pod.name", Value: "ping/pong-b"},
			},
			Metrics: []detailed.MetricRow{
				{
					ID:     docker.CPUTotalUsage,
					Format: "percent",
					Value:  0.05,
					Metric: &fixture.ServerContainerCPUMetric,
				},
				{
					ID:     docker.MemoryUsage,
					Format: "filesize",
					Value:  0.06,
					Metric: &fixture.ServerContainerMemoryMetric,
				},
			},
		},
		Pseudo:   false,
		Controls: []detailed.ControlInstance{},
		Children: []detailed.NodeSummaryGroup{
			{
				Label:      "Processes",
				TopologyID: "processes",
				Columns:    []detailed.Column{detailed.MakeColumn(process.PID), detailed.MakeColumn(process.CPUUsage), detailed.MakeColumn(process.MemoryUsage)},
				Nodes: []detailed.NodeSummary{
					{
						ID:       fmt.Sprintf("process:%s:%s", "server.hostname.com", fixture.ServerPID),
						Label:    "apache",
						Linkable: true,
						Metadata: []detailed.MetadataRow{
							{ID: process.PID, Value: fixture.ServerPID, Prime: true},
						},
						Metrics: []detailed.MetricRow{},
					},
				},
			},
		},
		Parents: []detailed.Parent{
			{
				ID:         render.MakeContainerImageID(fixture.ServerContainerImageName),
				Label:      fixture.ServerContainerImageName,
				TopologyID: "containers-by-image",
			},
			{
				ID:         render.MakeHostID(fixture.ServerHostName),
				Label:      fixture.ServerHostName,
				TopologyID: "hosts",
			},
		},
		Connections: []detailed.NodeSummaryGroup{
			{
				ID:    "incoming-connections",
				Label: "Inbound",
				Columns: []detailed.Column{
					{ID: "local_port", Label: "Port"},
					{ID: "count", Label: "Count", DefaultSort: true},
				},
				Nodes: []detailed.NodeSummary{},
			},
			{
				ID:    "outgoing-connections",
				Label: "Outbound",
				Columns: []detailed.Column{
					{ID: "remote_port", Label: "Port"},
					{ID: "count", Label: "Count", DefaultSort: true},
				},
				Nodes: []detailed.NodeSummary{},
			},
		},
	}
	if !reflect.DeepEqual(want, have) {
		t.Errorf("%s", test.Diff(want, have))
	}
}
