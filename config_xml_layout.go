package goarklog

import (
	"strings"
)

func (a xmlAppender) layout() (layoutConfig, error) {
	for _, layout := range []xmlLayout{
		a.PatternLayout,
		a.TextLayout,
		a.JsonLayout,
		a.JSONLayout,
		a.JsonTemplate,
		a.XmlLayout,
		a.XMLLayout,
		a.CsvLayout,
		a.CSVLayout,
		a.GelfLayout,
		a.GELFLayout,
		a.Rfc5424Layout,
		a.RFC5424Layout,
		a.SyslogLayout,
		a.YamlLayout,
		a.YAMLLayout,
		a.HtmlLayout,
		a.HTMLLayout,
		a.Layout,
	} {
		if layout.XMLName.Local == "" &&
			strings.TrimSpace(layout.Type) == "" &&
			strings.TrimSpace(layout.Pattern) == "" &&
			strings.TrimSpace(layout.Template) == "" &&
			strings.TrimSpace(layout.TemplateURI) == "" &&
			layout.emptyOptions() {
			continue
		}
		return layout.config()
	}
	return layoutConfig{}, nil
}

func (l xmlLayout) emptyOptions() bool {
	return strings.TrimSpace(l.Compact) == "" &&
		strings.TrimSpace(l.EventEOL) == "" &&
		strings.TrimSpace(l.Complete) == "" &&
		strings.TrimSpace(l.IncludeStacktrace) == "" &&
		strings.TrimSpace(l.StacktraceAsString) == "" &&
		strings.TrimSpace(l.PropertiesAsList) == "" &&
		strings.TrimSpace(l.IncludeNullDelimiter) == "" &&
		strings.TrimSpace(l.DisableANSI) == "" &&
		strings.TrimSpace(l.Header) == "" &&
		strings.TrimSpace(l.Footer) == ""
}

func (l xmlLayout) config() (layoutConfig, error) {
	kind := firstNonBlank(l.Type, l.XMLName.Local)
	switch normalizeKind(kind) {
	case "", "patternlayout", "pattern":
		kind = "pattern"
	case "textlayout", "text":
		kind = "text"
	case "jsonlayout", "json":
		kind = "json"
	case "jsontemplatelayout", "jsontemplate":
		kind = "jsonTemplate"
	case "xmllayout", "xml":
		kind = "xml"
	case "csvlayout", "csv":
		kind = "csv"
	case "gelflayout", "gelf":
		kind = "gelf"
	case "rfc5424layout", "rfc5424":
		kind = "rfc5424"
	case "sysloglayout", "syslog":
		kind = "syslog"
	case "yamllayout", "yaml":
		kind = "yaml"
	case "htmllayout", "html":
		kind = "html"
	default:
		kind = l.Type
	}
	compact, err := parseXMLBool(l.Compact, "compact")
	if err != nil {
		return layoutConfig{}, err
	}
	eventEOL, err := parseXMLBool(l.EventEOL, "eventEol")
	if err != nil {
		return layoutConfig{}, err
	}
	complete, err := parseXMLBool(l.Complete, "complete")
	if err != nil {
		return layoutConfig{}, err
	}
	includeStacktrace, err := parseXMLBool(l.IncludeStacktrace, "includeStacktrace")
	if err != nil {
		return layoutConfig{}, err
	}
	stacktraceAsString, err := parseXMLBool(l.StacktraceAsString, "stacktraceAsString")
	if err != nil {
		return layoutConfig{}, err
	}
	propertiesAsList, err := parseXMLBool(l.PropertiesAsList, "propertiesAsList")
	if err != nil {
		return layoutConfig{}, err
	}
	includeNullDelimiter, err := parseXMLBool(l.IncludeNullDelimiter, "includeNullDelimiter")
	if err != nil {
		return layoutConfig{}, err
	}
	disableANSI, err := parseXMLBool(l.DisableANSI, "disableAnsi")
	if err != nil {
		return layoutConfig{}, err
	}
	return layoutConfig{
		Type:                 kind,
		Pattern:              l.Pattern,
		EventTemplate:        l.Template,
		EventTemplateURI:     l.TemplateURI,
		Compact:              compact,
		EventEOL:             eventEOL,
		Complete:             complete,
		IncludeStacktrace:    includeStacktrace,
		StacktraceAsString:   stacktraceAsString,
		PropertiesAsList:     propertiesAsList,
		IncludeNullDelimiter: includeNullDelimiter,
		DisableANSI:          disableANSI,
		Header:               l.Header,
		Footer:               l.Footer,
	}, nil
}
