package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

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

	// When a title is provided, map the layout's TITLE placeholder to a known
	// objectId so we can target it with a follow-up InsertText. The ID must be
	// unique within the presentation, so we derive it from a nanosecond timestamp.
	var titleID string
	if params.Title != "" {
		titleID = fmt.Sprintf("ps_title_%x", time.Now().UnixNano())
		createReq.PlaceholderIdMappings = []slidePlaceholderIDMapping{
			{
				LayoutPlaceholder: slidePlaceholderRef{Type: "TITLE", Index: 0},
				ObjectId:          titleID,
			},
		}
	}

	batchURL := a.conn.slidesBaseURL + "/v1/presentations/" + url.PathEscape(params.PresentationID) + ":batchUpdate"

	// First batchUpdate: create the slide (and mint the title placeholder ID via
	// placeholderIdMappings). We deliberately do NOT combine CreateSlide and
	// InsertText into a single batchUpdate: the Slides API validates the full
	// request batch before applying any of it, and InsertText targeting a
	// placeholder that only exists after CreateSlide runs can be silently
	// dropped. Google's own samples use two separate batchUpdate calls for this
	// pattern — see
	// https://developers.google.com/slides/api/samples/slide
	createBody := slidesBatchUpdateRequest{
		Requests: []slidesBatchRequest{{CreateSlide: createReq}},
	}
	var createResp slidesBatchUpdateResponse
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, batchURL, createBody, &createResp); err != nil {
		return nil, err
	}

	slideID := ""
	if len(createResp.Replies) > 0 && createResp.Replies[0].CreateSlide != nil {
		slideID = createResp.Replies[0].CreateSlide.ObjectID
	}

	// Second batchUpdate: populate the title placeholder with the user's text.
	if titleID != "" {
		textBody := slidesBatchUpdateRequest{
			Requests: []slidesBatchRequest{{
				InsertText: &slidesInsertTextReq{
					ObjectId: titleID,
					Text:     params.Title,
				},
			}},
		}
		if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, batchURL, textBody, nil); err != nil {
			return nil, err
		}
	}

	return connectors.JSONResult(map[string]string{
		"slide_id":        slideID,
		"presentation_id": params.PresentationID,
	})
}
