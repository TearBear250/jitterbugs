package report

import (
	"html/template"
	"os"
	"time"

	"github.com/backbiten/jitterbugs/internal/core"
)

const multiHTMLTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>QAQC Multi-Repo Audit Report</title>
  <style>
    body{font-family:system-ui,sans-serif;max-width:1080px;margin:2rem auto;padding:0 1rem;color:#333}
    h1{margin-bottom:.25rem}
    .meta{color:#666;font-size:.9rem;margin-bottom:1.5rem}
    .badge{display:inline-block;padding:.2em .6em;border-radius:4px;font-weight:700;text-transform:uppercase;font-size:.8rem}
    .pass{background:#d4edda;color:#155724}
    .warning{background:#fff3cd;color:#856404}
    .fail{background:#f8d7da;color:#721c24}
    h2{margin-top:2rem;border-bottom:2px solid #dee2e6;padding-bottom:.3rem}
    table{width:100%;border-collapse:collapse;margin-top:.75rem}
    th,td{border:1px solid #dee2e6;padding:.5rem .75rem;text-align:left;vertical-align:top}
    th{background:#f8f9fa;font-weight:600}
    tr:nth-child(even){background:#fafafa}
    code{font-size:.85em;background:#f4f4f4;padding:.1em .3em;border-radius:3px}
    .findings-heading{margin-top:1.5rem;font-size:1rem}
  </style>
</head>
<body>
<h1>QAQC Multi-Repo Audit Report</h1>
<div class="meta">
  <strong>Owner:</strong> {{.Owner}}<br>
  <strong>Scanned at:</strong> {{.Timestamp.UTC.Format "2006-01-02 15:04:05"}} UTC<br>
  <strong>Repositories scanned:</strong> {{len .Repos}}<br>
  <strong>Overall status:</strong> <span class="badge {{.OverallStatus}}">{{.OverallStatus}}</span>
</div>

{{range .Repos}}
<h2>{{.RepoPath}} <span class="badge {{.OverallStatus}}">{{.OverallStatus}}</span></h2>
<table>
  <tr><th>Check</th><th>Status</th><th>Message</th></tr>
  {{range .Results}}
  <tr>
    <td>{{.Name}}</td>
    <td><span class="badge {{.Status}}">{{.Status}}</span></td>
    <td>{{.Message}}</td>
  </tr>
  {{end}}
</table>
{{$findings := repoFindings .}}{{if $findings}}
<h3 class="findings-heading">Findings</h3>
<table>
  <tr><th>Location</th><th>Pattern</th><th>Detail</th></tr>
  {{range $findings}}
  <tr>
    <td>{{.File}}</td>
    <td>{{.Pattern}}</td>
    <td><code>{{.Match}}</code></td>
  </tr>
  {{end}}
</table>
{{end}}
{{end}}
</body>
</html>`

// multiReportView is the template data container for multi-repo reports.
type multiReportView struct {
	Owner         string
	Timestamp     time.Time
	Repos         []*core.Report
	OverallStatus core.Severity
}

// WriteMultiHTML renders an aggregated multi-repository audit report as HTML.
func WriteMultiHTML(owner string, timestamp time.Time, repos []*core.Report, path string) error {
	overall := core.SeverityPass
	for _, r := range repos {
		switch r.OverallStatus {
		case core.SeverityFail:
			overall = core.SeverityFail
		case core.SeverityWarning:
			if overall != core.SeverityFail {
				overall = core.SeverityWarning
			}
		}
	}

	funcMap := template.FuncMap{
		"repoFindings": func(rpt *core.Report) []core.Finding {
			var all []core.Finding
			for _, result := range rpt.Results {
				all = append(all, result.Findings...)
			}
			return all
		},
	}

	tmpl, err := template.New("multireport").Funcs(funcMap).Parse(multiHTMLTmpl)
	if err != nil {
		return err
	}

	view := &multiReportView{
		Owner:         owner,
		Timestamp:     timestamp,
		Repos:         repos,
		OverallStatus: overall,
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, view)
}
