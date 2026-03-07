package aws

import (
	"context"
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listS3ObjectsAction implements connectors.Action for aws.list_s3_objects.
type listS3ObjectsAction struct {
	conn *AWSConnector
}

type listS3ObjectsParams struct {
	Region  string `json:"region"`
	Bucket  string `json:"bucket"`
	Prefix  string `json:"prefix"`
	MaxKeys int    `json:"max_keys"`
}

func (p *listS3ObjectsParams) validate() error {
	if err := validateRegion(p.Region); err != nil {
		return err
	}
	if p.Bucket == "" {
		return &connectors.ValidationError{Message: "missing required parameter: bucket"}
	}
	if p.MaxKeys == 0 {
		p.MaxKeys = 1000
	}
	if p.MaxKeys < 1 || p.MaxKeys > 1000 {
		return &connectors.ValidationError{Message: "max_keys must be between 1 and 1000"}
	}
	return nil
}

type listBucketResult struct {
	XMLName     xml.Name `xml:"ListBucketResult"`
	Name        string   `xml:"Name"`
	Prefix      string   `xml:"Prefix"`
	IsTruncated bool     `xml:"IsTruncated"`
	Contents    []struct {
		Key          string `xml:"Key"`
		LastModified string `xml:"LastModified"`
		Size         int64  `xml:"Size"`
		StorageClass string `xml:"StorageClass"`
		ETag         string `xml:"ETag"`
	} `xml:"Contents"`
}

type s3ObjectInfo struct {
	Key          string `json:"key"`
	LastModified string `json:"last_modified"`
	Size         int64  `json:"size"`
	StorageClass string `json:"storage_class"`
}

// Execute lists objects in an S3 bucket using the S3 REST API (path-style).
func (a *listS3ObjectsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[listS3ObjectsParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	query := "list-type=2&max-keys=" + strconv.Itoa(params.MaxKeys)
	if params.Prefix != "" {
		query += "&prefix=" + params.Prefix
	}

	host := fmt.Sprintf("s3.%s.amazonaws.com", params.Region)
	path := "/" + params.Bucket

	respBody, err := a.conn.do(ctx, req.Credentials, "GET", host, path, query, nil)
	if err != nil {
		return nil, err
	}

	var xmlResp listBucketResult
	if err := xml.Unmarshal(respBody, &xmlResp); err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("parsing S3 response: %v", err)}
	}

	objects := make([]s3ObjectInfo, 0, len(xmlResp.Contents))
	for _, obj := range xmlResp.Contents {
		objects = append(objects, s3ObjectInfo{
			Key:          obj.Key,
			LastModified: obj.LastModified,
			Size:         obj.Size,
			StorageClass: obj.StorageClass,
		})
	}

	return connectors.JSONResult(map[string]any{
		"bucket":       params.Bucket,
		"prefix":       params.Prefix,
		"objects":      objects,
		"count":        len(objects),
		"is_truncated": xmlResp.IsTruncated,
	})
}
