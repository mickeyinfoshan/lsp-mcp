package lsp

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/mickeyinfoshan/lsp-mcp/pkg/types"
	protocol "go.lsp.dev/protocol"
)

// parseDefinitionResponse parses definition response
func (c *Client) parseDefinitionResponse(result json.RawMessage) (*types.FindDefinitionResponse, error) {
	response := &types.FindDefinitionResponse{}

	// First try parsing as an array
	var rawArr []json.RawMessage
	if err := json.Unmarshal(result, &rawArr); err == nil {
		if len(rawArr) == 0 {
			// Empty array is valid; return empty response
			response.Message = "definition not found."
			return response, nil
		}
		var probe map[string]interface{}
		if err := json.Unmarshal(rawArr[0], &probe); err == nil {
			if _, ok := probe["targetUri"]; ok {
				// Is LocationLink
				var locationLinks []protocol.LocationLink
				if err := json.Unmarshal(result, &locationLinks); err == nil {
					response.LocationLinks = locationLinks
					// Agent-friendly structured output
					for _, link := range locationLinks {
						file, line, char := parseFileLineChar(link.TargetURI, link.TargetSelectionRange)
						summary := formatSummary(file, line, char)
						loc := protocol.Location{URI: link.TargetURI, Range: link.TargetSelectionRange}
						response.AgentResults = append(response.AgentResults, types.AgentDefinitionResult{
							Type:      "location_link",
							File:      file,
							Line:      line,
							Character: char,
							Summary:   summary,
							Range:     &link.TargetSelectionRange,
							Location:  &loc,
						})
					}
					response.Message = formatAgentMessage(len(response.AgentResults), "definition")
					return response, nil
				}
			} else if _, ok := probe["uri"]; ok {
				// Is Location
				var locations []protocol.Location
				if err := json.Unmarshal(result, &locations); err == nil {
					response.Locations = locations
					for _, loc := range locations {
						file, line, char := parseFileLineChar(loc.URI, loc.Range)
						summary := formatSummary(file, line, char)
						response.AgentResults = append(response.AgentResults, types.AgentDefinitionResult{
							Type:      "location",
							File:      file,
							Line:      line,
							Character: char,
							Summary:   summary,
							Range:     &loc.Range,
							Location:  &loc,
						})
					}
					response.Message = formatAgentMessage(len(response.AgentResults), "definition")
					return response, nil
				}
			}
		}
	}

	// Try parsing as a single Location
	var location protocol.Location
	if err := json.Unmarshal(result, &location); err == nil && location.URI != "" {
		response.Locations = []protocol.Location{location}
		file, line, char := parseFileLineChar(location.URI, location.Range)
		summary := formatSummary(file, line, char)
		response.AgentResults = []types.AgentDefinitionResult{{
			Type:      "location",
			File:      file,
			Line:      line,
			Character: char,
			Summary:   summary,
			Range:     &location.Range,
			Location:  &location,
		}}
		response.Message = formatAgentMessage(1, "definition")
		return response, nil
	}

	// Try parsing as a single LocationLink
	var locationLink protocol.LocationLink
	if err := json.Unmarshal(result, &locationLink); err == nil && locationLink.TargetURI != "" {
		response.LocationLinks = []protocol.LocationLink{locationLink}
		file, line, char := parseFileLineChar(locationLink.TargetURI, locationLink.TargetSelectionRange)
		summary := formatSummary(file, line, char)
		loc := protocol.Location{URI: locationLink.TargetURI, Range: locationLink.TargetSelectionRange}
		response.AgentResults = []types.AgentDefinitionResult{{
			Type:      "location_link",
			File:      file,
			Line:      line,
			Character: char,
			Summary:   summary,
			Range:     &locationLink.TargetSelectionRange,
			Location:  &loc,
		}}
		response.Message = formatAgentMessage(1, "definition")
		return response, nil
	}

	// If parsing fails, check for null
	if string(result) == "null" {
		response.Message = "definition not found."
		return response, nil
	}

	return nil, fmt.Errorf("failed to parse definition response: %s", string(result))
}

