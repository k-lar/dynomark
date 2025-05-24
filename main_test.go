package main

import (
	"testing"
)

type TestQuery struct {
	name     string
	query    string
	expected string
}

func runTestQueries(t *testing.T, queries []TestQuery) {
	for _, test := range queries {
		msg, err := executeQuery(test.query, false)
		if err != nil {
			t.Errorf("Error executing query: %v", err)
			continue
		}
		if msg != test.expected {
			t.Errorf("\nQuery: %s\nExpected output:\n%s\nGot:\n%s", test.query, test.expected, msg)
		}
	}
}

func TestListQueries(t *testing.T) {
	queries := []TestQuery{
		{
			name:  "LIST query on a directory",
			query: "LIST FROM \"examples/\"",
			expected: `- movie_reviews.md
- tasks.md
- test.md`,
		},
	}
	runTestQueries(t, queries)
}

func TestParagraphQueries(t *testing.T) {
	queries := []TestQuery{
		{
			name:  "PARAGRAPH query with a single file",
			query: "PARAGRAPH FROM \"examples/tasks.md\"",
			expected: `Lorem ipsum dolor sit amet, officia excepteur ex fugiat reprehenderit enim labore
culpa sint ad nisi Lorem pariatur mollit ex esse exercitation amet. Nisi anim
cupidatat excepteur officia. Reprehenderit nostrud nostrud ipsum Lorem est aliquip
amet voluptate voluptate dolor minim nulla est proident. Nostrud officia pariatur
ut officia. Sit irure elit esse ea nulla sunt ex occaecat reprehenderit commodo
officia dolor Lorem duis laboris cupidatat officia voluptate. Culpa proident
adipisicing id nulla nisi laboris ex in Lorem sunt duis officia eiusmod. Aliqua
reprehenderit commodo ex non excepteur duis sunt velit enim. Voluptate laboris
sint cupidatat ullamco ut ea consectetur et est culpa et culpa duis.

Lorem ipsum dolor sit amet, qui minim labore adipisicing minim sint cillum sint
consectetur cupidatat.`,
		},
		{
			name:  "PARAGRAPH query with a single file and a condition",
			query: "PARAGRAPH FROM \"examples/movie_reviews.md\" WHERE CONTAINS \"Generic\"",
			expected: `**Generic race movie**
**Generic Action movie**`,
		},
		{
			name:  "PARAGRAPH query with a single file and a negative condition",
			query: "PARAGRAPH FROM \"examples/movie_reviews.md\" WHERE NOT CONTAINS \"Lorem\"",
			expected: `**Generic race movie**


**Generic Action movie**


What's cool:`,
		},
	}
	runTestQueries(t, queries)
}

func TestTaskQueries(t *testing.T) {
	queries := []TestQuery{
		{
			name:  "TASK query with a single file",
			query: "TASK FROM \"examples/tasks.md\"",
			expected: `- [ ] Task 1
- [X] Task 2
- [ ] Task 3
- [X] Task 4
- [X] Task 5
- [ ] Task 6
- [ ] Task 7
- [X] Task 8`,
		},
		{
			name:  "TASK query with a single file and a condition",
			query: "TASK FROM \"examples/tasks.md\" WHERE CHECKED",
			expected: `- [X] Task 2
- [X] Task 4
- [X] Task 5
- [X] Task 8`,
		},
		{
			name:  "TASK query with a single file and a negative condition",
			query: "TASK FROM \"examples/tasks.md\" WHERE NOT CHECKED",
			expected: `- [ ] Task 1
- [ ] Task 3
- [ ] Task 6
- [ ] Task 7`,
		},
	}
	runTestQueries(t, queries)
}

func TestUnorderedListQueries(t *testing.T) {
	queries := []TestQuery{
		{
			name:  "UNORDEREDLIST query with a single file",
			query: "UNORDEREDLIST FROM \"examples/test.md\"",
			expected: `- Item 1
- Item 2
- Item 3 that's
  like really
  really
  really
  long
- Item 4`,
		},
		{
			name:     "UNORDEREDLIST query with a single file and a condition",
			query:    "UNORDEREDLIST FROM \"examples/test.md\" WHERE CONTAINS \"Item 2\"",
			expected: `- Item 2`,
		},
		{
			name:  "UNORDEREDLIST query with a single file and a negative condition",
			query: "UNORDEREDLIST FROM \"examples/test.md\" WHERE NOT CONTAINS \"Item 2\"",
			expected: `- Item 1
- Item 3 that's
  like really
  really
  really
  long
- Item 4`,
		},
		{
			name:  "UNORDEREDLIST query with a single file and a condition for a long item",
			query: "UNORDEREDLIST FROM \"examples/test.md\" WHERE CONTAINS \"really\"",
			expected: `- Item 3 that's
  like really
  really
  really
  long`,
		},
		{
			name:  "UNORDEREDLIST query with a single file and a condition for excluding a long item",
			query: "UNORDEREDLIST FROM \"examples/test.md\" WHERE NOT CONTAINS \"really\"",
			expected: `- Item 1
- Item 2
- Item 4`,
		},
	}

	runTestQueries(t, queries)
}
