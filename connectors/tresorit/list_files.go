package tresorit

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type listFilesAction struct {
	conn *TresoritConnector
}

type listFilesParams struct {
	Tresor  string `json:"tresor"`
	Prefix  string `json:"prefix"`
	MaxKeys int    `json:"max_keys"`
}

func (p *listFilesParams) validate() error {
	if err := validateTresorName(p.Tresor); err != nil {
		return err
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
		ETag         string `xml:"ETag"`
	} `xml:"Contents"`
	CommonPrefixes []struct {
		Prefix string `xml:"Prefix"`
	} `xml:"CommonPrefixes"`
}

type fileInfo struct {
	Key          string `json:"key"`
	LastModified string `json:"last_modified,omitempty"`
	Size         int64  `json:"size"`
	IsFolder     bool   `json:"is_folder"`
}

func (a *listFilesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidate[listFilesParams](req.Parameters)
	if err != nil {
		return nil, err
	}

	qp := url.Values{}
	qp.Set("list-type", "2")
	qp.Set("max-keys", strconv.Itoa(params.MaxKeys))
	if params.Prefix != "" {
		qp.Set("prefix", params.Prefix)
	}
	qp.Set("delimiter", "/")

	respBody, err := a.conn.do(ctx, req.Credentials, "GET", objectPath(params.Tresor, ""), qp.Encode(), nil, "")
	if err != nil {
		return nil, err
	}

	var xmlResp listBucketResult
	if err := xml.Unmarshal(respBody, &xmlResp); err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("parsing Tresorit list response: %v", err)}
	}

	files := make([]fileInfo, 0, len(xmlResp.Contents)+len(xmlResp.CommonPrefixes))
	for _, prefix := range xmlResp.CommonPrefixes {
		files = append(files, fileInfo{Key: prefix.Prefix, IsFolder: true})
	}
	for _, obj := range xmlResp.Contents {
		if obj.Key == params.Prefix && obj.Size == 0 && stringsHasSuffix(obj.Key, "/") {
			continue
		}
		files = append(files, fileInfo{
			Key:          obj.Key,
			LastModified: obj.LastModified,
			Size:         obj.Size,
			IsFolder:     stringsHasSuffix(obj.Key, "/"),
		})
	}

	return connectors.JSONResult(map[string]any{
		"tresor":       params.Tresor,
		"prefix":       params.Prefix,
		"files":        files,
		"count":        len(files),
		"is_truncated": xmlResp.IsTruncated,
	})
}
