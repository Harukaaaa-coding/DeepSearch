package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
)

type docxLine struct {
	Text  string
	Style string
}

func buildReportDOCX(result ResearchResult) ([]byte, error) {
	if strings.TrimSpace(result.Question) == "" || strings.TrimSpace(result.Report.Analysis) == "" {
		return nil, fmt.Errorf("研究报告内容不完整，无法导出 Word 文档")
	}

	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	files := map[string]string{
		"[Content_Types].xml":          docxContentTypes(),
		"_rels/.rels":                  docxRootRelationships(),
		"docProps/core.xml":            docxCoreProperties(result),
		"docProps/app.xml":             docxAppProperties(),
		"word/styles.xml":              docxStyles(),
		"word/document.xml":            docxDocument(result),
		"word/_rels/document.xml.rels": docxDocumentRelationships(),
	}
	for name, content := range files {
		entry, err := archive.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func docxDocument(result ResearchResult) string {
	lines := []docxLine{
		{Text: "深度研究报告", Style: "Title"},
		{Text: "研究问题：" + cleanDOCXText(result.Question), Style: "Subtitle"},
	}
	if !result.CompletedAt.IsZero() {
		lines = append(lines, docxLine{Text: "完成时间：" + result.CompletedAt.Local().Format("2006-01-02 15:04"), Style: "Metadata"})
	}
	lines = append(lines,
		docxLine{Text: "研究结论", Style: "Heading1"},
	)
	lines = append(lines, parseReportMarkdown(result.Report.Analysis)...)
	lines = append(lines,
		docxLine{Text: "研究方法", Style: "Heading1"},
		docxLine{Text: cleanDOCXText(result.Report.Method), Style: "Body"},
		docxLine{Text: "证据清单", Style: "Heading1"},
	)
	for _, evidence := range result.Evidence {
		line := fmt.Sprintf("[%s] %s（%s置信度，%d/100）：%s", evidence.ID, evidence.SourceTitle, evidence.Confidence, evidence.Score, evidence.Quote)
		lines = append(lines, docxLine{Text: cleanDOCXText(line), Style: "Body"})
		if strings.TrimSpace(evidence.URL) != "" {
			lines = append(lines, docxLine{Text: "来源：" + evidence.URL, Style: "Source"})
		}
	}
	if len(result.Report.Conflicts) > 0 {
		lines = append(lines, docxLine{Text: "冲突与待核验点", Style: "Heading1"})
		for _, conflict := range result.Report.Conflicts {
			lines = append(lines, docxLine{Text: "• " + cleanDOCXText(conflict), Style: "Bullet"})
		}
	}
	if len(result.Report.Limitations) > 0 {
		lines = append(lines, docxLine{Text: "研究限制", Style: "Heading1"})
		for _, limitation := range result.Report.Limitations {
			lines = append(lines, docxLine{Text: "• " + cleanDOCXText(limitation), Style: "Bullet"})
		}
	}

	var body strings.Builder
	body.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	body.WriteString(`<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`)
	for _, line := range lines {
		if strings.TrimSpace(line.Text) != "" {
			body.WriteString(docxParagraph(line))
		}
	}
	body.WriteString(`<w:sectPr><w:pgSz w:w="12240" w:h="15840"/><w:pgMar w:top="1440" w:right="1440" w:bottom="1440" w:left="1440" w:header="720" w:footer="720" w:gutter="0"/></w:sectPr></w:body></w:document>`)
	return body.String()
}

func parseReportMarkdown(markdown string) []docxLine {
	markdown = strings.ReplaceAll(markdown, "\r", "")
	markdown = strings.NewReplacer("\\###", "###", "\\##", "##", "\\#", "#", "\\-", "-", "\\*", "*", "\\`", "`").Replace(markdown)
	markdown = splitInlineHeadings(markdown)
	lines := make([]docxLine, 0, 16)
	for _, raw := range strings.Split(markdown, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.Trim(line, "-") == "" {
			continue
		}
		level := 0
		for level < len(line) && line[level] == '#' {
			level++
		}
		if level > 0 && level < len(line) && line[level] == ' ' {
			style := "Heading1"
			if level >= 3 {
				style = "Heading2"
			}
			lines = append(lines, docxLine{Text: cleanDOCXText(strings.TrimSpace(line[level:])), Style: style})
			continue
		}
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			lines = append(lines, docxLine{Text: "• " + cleanDOCXText(strings.TrimSpace(line[2:])), Style: "Bullet"})
			continue
		}
		lines = append(lines, docxLine{Text: cleanDOCXText(line), Style: "Body"})
	}
	return lines
}

func splitInlineHeadings(text string) string {
	for _, marker := range []string{" ### ", " ## "} {
		text = strings.ReplaceAll(text, marker, "\n"+strings.TrimSpace(marker)+"\n")
	}
	return text
}

func cleanDOCXText(text string) string {
	text = strings.NewReplacer("**", "", "`", "", "\t", " ").Replace(text)
	return strings.TrimSpace(text)
}

func docxParagraph(line docxLine) string {
	style, spacing, bold, size, color, indent := "Normal", `w:after="120"`, false, "22", "000000", ""
	switch line.Style {
	case "Title":
		style, spacing, bold, size = "Title", `w:after="260"`, true, "34"
	case "Subtitle":
		style, spacing, size, color = "Subtitle", `w:after="100"`, "22", "555555"
	case "Metadata":
		style, spacing, size, color = "Metadata", `w:after="260"`, "18", "666666"
	case "Heading1":
		style, spacing, bold, size = "Heading1", `w:before="260" w:after="100"`, true, "28"
	case "Heading2":
		style, spacing, bold, size = "Heading2", `w:before="180" w:after="70"`, true, "24"
	case "Source":
		style, spacing, size, color = "Source", `w:after="100"`, "18", "666666"
	case "Bullet":
		style, spacing, indent = "Bullet", `w:after="80"`, ` w:left="360" w:hanging="180"`
	}
	rPr := fmt.Sprintf(`<w:rPr><w:rFonts w:ascii="Aptos" w:hAnsi="Aptos" w:eastAsia="Microsoft YaHei"/><w:color w:val="%s"/><w:sz w:val="%s"/><w:szCs w:val="%s"/>`, color, size, size)
	if bold {
		rPr += `<w:b/>`
	}
	rPr += `</w:rPr>`
	return fmt.Sprintf(`<w:p><w:pPr><w:pStyle w:val="%s"/><w:spacing %s/><w:ind%s/></w:pPr><w:r>%s<w:t xml:space="preserve">%s</w:t></w:r></w:p>`, style, spacing, indent, rPr, docxEscape(line.Text))
}

func docxEscape(text string) string {
	var output bytes.Buffer
	_ = xml.EscapeText(&output, []byte(text))
	return output.String()
}

func docxContentTypes() string {
	return `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/><Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/><Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/><Override PartName="/docProps/app.xml" ContentType="application/vnd.openxmlformats-officedocument.extended-properties+xml"/></Types>`
}

func docxRootRelationships() string {
	return `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/extended-properties" Target="docProps/app.xml"/></Relationships>`
}

func docxDocumentRelationships() string {
	return `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`
}

func docxCoreProperties(result ResearchResult) string {
	title := docxEscape("深度研究报告 " + cleanDOCXText(result.Question))
	return `<?xml version="1.0" encoding="UTF-8"?><cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><dc:title>` + title + `</dc:title><dc:creator>DeepSearch</dc:creator><cp:lastModifiedBy>DeepSearch</cp:lastModifiedBy></cp:coreProperties>`
}

func docxAppProperties() string {
	return `<?xml version="1.0" encoding="UTF-8"?><Properties xmlns="http://schemas.openxmlformats.org/officeDocument/2006/extended-properties"><Application>DeepSearch</Application></Properties>`
}

func docxStyles() string {
	return `<?xml version="1.0" encoding="UTF-8"?><w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:docDefaults><w:rPrDefault><w:rPr><w:rFonts w:ascii="Aptos" w:hAnsi="Aptos" w:eastAsia="Microsoft YaHei"/><w:sz w:val="22"/><w:szCs w:val="22"/></w:rPr></w:rPrDefault></w:docDefaults><w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style><w:style w:type="paragraph" w:styleId="Title"><w:name w:val="Title"/></w:style><w:style w:type="paragraph" w:styleId="Subtitle"><w:name w:val="Subtitle"/></w:style><w:style w:type="paragraph" w:styleId="Metadata"><w:name w:val="Metadata"/></w:style><w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="Heading 1"/></w:style><w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="Heading 2"/></w:style><w:style w:type="paragraph" w:styleId="Bullet"><w:name w:val="Bullet"/></w:style><w:style w:type="paragraph" w:styleId="Source"><w:name w:val="Source"/></w:style></w:styles>`
}
