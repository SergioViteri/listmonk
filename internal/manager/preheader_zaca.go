package manager

// ZACA: per-campaign preheader (a.k.a. preview text).
//
// The preheader is the snippet inbox lists show next to the subject line.
// Left unset, e-mail clients grab whatever text comes first in the body,
// usually the "view in browser" link (see static/email-templates/default.tpl).
// Upstream rejected adding this (knadh/listmonk#1240, backed by #924), so it
// lives here in the fork.
//
// No schema change: the value is stored under the "preheader" key of the
// existing, previously-unused campaigns.attribs JSONB column (see
// models.Campaign.Attribs), which the create/update campaign API already
// reads and writes verbatim — no backend API changes were needed for that
// part.
//
// Injection happens right here, in CampaignMessage.render() (message.go),
// because it's the single choke point every campaign render path already
// funnels through: actual sends (manager/pipe.go), test sends and both
// browser previews (cmd/campaigns.go), and the public archive
// (cmd/archive.go, cmd/public.go) all call Manager.NewCampaignMessage.
//
// Decisions worth knowing about (see ZACA-CHANGES.md for the full writeup):
//   - Plain-text campaigns are left untouched: there's no HTML body to hide
//     a preheader in.
//   - Campaigns without attribs.preheader set behave exactly as before —
//     nothing is injected, no empty hidden div.
//   - No attempt is made to detect a preheader div already hand-rolled into
//     the body (the workaround this feature replaces, e.g. campaign 7 /
//     Tsukimi FR imported from Brevo). If a campaign already has one and its
//     attribs.preheader also gets filled in, both end up in the rendered
//     e-mail. Sniffing arbitrary HTML for "is this a preheader" reliably
//     isn't worth the false positives/negatives it'd introduce; instead,
//     remove the hand-rolled div by hand when adopting the field on a given
//     campaign.
import (
	"bytes"
	"html"
	"regexp"
	"strings"

	"github.com/knadh/listmonk/models"
)

// reZacaBodyTag matches the opening <body ...> tag so the preheader element
// can be inserted right after it.
var reZacaBodyTag = regexp.MustCompile(`(?is)<body[^>]*>`)

// zacaPreheaderPad is repeated after the preheader text as invisible filler.
// Once the actual text runs out, e-mail clients pad the inbox preview
// snippet with these characters instead of pulling in the next bit of
// visible body copy — the standard alternating zero-width-non-joiner /
// non-breaking-space trick used across e-mail marketing tooling.
const zacaPreheaderPad = "&zwnj;&nbsp;"

// zacaPreheaderPadRepeat repetitions of zacaPreheaderPad are enough to cover
// the ~140-character snippet length most inbox lists show.
const zacaPreheaderPadRepeat = 100

// zacaCampaignPreheader returns the preheader text configured on a campaign
// via attribs.preheader, or "" if it's unset, blank, or not a string.
func zacaCampaignPreheader(c *models.Campaign) string {
	if c == nil || c.Attribs == nil {
		return ""
	}
	v, ok := c.Attribs["preheader"].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(v)
}

// zacaInjectPreheader inserts the campaign's preheader (if any) as a hidden
// element right after the opening <body> tag. It's a no-op for plain-text
// campaigns, campaigns with no preheader configured, and bodies with no
// recognizable <body> tag (e.g. plain-text content rendered without a
// wrapping template).
func zacaInjectPreheader(body []byte, c *models.Campaign) []byte {
	if c == nil || c.ContentType == models.CampaignContentTypePlain {
		return body
	}

	preheader := zacaCampaignPreheader(c)
	if preheader == "" {
		return body
	}

	loc := reZacaBodyTag.FindIndex(body)
	if loc == nil {
		return body
	}
	insertAt := loc[1]

	var b bytes.Buffer
	b.Grow(len(body) + 512)
	b.Write(body[:insertAt])
	b.WriteString(`<div style="display:none;font-size:1px;line-height:1px;max-height:0;max-width:0;opacity:0;overflow:hidden;mso-hide:all;">`)
	b.WriteString(html.EscapeString(preheader))
	b.WriteString(strings.Repeat(zacaPreheaderPad, zacaPreheaderPadRepeat))
	b.WriteString(`</div>`)
	b.Write(body[insertAt:])

	return b.Bytes()
}