// parseFileLineChar parses URI and range, returns full path, line (1-based), char (1-based)
func parseFileLineChar(uri protocol.DocumentURI, rng protocol.Range) (string, int, int) {
	file := strings.TrimPrefix(string(uri), "file://")
	line := int(rng.Start.Line) + 1
	char := int(rng.Start.Character) + 1
	return file, line, char
}

// formatSummary builds an agent-friendly summary (full path)
func formatSummary(file string, line, char int) string {
	return fmt.Sprintf("Jump to %s line %d column %d", file, line, char)
}

// formatAgentMessage supports typed labels
func formatAgentMessage(count int, typ string) string {
	if count == 0 {
		return fmt.Sprintf("No %s found.", typ)
	}
	if count == 1 {
		return fmt.Sprintf("Found 1 %s.", typ)
	}
	return fmt.Sprintf("Found %d %ss.", count, typ)
}

// parseReferencesResponse parses references response
func (c *Client) parseReferencesResponse(result json.RawMessage) (*types.FindReferencesResponse, error) {
	response := &types.FindReferencesResponse{}

	// Try parsing as Location array
	var locations []protocol.Location
	if err := json.Unmarshal(result, &locations); err == nil {
		response.Locations = locations
		// Agent-friendly structured output
		for _, loc := range locations {
			file, line, char := parseFileLineChar(loc.URI, loc.Range)
			summary := fmt.Sprintf("Referenced at %s line %d column %d", file, line, char)
			response.AgentResults = append(response.AgentResults, types.AgentReferenceResult{
				Type:      "reference",
				File:      file,
				Line:      line,
				Character: char,
				Summary:   summary,
				Range:     &loc.Range,
				Location:  &loc,
			})
		}
		response.Message = formatAgentMessage(len(response.AgentResults), "reference")
		return response, nil
	}

	// If parsing fails, check for null
	if string(result) == "null" {
		response.Message = "No references found."
		return response, nil
	}

	return nil, fmt.Errorf("failed to parse references response: %s", string(result))
}

// parseHoverResponse parses hover response
func (c *Client) parseHoverResponse(result json.RawMessage) (*types.HoverResponse, error) {
	response := &types.HoverResponse{}

	// Try parsing as Hover object
	var hover struct {
		Contents interface{}     `json:"contents"`
		Range    *protocol.Range `json:"range,omitempty"`
	}

	if err := json.Unmarshal(result, &hover); err == nil {
		var (
			rawMarkdown     string
			typeSignature   string
			importStatement string
			doc             string
			summary         string
		)
		// Process contents field
		if hover.Contents != nil {
			// 1. Parse contents as kind/value
			var value string
			switch v := hover.Contents.(type) {
			case map[string]interface{}:
				if val, ok := v["value"].(string); ok {
					value = val
				}
			case []interface{}:
				// Multiple MarkedString; concatenate
				for _, item := range v {
					if m, ok := item.(map[string]interface{}); ok {
						if val, ok := m["value"].(string); ok {
							value += val + "\n"
						}
					}
				}
			case string:
				value = v
			}

			// 2. Unescape characters
			value = htmlUnescapeString(value)
			rawMarkdown = value

			// 3. Extract code block content
			re := regexp.MustCompile("```[a-zA-Z]*\\n([\\s\\S]+?)```")
			matches := re.FindStringSubmatch(value)
			if len(matches) > 1 {
				block := matches[1]
				lines := strings.Split(block, "\n")
				if len(lines) > 0 {
					typeSignature = strings.TrimSpace(lines[0])
				}
				if len(lines) > 1 {
					for _, l := range lines[1:] {
						l = strings.TrimSpace(l)
						if strings.HasPrefix(l, "import ") {
							importStatement = l
						} else if l != "" && doc == "" {
							doc = l
						}
					}
				}
			}

			// 4. Build summary and doc
			if typeSignature != "" && importStatement != "" {
				summary = "Type definition: " + typeSignature + ", can be imported via " + importStatement + "."
			} else if typeSignature != "" {
				summary = "Type definition: " + typeSignature
			} else {
				summary = value
			}
			// Prefer doc comment; fallback to typeSignature
			if doc == "" {
				doc = typeSignature
			}

			file := ""
			line, char := 0, 0
			var rng *protocol.Range
			if hover.Range != nil {
				file = "(unknown)"
				line = int(hover.Range.Start.Line) + 1
				char = int(hover.Range.Start.Character) + 1
				rng = hover.Range
			}
			response.AgentResults = []types.AgentHoverResult{{
				Summary:         summary,
				Doc:             doc,
				Type:            "hover",
				File:            file,
				Line:            line,
				Character:       char,
				Range:           rng,
				Location:        nil,
				TypeSignature:   typeSignature,
				ImportStatement: importStatement,
				RawMarkdown:     rawMarkdown,
			}}
			response.Message = summary
			response.Contents = hover.Contents
		}
		response.Range = hover.Range
		return response, nil
	}

	// If parsing fails, check for null
	if string(result) == "null" {
		response.Message = "No hover info."
		return response, nil
	}

	return nil, fmt.Errorf("failed to parse hover response: %s", string(result))
}

