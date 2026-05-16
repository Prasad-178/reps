package ingest

import "strings"

// roughTokens estimates ~4 chars/token. Cheap and works for chunking budgets.
func roughTokens(s string) int { return (len(s) + 3) / 4 }

// chunkBySemanticBoundaries splits text on blank-line paragraphs, then packs
// paragraphs together up to maxTokens. If a single paragraph exceeds maxTokens,
// it is split on newlines, then on whitespace as a last resort.
func chunkBySemanticBoundaries(text string, maxTokens int) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	paragraphs := splitParagraphs(text)
	var chunks []string
	var cur strings.Builder
	curTok := 0
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		chunks = append(chunks, strings.TrimSpace(cur.String()))
		cur.Reset()
		curTok = 0
	}
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		ptok := roughTokens(p)
		if ptok > maxTokens {
			flush()
			for _, sub := range splitOversized(p, maxTokens) {
				chunks = append(chunks, sub)
			}
			continue
		}
		if curTok+ptok > maxTokens {
			flush()
		}
		if cur.Len() > 0 {
			cur.WriteString("\n\n")
		}
		cur.WriteString(p)
		curTok += ptok
	}
	flush()
	return chunks
}

func splitParagraphs(s string) []string {
	parts := strings.Split(s, "\n\n")
	out := parts[:0]
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			out = append(out, p)
		}
	}
	return out
}

func splitOversized(p string, maxTokens int) []string {
	lines := strings.Split(p, "\n")
	var out []string
	var cur strings.Builder
	curTok := 0
	for _, ln := range lines {
		lt := roughTokens(ln)
		if lt > maxTokens {
			if cur.Len() > 0 {
				out = append(out, strings.TrimSpace(cur.String()))
				cur.Reset()
				curTok = 0
			}
			out = append(out, splitOnWhitespace(ln, maxTokens)...)
			continue
		}
		if curTok+lt > maxTokens {
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
			curTok = 0
		}
		if cur.Len() > 0 {
			cur.WriteString("\n")
		}
		cur.WriteString(ln)
		curTok += lt
	}
	if cur.Len() > 0 {
		out = append(out, strings.TrimSpace(cur.String()))
	}
	return out
}

func splitOnWhitespace(s string, maxTokens int) []string {
	words := strings.Fields(s)
	var out []string
	var cur strings.Builder
	curTok := 0
	for _, w := range words {
		wt := roughTokens(w) + 1
		if curTok+wt > maxTokens && cur.Len() > 0 {
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
			curTok = 0
		}
		if cur.Len() > 0 {
			cur.WriteByte(' ')
		}
		cur.WriteString(w)
		curTok += wt
	}
	if cur.Len() > 0 {
		out = append(out, strings.TrimSpace(cur.String()))
	}
	return out
}
