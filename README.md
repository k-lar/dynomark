# DynoMark

Dynomark strives to be a markdown query language engine, similar to obsidian's
[Dataview plugin](https://github.com/blacksmithgu/obsidian-dataview).

This program can be used with editors like neovim and emacs to provide a similar
experience to Dataview (but very barebones for now).

## Installation

Requirements:
- Go (1.22.5)

```bash
# Clone the repository
git clone https://github.com/k-lar/dynomark
cd dynomark/

# Compile the program
make

# Install the program
sudo make install

# If you want to uninstall the program
sudo make uninstall
```

## Roadmap

- [ ] Completed engine
    - [X] LIST support
    - [X] TASKS support
    - [X] PARAGRAPH support
    - [X] ORDEREDLIST support
    - [X] UNORDEREDLIST support
    - [X] FENCEDCODE support
    - [X] Limits
    - [X] Conditional statements
        - [X] AND
        - [X] OR
    - [X] IS statement (equals / ==)
    - [ ] ORDER BY
        - [ ] ASCENDING
        - [ ] DESCENDING
    - [X] GROUP BY (metadata)
    - [X] Metadata parsing
    - [X] Query multiple files/directories at once
    - [X] Support metadata/tag based conditionals (e.g. TABLE author, published FROM example.md WHERE [author] IS "Shakespeare")
    - [X] TABLE support
        - [X] TABLE_NO_ID support (A TABLE query without ID/File column)
        - [X] Support AS statements (e.g. TABLE author AS "Author", published AS "Date published" FROM ...)
- [X] [🎉 Neovim plugin 🎉](https://github.com/k-lar/dynomark.nvim)
- [ ] Emacs plugin
- [ ] Visual Studio Code plugin
- [ ] Query syntax doc

## Examples

Here's an example markdown document:

````md
# Test Markdown File

This is a test markdown file to test the Dynomark parser.

## Tasks

- [ ] Implement DynoMark parser
- [x] Create test markdown file
- [ ] Write unit tests
- [x] Design CLI interface

## Lists

### Unordered List

- Item 1
- Item 2
- Item 3 that's
  like really
  really
  really
  long
- Item 4

### Ordered List

1. First item
2. Second item
3. Third item that's
   kinda
   sorta
   long-ish
4. Fourth item

## Code

Here's a sample code block:

```go
func main() {
    fmt.Println("Hello, DynoMark!")
}
```
````

Here are some queries and their results:

List of files in the `examples/` directory:  
Query: `LIST FROM "examples/"`

Result:

```
- movie_reviews.md
- tasks.md
- test.md
```

Paragraphs from the `examples/movie_reviews.md` and `examples/tasks.md` files:  
Query: `paragraph from examples/movie_reviews.md, examples/tasks.md`

Result:

```
Some movie review stuff here.

This is a test markdown file to test the Dynomark parser.
```

List of tasks in the `examples/test.md` file:  
Query: `TASK FROM "examples/test.md" WHERE NOT CHECKED`

Result:

```
- [ ] Implement DynoMark parser
- [ ] Write unit tests
```

List of tasks in all .md files inside `examples/` directory, grouped by file path:  
Query: `TASK FROM todos/ WHERE NOT CHECKED GROUP BY [file.path]`

Result:

```
- todos/todo-1.md                                                  
    - [ ] Task 1                                                   
    - [ ] Task 3                                                   
                                                                   
- todos/todo-2.md                                                  
    - [ ] Item 1                                                   
                                                                   
- todos/todo-3.md                                                  
    - [ ] Other task 1                                             
    - [ ] Other task 2 
```

All unordered lists in `examples/test.md`:  
Query: `UNORDEREDLIST FROM "examples/test.md"`

Result:

```
- Item 1
- Item 2
- Item 3 that's
  like really
  really
  really
  long
- Item 4
```

All unordered list items in `examples/test.md` where the list contains the word "really":  
Query: `UNORDEREDLIST FROM "examples/test.md" WHERE CONTAINS "really"`

Result:

```
- Item 3 that's
  like really
  really
  really
  long
```

All ordered list items in `examples/test.md` where the list contains the word "kinda":  
Query: `ORDEREDLIST FROM "examples/test.md" WHERE CONTAINS "kinda"`

Result:

```
3. Third item that's
   kinda
   sorta
   long-ish
```

All tasks in `examples/test.md` but limit the results to the first 2:  
Query: `TASK FROM "examples/test.md" LIMIT 2`

Result:

```
- [ ] Implement DynoMark parser
- [x] Create test markdown file
```

All tasks in `examples/test.md` where the tasks contains either the word "unit" or "CLI":  
Query: `TASK FROM "examples/test.md" WHERE CONTAINS "CLI" OR CONTAINS "unit"`

Result:

```
- [ ] Write unit tests
- [x] Design CLI interface
```

All tasks in `examples/test.md` where the tasks contains either the word "unit"
or "CLI" and the task is not checked:  
Query: `TASK FROM "examples/test.md" WHERE CONTAINS "CLI" OR CONTAINS "unit" AND NOT CHECKED`

Result:

```
- [ ] Write unit tests
```

All fenced code blocks in `examples/test.md`:
Query: `FENCEDCODE FROM "examples/test.md"`

Result:

```
func main() {
    fmt.Println("Hello, DynoMark!")
}
```

## Metadata support

Dynomark supports metadata in the form of key-value pairs. For now, you can use the
[dataview syntax](https://blacksmithgu.github.io/obsidian-dataview/annotation/add-metadata/)
to add metadata to your markdown files. Currently only the standard metadata
syntax is supported and not the alternative "hidden" syntax (maybe in the future).
To reference metadata in your queries, you have to use the following syntax:
`[metadata_key]`

The only place where that syntax is not required is in the `TABLE` query,
where you can use the metadata key directly as shown in the examples below.

There are 8 metadata fields that are defined by default for every file it processes:
- `file.path`: The path of the file
- `file.folder`: The folder of the file where it's located
- `file.link`: The markdown link to the file (relative to your current working directory)
- `file.size`: The size of the file in bytes
- `file.cday`: The creation day of the file in ISO8601 format
- `file.mday`: The modification day of the file in ISO8601 format
- `file.ctime`: The creation time of the file in ISO8601 format
- `file.mtime`: The modification time of the file in ISO8601 format

NOTE:  
IS is a strict version of the CONTAINS statement, it will only match if 
the metadata value is exactly the same as the argument after IS. It can
also be used with normal queries where CONTAINS doesn't cut it,
but that's rare because you would have to know the exact value of the
result you're looking for.

You can use metadata in your queries like so:
```
PARAGRAPH FROM "examples/" WHERE [author] IS "Shakespeare"
```

This will return all paragraphs from all .md files from `examples/`
where the metadata key `author` is `Shakespeare`.

## Tables

Dynomark supports querying metadata from files in a table format.

Here's an example query that queries all files in the `todos/`
directory by their creation date and their title:
`TABLE file.cday AS "Date created", title AS "Title" FROM todos/`

That would return a table like this:
```
| File      | Date created | Title   |
|-----------|--------------|---------|
| todo-1.md | 2024-08-17   | Title 1 |
| todo-2.md | 2024-08-18   | Title 2 |
| todo-3.md | 2024-08-19   | Title 3 |
| todo-4.md | 2024-08-20   | Title 4 |
| todo-5.md | 2024-08-21   | Title 5 |
```

You can also use the `TABLE_NO_ID` statement to create a table without the ID/File column:  
`TABLE_NO_ID file.cday AS "Date created", title AS "Title" FROM todos/`

That would return a table like this:
```
| Date created | Title   |
|--------------|---------|
| 2024-08-17   | Title 1 |
| 2024-08-18   | Title 2 |
| 2024-08-19   | Title 3 |
| 2024-08-20   | Title 4 |
| 2024-08-21   | Title 5 |
```

And an example with metadata conditionals:
`TABLE_NO_ID file.cday AS "Date created", title AS "Title" FROM todos/ WHERE [title] IS "Title 2"`

That would return a table like this:
```
| Date created | Title   |
|--------------|---------|
| 2024-08-18   | Title 2 |
```

The AS statement is optional. If you don't provide an alias, the metadata
key will be used as the column name.
