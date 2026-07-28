package manager

import (
	"strings"
	"testing"

	"github.com/knadh/listmonk/models"
)

func TestZacaInjectPreheader(t *testing.T) {
	htmlBody := []byte(`<!DOCTYPE html><html><head></head><body style="margin:0"><div>hello</div></body></html>`)

	cases := []struct {
		name        string
		contentType string
		attribs     models.JSON
		body        []byte
		wantChanged bool
	}{
		{
			name:        "no preheader configured leaves body untouched",
			contentType: models.CampaignContentTypeHTML,
			attribs:     models.JSON{},
			body:        htmlBody,
			wantChanged: false,
		},
		{
			name:        "nil attribs leaves body untouched",
			contentType: models.CampaignContentTypeHTML,
			attribs:     nil,
			body:        htmlBody,
			wantChanged: false,
		},
		{
			name:        "plain text campaigns are never touched, even with a preheader set",
			contentType: models.CampaignContentTypePlain,
			attribs:     models.JSON{"preheader": "Should not appear"},
			body:        []byte("plain text body, no html at all"),
			wantChanged: false,
		},
		{
			name:        "blank preheader (whitespace only) is treated as unset",
			contentType: models.CampaignContentTypeHTML,
			attribs:     models.JSON{"preheader": "   "},
			body:        htmlBody,
			wantChanged: false,
		},
		{
			name:        "no <body> tag to inject into is a no-op",
			contentType: models.CampaignContentTypeHTML,
			attribs:     models.JSON{"preheader": "Hello there"},
			body:        []byte(`<div>content with no body tag</div>`),
			wantChanged: false,
		},
		{
			name:        "html campaign with preheader gets it injected right after <body>",
			contentType: models.CampaignContentTypeHTML,
			attribs:     models.JSON{"preheader": "Come see our new arrivals"},
			body:        htmlBody,
			wantChanged: true,
		},
		{
			name:        "richtext, markdown and visual all apply the same as html",
			contentType: models.CampaignContentTypeVisual,
			attribs:     models.JSON{"preheader": "Visual campaign preview text"},
			body:        htmlBody,
			wantChanged: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := &models.Campaign{ContentType: tc.contentType, Attribs: tc.attribs}
			out := zacaInjectPreheader(tc.body, c)

			changed := string(out) != string(tc.body)
			if changed != tc.wantChanged {
				t.Fatalf("changed = %v, want %v (out: %s)", changed, tc.wantChanged, out)
			}
		})
	}
}

func TestZacaInjectPreheaderPlacementAndEscaping(t *testing.T) {
	body := []byte(`<html><body class="mail" style="margin:0"><div>the actual content</div></body></html>`)
	c := &models.Campaign{
		ContentType: models.CampaignContentTypeHTML,
		Attribs:     models.JSON{"preheader": `<script>alert(1)</script> & "quotes" 'n stuff`},
	}

	out := string(zacaInjectPreheader(body, c))

	bodyTagEnd := strings.Index(out, `<body class="mail" style="margin:0">`) + len(`<body class="mail" style="margin:0">`)
	divStart := strings.Index(out, "<div>the actual content</div>")
	if bodyTagEnd <= 0 || divStart <= 0 {
		t.Fatalf("could not locate anchors in output: %s", out)
	}
	if bodyTagEnd != strings.Index(out, "<div style=\"display:none") {
		t.Fatalf("preheader element wasn't inserted immediately after <body>: %s", out)
	}
	if strings.Contains(out, "<script>alert(1)</script>") {
		t.Fatalf("preheader text was not escaped, raw markup leaked into output: %s", out)
	}
	if !strings.Contains(out, "&lt;script&gt;") {
		t.Fatalf("expected escaped script tag in output: %s", out)
	}
	if !strings.Contains(out, "&#34;quotes&#34;") && !strings.Contains(out, "&quot;quotes&quot;") {
		t.Fatalf("expected escaped double quotes in output: %s", out)
	}
	if !strings.Contains(out, "&#39;n stuff") {
		t.Fatalf("expected escaped single quote in output: %s", out)
	}
	// Original body content must still be present, untouched.
	if !strings.Contains(out, "<div>the actual content</div>") {
		t.Fatalf("original body content was lost: %s", out)
	}
}

func TestZacaCampaignPreheader(t *testing.T) {
	if got := zacaCampaignPreheader(nil); got != "" {
		t.Fatalf("nil campaign: got %q, want empty", got)
	}
	if got := zacaCampaignPreheader(&models.Campaign{}); got != "" {
		t.Fatalf("nil attribs: got %q, want empty", got)
	}
	if got := zacaCampaignPreheader(&models.Campaign{Attribs: models.JSON{"preheader": 42}}); got != "" {
		t.Fatalf("non-string preheader: got %q, want empty", got)
	}
	if got := zacaCampaignPreheader(&models.Campaign{Attribs: models.JSON{"preheader": "  Hi there  "}}); got != "Hi there" {
		t.Fatalf("got %q, want trimmed value", got)
	}
}
