package instacart

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	maxLineItems      = 200
	maxInstructions   = 50
	maxInstructionLen = 2000
	maxTitleLen       = 512
	maxImageURLLen    = 2048
)

// createProductsLinkAction implements connectors.Action for instacart.create_products_link.
type createProductsLinkAction struct {
	conn *InstacartConnector
}

type createProductsLinkParams struct {
	Title                    *string           `json:"title,omitempty"`
	ImageURL                 *string           `json:"image_url,omitempty"`
	LinkType                 string            `json:"link_type,omitempty"`
	ExpiresIn                *int              `json:"expires_in,omitempty"`
	Instructions             []string          `json:"instructions,omitempty"`
	LineItems                []json.RawMessage `json:"line_items"`
	LandingPageConfiguration json.RawMessage   `json:"landing_page_configuration,omitempty"`
}

func (p *createProductsLinkParams) validate() error {
	if p.Title != nil {
		if len(strings.TrimSpace(*p.Title)) == 0 {
			return &connectors.ValidationError{Message: "title cannot be empty or whitespace-only when provided"}
		}
		if connectors.RuneLen(*p.Title) > maxTitleLen {
			return &connectors.ValidationError{Message: fmt.Sprintf("title exceeds maximum length (%d characters)", maxTitleLen)}
		}
	}
	if p.ImageURL != nil {
		if strings.TrimSpace(*p.ImageURL) == "" {
			return &connectors.ValidationError{Message: "image_url cannot be empty or whitespace-only when provided"}
		}
		if connectors.RuneLen(*p.ImageURL) > maxImageURLLen {
			return &connectors.ValidationError{Message: fmt.Sprintf("image_url exceeds maximum length (%d characters)", maxImageURLLen)}
		}
	}

	if len(p.LineItems) == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: line_items (non-empty array)"}
	}
	if len(p.LineItems) > maxLineItems {
		return &connectors.ValidationError{Message: fmt.Sprintf("line_items exceeds maximum of %d items", maxLineItems)}
	}
	if p.LinkType != "" && p.LinkType != "shopping_list" && p.LinkType != "recipe" {
		return &connectors.ValidationError{Message: "link_type must be shopping_list or recipe when provided"}
	}
	if p.ExpiresIn != nil {
		if *p.ExpiresIn < 1 || *p.ExpiresIn > 365 {
			return &connectors.ValidationError{Message: "expires_in must be between 1 and 365 when provided"}
		}
	}
	if len(p.Instructions) > maxInstructions {
		return &connectors.ValidationError{Message: fmt.Sprintf("instructions exceeds maximum of %d entries", maxInstructions)}
	}
	for i, line := range p.Instructions {
		if connectors.RuneLen(line) > maxInstructionLen {
			return &connectors.ValidationError{Message: fmt.Sprintf("instructions[%d] exceeds maximum length", i)}
		}
	}

	for i, raw := range p.LineItems {
		var item struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(raw, &item); err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("line_items[%d]: invalid JSON object", i)}
		}
		if item.Name == "" {
			return &connectors.ValidationError{Message: fmt.Sprintf("line_items[%d]: name is required", i)}
		}
		if connectors.RuneLen(item.Name) > maxLineItemNameChars {
			return &connectors.ValidationError{Message: fmt.Sprintf("line_items[%d]: name exceeds maximum length (%d characters)", i, maxLineItemNameChars)}
		}
	}

	if len(p.LandingPageConfiguration) > 0 && !json.Valid(p.LandingPageConfiguration) {
		return &connectors.ValidationError{Message: "landing_page_configuration must be valid JSON"}
	}
	return nil
}

func (a *createProductsLinkAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	params, err := parseAndValidateProductsLinkParams(req.Parameters)
	if err != nil {
		return nil, err
	}

	body := map[string]any{"line_items": params.LineItems}
	if params.Title != nil {
		body["title"] = *params.Title
	}
	if params.ImageURL != nil {
		body["image_url"] = *params.ImageURL
	}
	if params.LinkType != "" {
		body["link_type"] = params.LinkType
	}
	if params.ExpiresIn != nil {
		body["expires_in"] = *params.ExpiresIn
	}
	if len(params.Instructions) > 0 {
		body["instructions"] = params.Instructions
	}
	if len(params.LandingPageConfiguration) > 0 {
		var landing any
		if err := json.Unmarshal(params.LandingPageConfiguration, &landing); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("landing_page_configuration: %v", err)}
		}
		body["landing_page_configuration"] = landing
	}

	var apiResp struct {
		ProductsLinkURL string `json:"products_link_url"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/idp/v1/products/products_link", body, &apiResp); err != nil {
		return nil, err
	}

	if strings.TrimSpace(apiResp.ProductsLinkURL) == "" {
		return nil, &connectors.ExternalError{Message: "Instacart returned an empty products_link_url"}
	}

	return connectors.JSONResult(apiResp)
}
