package aws

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getMetricsAction implements connectors.Action for aws.get_metrics.
type getMetricsAction struct {
	conn *AWSConnector
}

type metricDimension struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type getMetricsParams struct {
	Region     string            `json:"region"`
	Namespace  string            `json:"namespace"`
	MetricName string            `json:"metric_name"`
	Dimensions []metricDimension `json:"dimensions"`
	StartTime  string            `json:"start_time"`
	EndTime    string            `json:"end_time"`
	Period     int               `json:"period"`
	Stat       string            `json:"stat"`
}

func (p *getMetricsParams) validate() error {
	if p.Region == "" {
		return &connectors.ValidationError{Message: "missing required parameter: region"}
	}
	if p.Namespace == "" {
		return &connectors.ValidationError{Message: "missing required parameter: namespace"}
	}
	if p.MetricName == "" {
		return &connectors.ValidationError{Message: "missing required parameter: metric_name"}
	}
	if p.StartTime == "" {
		return &connectors.ValidationError{Message: "missing required parameter: start_time"}
	}
	if p.EndTime == "" {
		return &connectors.ValidationError{Message: "missing required parameter: end_time"}
	}
	if p.Period <= 0 {
		return &connectors.ValidationError{Message: "missing or invalid required parameter: period"}
	}
	if p.Stat == "" {
		p.Stat = "Average"
	}
	validStats := map[string]bool{
		"Average": true, "Sum": true, "Minimum": true, "Maximum": true, "SampleCount": true,
	}
	if !validStats[p.Stat] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid stat %q: must be Average, Sum, Minimum, Maximum, or SampleCount", p.Stat)}
	}
	return nil
}

type getMetricDataResponse struct {
	XMLName xml.Name `xml:"GetMetricStatisticsResponse"`
	Result  struct {
		Datapoints []struct {
			Timestamp   string  `xml:"Timestamp"`
			Average     float64 `xml:"Average"`
			Sum         float64 `xml:"Sum"`
			Minimum     float64 `xml:"Minimum"`
			Maximum     float64 `xml:"Maximum"`
			SampleCount float64 `xml:"SampleCount"`
			Unit        string  `xml:"Unit"`
		} `xml:"Datapoints>member"`
		Label string `xml:"Label"`
	} `xml:"GetMetricStatisticsResult"`
}

type datapointInfo struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
	Unit      string  `json:"unit"`
}

// Execute retrieves CloudWatch metrics via the CloudWatch Query API.
func (a *getMetricsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getMetricsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	queryParams := url.Values{}
	queryParams.Set("Action", "GetMetricStatistics")
	queryParams.Set("Version", "2010-08-01")
	queryParams.Set("Namespace", params.Namespace)
	queryParams.Set("MetricName", params.MetricName)
	queryParams.Set("StartTime", params.StartTime)
	queryParams.Set("EndTime", params.EndTime)
	queryParams.Set("Period", strconv.Itoa(params.Period))
	queryParams.Set("Statistics.member.1", params.Stat)

	for i, dim := range params.Dimensions {
		queryParams.Set(fmt.Sprintf("Dimensions.member.%d.Name", i+1), dim.Name)
		queryParams.Set(fmt.Sprintf("Dimensions.member.%d.Value", i+1), dim.Value)
	}

	host := fmt.Sprintf("monitoring.%s.amazonaws.com", params.Region)
	body := []byte(queryParams.Encode())

	respBody, err := a.conn.do(ctx, req.Credentials, "POST", host, "/", "", body)
	if err != nil {
		return nil, err
	}

	var xmlResp getMetricDataResponse
	if err := xml.Unmarshal(respBody, &xmlResp); err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("parsing CloudWatch response: %v", err)}
	}

	datapoints := make([]datapointInfo, 0, len(xmlResp.Result.Datapoints))
	for _, dp := range xmlResp.Result.Datapoints {
		var value float64
		switch params.Stat {
		case "Average":
			value = dp.Average
		case "Sum":
			value = dp.Sum
		case "Minimum":
			value = dp.Minimum
		case "Maximum":
			value = dp.Maximum
		case "SampleCount":
			value = dp.SampleCount
		}
		datapoints = append(datapoints, datapointInfo{
			Timestamp: dp.Timestamp,
			Value:     value,
			Unit:      dp.Unit,
		})
	}

	return connectors.JSONResult(map[string]any{
		"label":      xmlResp.Result.Label,
		"datapoints": datapoints,
		"stat":       params.Stat,
		"region":     params.Region,
	})
}
