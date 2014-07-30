/*
Package tabout - Tabular text output, convenience wrapper over "text/tabwriter"

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/
package tabout

import (
	"os"
	"strings"
	"text/tabwriter"
)

type TabOutput struct {
	fields []string
	*tabwriter.Writer
	headerWritten bool
}

func New(fields []string, enabled map[string]bool) *TabOutput {
	tab := TabOutput{fields, nil, false}
	if enabled != nil {
		// filter out disabled fields, TODO: make this a utility function
		for i := len(tab.fields) - 1; i >= 0; i-- {
			if value, ok := enabled[tab.fields[i]]; ok && !value {
				tab.fields = append(tab.fields[:i],
					tab.fields[i+1:]...)
			}
		}
	}
	return &tab
}
func (t *TabOutput) SetFields(fields []string) {
	t.fields = fields
}

func (t *TabOutput) Write(values map[string]string) error {
	if t.Writer == nil {
		t.Writer = tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	}
	if !t.headerWritten {
		if _, err := t.Writer.Write([]byte(
			strings.Join(t.fields, "\t") + "\n")); err != nil {
			return err
		}
		t.headerWritten = true
	}
	output := make([]string, len(t.fields))
	for i, key := range t.fields {
		// TODO: escape values, trim to max length
		if value, ok := values[key]; ok {
			output[i] = value
		} else {
			output[i] = "(nil)" // TODO: return error?
		}
	}
	_, err := t.Writer.Write([]byte(strings.Join(output, "\t") + "\n"))
	return err
}

func (t *TabOutput) Flush() error {
	if t.Writer != nil {
		t.Writer.Flush()
	}
	return nil
}
