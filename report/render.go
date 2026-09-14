package report

import (
	"encoding/json"
	"fmt"
	"html/template"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const linesPerPage = 52
const maxColumns = 108

func WriteBundle(report Report, directory string) ([]string, error) {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, err
	}
	files := []string{}
	jsonPath := filepath.Join(directory, "report.json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(jsonPath, data, 0o644); err != nil {
		return nil, err
	}
	files = append(files, jsonPath)

	htmlPath := filepath.Join(directory, "report.html")
	if err := writeHTML(report, htmlPath); err != nil {
		return nil, err
	}
	files = append(files, htmlPath)

	pngs, err := writePNGs(report, directory)
	if err != nil {
		return nil, err
	}
	return append(files, pngs...), nil
}

var htmlTemplate = template.Must(template.New("report").Funcs(template.FuncMap{
	"duration": func(d time.Duration) string { return d.Round(time.Millisecond).String() },
	"plain":    StripANSI,
	"status": func(code int) string {
		if code == 0 {
			return "passed"
		}
		return "failed"
	},
}).Parse(`<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Title}}</title><style>
:root{color-scheme:dark;--bg:#090b10;--card:#11151d;--line:#242b38;--text:#e8edf5;--muted:#8d99aa;--green:#49d17d;--red:#ff6b6b;--accent:#8ba4ff}*{box-sizing:border-box}body{margin:0;background:radial-gradient(circle at 20% 0,#172036 0,transparent 35%),var(--bg);color:var(--text);font:14px/1.55 ui-monospace,SFMono-Regular,Menlo,monospace}main{max-width:1120px;margin:auto;padding:48px 24px 80px}header{display:flex;justify-content:space-between;gap:24px;align-items:end;margin-bottom:28px}h1{font:700 38px/1.1 system-ui;margin:0 0 9px}.eyebrow{color:var(--accent);font-weight:700;letter-spacing:.12em;text-transform:uppercase;font-size:11px}.muted{color:var(--muted)}.summary{display:flex;gap:10px;flex-wrap:wrap}.pill{border:1px solid var(--line);border-radius:999px;padding:7px 11px;background:#0c1017}.command{background:rgba(17,21,29,.92);border:1px solid var(--line);border-radius:15px;margin:16px 0;overflow:hidden}.command-head{padding:15px 18px;display:flex;gap:14px;align-items:center;border-bottom:1px solid var(--line)}.dot{width:9px;height:9px;border-radius:50%;background:var(--green);flex:none}.failed .dot{background:var(--red)}code{overflow-wrap:anywhere;color:#f0f3fa}.meta{margin-left:auto;color:var(--muted);white-space:nowrap}pre{margin:0;padding:18px;overflow:auto;max-height:70vh;white-space:pre-wrap;word-break:break-word;background:#0c0f15;color:#cdd6e5}.empty{color:var(--muted);font-style:italic}.redacted{color:#ffcc66}footer{margin-top:30px;color:var(--muted);font-size:12px}@media print{body{background:white;color:#111}.command{break-inside:avoid;border-color:#ddd}pre{max-height:none;background:#f7f7f7;color:#111}.muted,.meta,footer{color:#666}}</style></head><body><main><header><div><div class="eyebrow">Proofshot report</div><h1>{{.Title}}</h1><div class="muted">Captured {{.CreatedAt.Format "02 Jan 2006 · 15:04 MST"}}</div></div><div class="summary"><span class="pill">{{len .Commands}} commands</span><span class="pill">{{.Failed}} failed</span><span class="pill">{{duration .Duration}}</span></div></header>{{range .Commands}}<section class="command {{status .ExitCode}}"><div class="command-head"><span class="dot"></span><code>$ {{.Command}}</code><span class="meta">exit {{.ExitCode}} · {{duration .Duration}}{{if .Redactions}} · {{.Redactions}} redacted{{end}}</span></div><pre>{{if .Output}}{{plain .Output}}{{else}}<span class="empty">No output</span>{{end}}</pre></section>{{end}}<footer>Generated locally by Proofshot · Searchable, portable, and secret-aware.</footer></main></body></html>`))

func writeHTML(r Report, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return htmlTemplate.Execute(f, r)
}

type renderedLine struct {
	text string
	kind int
}

func reportLines(r Report) []renderedLine {
	lines := []renderedLine{{"PROOFSHOT  " + r.Title, 0}, {fmt.Sprintf("%d commands  ·  %d failed  ·  %s", len(r.Commands), r.Failed, r.Duration.Round(time.Millisecond)), 1}, {"", 1}}
	for _, command := range r.Commands {
		kind := 2
		if command.ExitCode != 0 {
			kind = 3
		}
		lines = append(lines, renderedLine{fmt.Sprintf("$ %s", command.Command), kind}, renderedLine{fmt.Sprintf("exit %d  ·  %s  ·  %d redacted", command.ExitCode, command.Duration.Round(time.Millisecond), command.Redactions), 1})
		for _, line := range strings.Split(StripANSI(command.Output), "\n") {
			wrapped := wrap(line, maxColumns)
			for _, part := range wrapped {
				lines = append(lines, renderedLine{part, 4})
			}
		}
		lines = append(lines, renderedLine{"", 1})
	}
	return lines
}

func wrap(s string, width int) []string {
	if s == "" {
		return []string{""}
	}
	runes := []rune(strings.ReplaceAll(s, "\t", "    "))
	parts := []string{}
	for len(runes) > width {
		parts = append(parts, string(runes[:width]))
		runes = runes[width:]
	}
	return append(parts, string(runes))
}

func writePNGs(r Report, directory string) ([]string, error) {
	lines := reportLines(r)
	pages := (len(lines) + linesPerPage - 1) / linesPerPage
	paths := make([]string, 0, pages)
	for page := 0; page < pages; page++ {
		end := (page + 1) * linesPerPage
		if end > len(lines) {
			end = len(lines)
		}
		path := filepath.Join(directory, fmt.Sprintf("report-%02d.png", page+1))
		if err := writePNG(lines[page*linesPerPage:end], page+1, pages, path); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func writePNG(lines []renderedLine, page, pages int, path string) error {
	const width, height, pad, lineHeight = 1280, 1180, 56, 20
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{9, 11, 16, 255}}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(26, 26, width-26, height-26), &image.Uniform{color.RGBA{17, 21, 29, 255}}, image.Point{}, draw.Src)
	parsedFont, err := opentype.Parse(gomono.TTF)
	if err != nil {
		return err
	}
	face, err := opentype.NewFace(parsedFont, &opentype.FaceOptions{Size: 16, DPI: 96, Hinting: font.HintingFull})
	if err != nil {
		return err
	}
	colors := []color.RGBA{{139, 164, 255, 255}, {141, 153, 170, 255}, {73, 209, 125, 255}, {255, 107, 107, 255}, {205, 214, 229, 255}}
	for index, line := range lines {
		text := line.text
		if utf8.RuneCountInString(text) > maxColumns {
			text = string([]rune(text)[:maxColumns])
		}
		d := font.Drawer{Dst: img, Src: image.NewUniform(colors[line.kind]), Face: face, Dot: fixed.P(pad, 62+index*lineHeight)}
		d.DrawString(text)
	}
	footer := font.Drawer{Dst: img, Src: image.NewUniform(colors[1]), Face: face, Dot: fixed.P(width-150, height-42)}
	footer.DrawString(fmt.Sprintf("page %d / %d", page, pages))
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}
