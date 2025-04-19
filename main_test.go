package main

import (
    "testing"
)

// TestHelloName calls greetings.Hello with a name, checking
// for a valid return value.
func TestMetadataQuery(t *testing.T) {
    query := "LIST FROM \"examples/\" WHERE [author] IS Shakespeare"
    expectedOutput := "- shakespeare_quotes.md" 
    msg, _ := executeQuery(query, false)
    if msg != expectedOutput {
      t.Error("\n" + query + "\nis expected to output:\n" + expectedOutput + "\n Got \n" + msg)
    }
}