// htmlUnescapeString unescapes HTML entities
func htmlUnescapeString(s string) string {
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\\u003c", "<")
	s = strings.ReplaceAll(s, "\\u003e", ">")
	return s
}

// parseCompletionResponse parses completion response
func (c *Client) parseCompletionResponse(result json.RawMessage) (*types.CompletionResponse, error) {
	response := &types.CompletionResponse{}

	// Try parsing as CompletionList
	var completionList struct {
		IsIncomplete bool                      `json:"isIncomplete"`
		Items        []protocol.CompletionItem `json:"items"`
	}

	if err := json.Unmarshal(result, &completionList); err == nil {
		response.IsIncomplete = completionList.IsIncomplete
		response.Items = completionList.Items
		// Agent-friendly structured output
		for _, item := range completionList.Items {
			summary := fmt.Sprintf("Completion item: %s %s", item.Label, item.Detail)
			var textEditPtr *protocol.TextEdit
			if item.TextEdit != nil {
				te := *item.TextEdit
				textEditPtr = &te
			}
			response.AgentResults = append(response.AgentResults, types.AgentCompletionResult{
				Type:           "completion",
				File:           "",
				Line:           0,
				Character:      0,
				Summary:        summary,
				TextEdit:       textEditPtr,
				CompletionItem: item,
			})
		}
		response.Message = formatAgentMessage(len(response.AgentResults), "completion item")
		return response, nil
	}

	// Try parsing as CompletionItem array
	var items []protocol.CompletionItem
	if err := json.Unmarshal(result, &items); err == nil {
		response.Items = items
		for _, item := range items {
			summary := fmt.Sprintf("Completion item: %s %s", item.Label, item.Detail)
			var textEditPtr *protocol.TextEdit
			if item.TextEdit != nil {
				te := *item.TextEdit
				textEditPtr = &te
			}
			response.AgentResults = append(response.AgentResults, types.AgentCompletionResult{
				Type:           "completion",
				File:           "",
				Line:           0,
				Character:      0,
				Summary:        summary,
				TextEdit:       textEditPtr,
				CompletionItem: item,
			})
		}
		response.Message = formatAgentMessage(len(response.AgentResults), "completion item")
		return response, nil
	}

	// If parsing fails, check for null
	if string(result) == "null" {
		response.Message = "No completion items."
		return response, nil
	}

	return nil, fmt.Errorf("failed to parse completion response: %s", string(result))
}
