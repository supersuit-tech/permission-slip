package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const titlePlaceholderObjectID = "ps_title_placeholder"

// addSlideAction implements connectors.Action for google.add_slide.
// It adds a new slide to an existing presentation via batchUpdate.
type addSlideAction struct {
	conn *GoogleConnector
}

// addSlideParams is the user-facing parameter schema.
type addSlideParams struct {
	PresentationID string `json:"presentation_id"`
	Layout         string `json:"layout"`
	InsertionIndex *int   `json:"insertion_index,omitempty"`
	Title          string `json:"title,omitempty"`
}

// validSlideLayouts lists the allowed predefined layout values for the
// Google Slides API CreateSlideRequest. See:
// https://developers.google.com/slides/api/reference/rest/v1/presentations/request#PredefinedLayout
var validSlideLayouts = map[string]bool{
	"BLANK":                         true,
	"TITLE":                         true,
	"TITLE_AND_BODY":                true,
	"SECTION_HEADER":                true,
	"ONE_COLUMN_TEXT":               true,
	"MAIN_POINT":                    true,
	"BIG_NUMBER":                    true,
	"CAPTION_ONLY":                  true,
	"TITLE_ONLY":                    true,
	"SECTION_TITLE_AND_DESCRIPTION": true,
	"TITLE_AND_TWO_COLUMNS":         true,
}

func (p *addSlideParams) validate() error {
	if p.PresentationID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: presentation_id"}
	}
	if p.Layout != "" && !validSlideLayouts[p.Layout] {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid layout %q: see Google Slides PredefinedLayout reference for valid values", p.Layout),
		}
	}
	if p.InsertionIndex != nil && *p.InsertionIndex < 0 {
		return &connectors.ValidationError{Message: "insertion_index must be non-negative"}
	}
	return nil
}

func (p *addSlideParams) normalize() {
	if p.Layout == "" {
		p.Layout = "BLANK"
	}
}

// slidesBatchUpdateRequest is the Google Slides API batchUpdate request.
type slidesBatchUpdateRequest struct {
	Requests []slidesBatchRequest `json:"requests"`
}

type slidesBatchRequest struct {
	CreateSlide *createSlideRequest  `json:"createSlide,omitempty"`
	InsertText  *slidesInsertTextReq `json:"insertText,omitempty"`
}

type createSlideRequest struct {
	SlideLayoutReference  *slideLayoutReference       `json:"slideLayoutReference,omitempty"`
	InsertionIndex        *int                        `json:"insertionIndex,omitempty"`
	PlaceholderIdMappings []slidePlaceholderIDMapping `json:"placeholderIdMappings,omitempty"`
}

type slideLayoutReference struct {
	PredefinedLayout string `json:"predefinedLayout"`
}

type slidePlaceholderIDMapping struct {
	LayoutPlaceholder slidePlaceholderRef `json:"layoutPlaceholder"`
	ObjectId          string              `json:"objectId"`
}

type slidePlaceholderRef struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

type slidesInsertTextReq struct {
	ObjectId       string `json:"objectId"`
	InsertionIndex int    `json:"insertionIndex"`
	Text           string `json:"text"`
}

// slidesBatchUpdateResponse is the Google Slides API batchUpdate response.
type slidesBatchUpdateResponse struct {
	Replies []slidesBatchReply `json:"replies"`
}

type slidesBatchReply struct {
	CreateSlide *createSlideReply `json:"createSlide,omitempty"`
}

type createSlideReply struct {
	ObjectID string `json:"objectId"`
}

// Execute adds a new slide to a Google Slides presentation and returns its ID.
func (a *addSlideAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addSlideParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}
	params.normalize()

	createReq := &createSlideRequest{
		SlideLayoutReference: &slideLayoutReference{
			PredefinedLayout: params.Layout,
		},
	}
	if params.InsertionIndex != nil {
		createReq.InsertionIndex = params.InsertionIndex
	}

	requests := []slidesBatchRequest{{CreateSlide: createReq}}

	if params.Title != "" {
		createReq.PlaceholderIdMappings = []slidePlaceholderIDMapping{
			{
				LayoutPlaceholder: slidePlaceholderRef{Type: "TITLE", Index: 0},
				ObjectId:          titlePlaceholderObjectID,
			},
		}
		requests = append(requests, slidesBatchRequest{
			InsertText: &slidesInsertTextReq{
				ObjectId: titlePlaceholderObjectID,
				Text:     params.Title,
			},
		})
	}

	body := slidesBatchUpdateRequest{
		Requests: requests,
	}

	var resp slidesBatchUpdateResponse
	batchURL := a.conn.slidesBaseURL + "/v1/presentations/" + url.PathEscape(params.PresentationID) + ":batchUpdate"
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, batchURL, body, &resp); err != nil {
		return nil, err
	}

	slideID := ""
	if len(resp.Replies) > 0 && resp.Replies[0].CreateSlide != nil {
		slideID = resp.Replies[0].CreateSlide.ObjectID
	}

	return connectors.JSONResult(map[string]string{
		"slide_id":        slideID,
		"presentation_id": params.PresentationID,
	})
}
