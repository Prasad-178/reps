package ingest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
)

// fetchPage tries plain HTTP first; if response is suspicious (tiny body, looks
// JS-shell-rendered), falls back to headless Chrome via chromedp.
func fetchPage(ctx context.Context, url string) (string, error) {
	body, err := httpGet(ctx, url)
	if err == nil && looksRendered(body) {
		return cleanHTML(body), nil
	}
	body, err = chromeGet(ctx, url)
	if err != nil {
		return "", err
	}
	return cleanHTML(body), nil
}

func httpGet(ctx context.Context, url string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(cctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; reps-cli)")
	req.Header.Set("Accept", "text/html,*/*")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 5<<20))
	return string(b), err
}

// looksRendered returns true if a fetched body contains enough text to skip the
// chromedp fallback.
func looksRendered(html string) bool {
	stripped := stripTags(html)
	return len(strings.TrimSpace(stripped)) > 500
}

func chromeGet(ctx context.Context, url string) (string, error) {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.UserAgent("Mozilla/5.0 (Macintosh; reps-cli)"),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAlloc()
	cctx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	tctx, tcancel := context.WithTimeout(cctx, 45*time.Second)
	defer tcancel()
	var html string
	err := chromedp.Run(tctx,
		chromedp.Navigate(url),
		chromedp.Sleep(2*time.Second),
		chromedp.OuterHTML("html", &html),
	)
	if err != nil {
		return "", fmt.Errorf("chromedp: %w", err)
	}
	return html, nil
}

var (
	reScript = regexp.MustCompile(`(?is)<script.*?</script>`)
	reStyle  = regexp.MustCompile(`(?is)<style.*?</style>`)
	reNav    = regexp.MustCompile(`(?is)<nav.*?</nav>`)
	reHeader = regexp.MustCompile(`(?is)<header.*?</header>`)
	reFooter = regexp.MustCompile(`(?is)<footer.*?</footer>`)
	reAside  = regexp.MustCompile(`(?is)<aside.*?</aside>`)
	reForm   = regexp.MustCompile(`(?is)<form.*?</form>`)
	reTag    = regexp.MustCompile(`<[^>]+>`)
	reWS     = regexp.MustCompile(`[ \t]+`)
	reBlanks = regexp.MustCompile(`\n{3,}`)
)

func cleanHTML(html string) string {
	html = reScript.ReplaceAllString(html, "")
	html = reStyle.ReplaceAllString(html, "")
	html = reNav.ReplaceAllString(html, "")
	html = reHeader.ReplaceAllString(html, "")
	html = reFooter.ReplaceAllString(html, "")
	html = reAside.ReplaceAllString(html, "")
	html = reForm.ReplaceAllString(html, "")
	html = strings.ReplaceAll(html, "<br>", "\n")
	html = strings.ReplaceAll(html, "<br/>", "\n")
	html = strings.ReplaceAll(html, "<br />", "\n")
	html = strings.ReplaceAll(html, "</p>", "\n\n")
	html = strings.ReplaceAll(html, "</div>", "\n")
	html = strings.ReplaceAll(html, "</li>", "\n")
	text := reTag.ReplaceAllString(html, "")
	text = unescape(text)
	text = reWS.ReplaceAllString(text, " ")
	text = reBlanks.ReplaceAllString(text, "\n\n")
	return strings.TrimSpace(text)
}

func stripTags(html string) string {
	html = reScript.ReplaceAllString(html, "")
	html = reStyle.ReplaceAllString(html, "")
	return reTag.ReplaceAllString(html, " ")
}

var entityReplacer = strings.NewReplacer(
	"&amp;", "&",
	"&lt;", "<",
	"&gt;", ">",
	"&quot;", `"`,
	"&#39;", "'",
	"&apos;", "'",
	"&nbsp;", " ",
	"&mdash;", "—",
	"&ndash;", "–",
	"&hellip;", "…",
	"&rsquo;", "'",
	"&lsquo;", "'",
	"&ldquo;", `"`,
	"&rdquo;", `"`,
)

func unescape(s string) string { return entityReplacer.Replace(s) }
