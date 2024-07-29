# DynoMark

Dynomark strives to be a markdown query language engine, similar to obsidian's
[Dataview plugin](https://github.com/blacksmithgu/obsidian-dataview).

This program can be used with editors like neovim and emacs to provide a similar
experience to Dataview (but very barebones for now).

## Roadmap

- [ ] Working engine
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
    - [X] [🎉 Neovim plugin 🎉](https://github.com/k-lar/dynomark.nvim)
    - [ ] ORDER BY
        - [ ] ASCENDING
        - [ ] DESCENDING
    - [ ] TABLE support
- [ ] Query syntax doc

## Examples

Here's an example markdown document:

````md
# Test Markdown File

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

Query: `LIST FROM "examples/"`

Result:

```
movie_reviews.md
tasks.md
test.md
```

Query: `TASK FROM "examples/test.md" WHERE NOT CHECKED`

Result:

```
- [ ] Implement DynoMark parser
- [ ] Write unit tests
```

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

Query: `UNORDEREDLIST FROM "examples/test.md" WHERE CONTAINS "really"`

Result:

```
- Item 3 that's
  like really
  really
  really
  long
```

Query: `ORDEREDLIST FROM "examples/test.md" WHERE CONTAINS "kinda"`

Result:

```
3. Third item that's
   kinda
   sorta
   long-ish
```

Query: `TASK FROM "examples/test.md" LIMIT 2`

Result:

```
- [ ] Implement DynoMark parser
- [x] Create test markdown file
```

Query: `TASK FROM "examples/test.md" WHERE CONTAINS "CLI" OR CONTAINS "unit"`

Result:

```
- [ ] Write unit tests
- [x] Design CLI interface
```

Query: `TASK FROM "examples/test.md" WHERE CONTAINS "CLI" OR CONTAINS "unit" AND NOT CHECKED`A

Result:

```
- [ ] Write unit tests
```

Query: `FENCEDCODE FROM "examples/test.md"`

Result:

```
func main() {
    fmt.Println("Hello, DynoMark!")
}
```
