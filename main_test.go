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
			query: "LIST FROM \"examples/misc/\"",
			expected: `- movie_reviews.md
- tasks.md
- test.md`,
		},
		{
			name:     "LIST query on a directory with WHERE clause on file metadata",
			query:    "LIST FROM \"examples/misc/\" WHERE [file.shortname] IS \"test\"",
			expected: `- test.md`,
		},
		{
			name:     "LIST query on a directory with WHERE clause on user defined metadata",
			query:    "LIST FROM \"examples/misc/\" WHERE [author] IS \"John Doe\"",
			expected: `- movie_reviews.md`,
		},
	}

	runTestQueries(t, queries)
}

func TestParagraphQueries(t *testing.T) {
	queries := []TestQuery{
		{
			name:  "PARAGRAPH query with a single file",
			query: "PARAGRAPH FROM \"examples/misc/tasks.md\"",
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
			query: "PARAGRAPH FROM \"examples/misc/movie_reviews.md\" WHERE CONTAINS \"Generic\"",
			expected: `**Generic race movie**
**Generic Action movie**`,
		},
		{
			name:  "PARAGRAPH query with a single file and a negative condition",
			query: "PARAGRAPH FROM \"examples/misc/movie_reviews.md\" WHERE NOT CONTAINS \"Lorem\"",
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
			query: "TASK FROM \"examples/misc/tasks.md\"",
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
			query: "TASK FROM \"examples/misc/tasks.md\" WHERE CHECKED",
			expected: `- [X] Task 2
- [X] Task 4
- [X] Task 5
- [X] Task 8`,
		},
		{
			name:  "TASK query with a single file and a negative condition",
			query: "TASK FROM \"examples/misc/tasks.md\" WHERE NOT CHECKED",
			expected: `- [ ] Task 1
- [ ] Task 3
- [ ] Task 6
- [ ] Task 7`,
		},
		{
			name:  "TASK query with a single file and a limit of 3",
			query: "TASK FROM \"examples/misc/test.md\" LIMIT 3",
			expected: `- [ ] Implement DynoMark parser
- [ ] Implement DynoMark parser but better
- [x] Create test markdown file`,
		},
		{
			name:  "TASK query with a single file with two conditions with OR",
			query: "TASK FROM \"examples/misc/test.md\" WHERE CONTAINS \"CLI\" OR CONTAINS \"unit\"",
			expected: `- [ ] Write unit tests
- [x] Design CLI interface`,
		},
		{
			name:  "TASK query with a single file and a condition where the task is checked",
			query: "TASK FROM \"examples/misc/test.md\" WHERE CHECKED",
			expected: `- [x] Create test markdown file
- [x] Design CLI interface`,
		},
		{
			name:  "TASK query with a single file and a condition where the task is not checked",
			query: "TASK FROM \"examples/misc/test.md\" WHERE NOT CHECKED",
			expected: `- [ ] Implement DynoMark parser
- [ ] Implement DynoMark parser but better
- [ ] Write unit tests`,
		},
		{
			name:     "TASK query with a single file and 3 conditions with OR and AND",
			query:    "TASK FROM \"examples/misc/test.md\" WHERE CONTAINS \"CLI\" OR CONTAINS \"unit\" AND NOT CHECKED",
			expected: `- [ ] Write unit tests`,
		},
	}

	runTestQueries(t, queries)
}

func TestUnorderedListQueries(t *testing.T) {
	queries := []TestQuery{
		{
			name:  "UNORDEREDLIST query with a single file",
			query: "UNORDEREDLIST FROM \"examples/misc/test.md\"",
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
			query:    "UNORDEREDLIST FROM \"examples/misc/test.md\" WHERE CONTAINS \"Item 2\"",
			expected: `- Item 2`,
		},
		{
			name:  "UNORDEREDLIST query with a single file and a negative condition",
			query: "UNORDEREDLIST FROM \"examples/misc/test.md\" WHERE NOT CONTAINS \"Item 2\"",
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
			query: "UNORDEREDLIST FROM \"examples/misc/test.md\" WHERE CONTAINS \"really\"",
			expected: `- Item 3 that's
  like really
  really
  really
  long`,
		},
		{
			name:  "UNORDEREDLIST query with a single file and a condition for excluding a long item",
			query: "UNORDEREDLIST FROM \"examples/misc/test.md\" WHERE NOT CONTAINS \"really\"",
			expected: `- Item 1
- Item 2
- Item 4`,
		},
	}

	runTestQueries(t, queries)
}

func TestOrderedListQueries(t *testing.T) {
	queries := []TestQuery{
		{
			name:  "ORDEREDLIST query with a single file",
			query: "ORDEREDLIST FROM \"examples/misc/test.md\"",
			expected: `1. First item
2. Second item
3. Third item that's
   kinda
   sorta
   long-ish
4. Fourth item`,
		},
		{
			name:  "ORDEREDLIST query with a single file and a condition",
			query: "ORDEREDLIST FROM \"examples/misc/test.md\" WHERE CONTAINS \"kinda\"",
			expected: `3. Third item that's
   kinda
   sorta
   long-ish`,
		},
	}

	runTestQueries(t, queries)
}

func TestFencedCodeQueries(t *testing.T) {
	queries := []TestQuery{
		{
			name:  "FENCEDCODE query with a single file",
			query: "FENCEDCODE FROM \"examples/misc/test.md\"",
			expected: `func main() {
    fmt.Println("Hello, DynoMark!")
}`,
		},
	}

	runTestQueries(t, queries)
}

func TestTableQueries(t *testing.T) {
	queries := []TestQuery{
		{
			name:  "TABLE query with 5 files with title metadata (frontmatter)",
			query: "TABLE file.path AS \"Relative path\", title AS \"Title\" FROM \"examples/todos/\"",
			expected: `| File            | Relative path                  | Title                                     |
|-----------------|--------------------------------|-------------------------------------------|
| todo-basic.md   | examples/todos/todo-basic.md   | My basic TODOs                            |
| todo-long.md    | examples/todos/todo-long.md    | My long TODOs file                        |
| todo-nested.md  | examples/todos/todo-nested.md  | Weirdly nested TODOs                      |
| todo-project.md | examples/todos/todo-project.md | Project TODO                              |
| todo-states.md  | examples/todos/todo-states.md  | A file for my TODOs with different states |
`,
		},
	}

	runTestQueries(t, queries)
}

func TestTableNoIdQueries(t *testing.T) {
	queries := []TestQuery{
		{
			name:  "TABLE NO ID query with 5 files with title metadata (frontmatter)",
			query: "TABLE NO ID file.path AS \"Relative path\", title AS \"Title\" FROM \"examples/todos/\"",
			expected: `| Relative path                  | Title                                     |
|--------------------------------|-------------------------------------------|
| examples/todos/todo-basic.md   | My basic TODOs                            |
| examples/todos/todo-long.md    | My long TODOs file                        |
| examples/todos/todo-nested.md  | Weirdly nested TODOs                      |
| examples/todos/todo-project.md | Project TODO                              |
| examples/todos/todo-states.md  | A file for my TODOs with different states |
`,
		},
	}

	runTestQueries(t, queries)
}
