/*
Package csvout - CSV output

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/
package csvout

import (
	"fmt"
	"strings"
)

// TODO: share code base with tabout
type CSVOutput struct {
	fields        []string
	headerWritten bool
}

func New(fields []string) *CSVOutput {
	return &CSVOutput{fields, false}
}

func (t *CSVOutput) Write(values map[string]string) error {
	if !t.headerWritten {
		if _, err := fmt.Println(
			strings.Join(t.fields, "\t")); err != nil {
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
	_, err := fmt.Println(strings.Join(output, "\t"))
	return err
}

func (t *CSVOutput) Flush() error {
	return nil
}
