package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

var version string = "0.1.0"

type TokenType int

const (
	TOKEN_KEYWORD TokenType = iota
	TOKEN_IDENTIFIER
	TOKEN_FUNCTION
	TOKEN_NOT
	TOKEN_LOGICAL_OP
	TOKEN_STRING
	TOKEN_NUMBER
	TOKEN_COMMA
	TOKEN_EOF
	TOKEN_TABLE
	TOKEN_TABLE_NO_ID
	TOKEN_AS
)

type Token struct {
	Type  TokenType
	Value string
}

type QueryType string

type Metadata map[string]interface{}

const (
	LIST          QueryType = "LIST"
	TASK          QueryType = "TASK"
	PARAGRAPH     QueryType = "PARAGRAPH"
	ORDEREDLIST   QueryType = "ORDEREDLIST"
	UNORDEREDLIST QueryType = "UNORDEREDLIST"
	FENCEDCODE    QueryType = "FENCEDCODE"
	TABLE         QueryType = "TABLE"
	TABLE_NO_ID   QueryType = "TABLE_NO_ID"
)

type ASTNode interface{}

type ColumnDefinition struct {
	Name  string
	Alias string
}

type QueryNode struct {
	Type    QueryType
	From    []string
	Where   *WhereNode
	Limit   int
	Columns []ColumnDefinition
}

type WhereNode struct {
	Conditions []ConditionNode
}

type ConditionNode struct {
	IsNegated bool
	Function  string
	Value     string
	LogicalOp string // "AND" or "OR"
}

func Lex(input string) []Token {
	var tokens []Token
	words := strings.Fields(input)

	// If word has comma suffix, split it into two tokens
	for i := 0; i < len(words); i++ {
		if strings.HasSuffix(words[i], ",") {
			words = append(words[:i+1], append([]string{","}, words[i+1:]...)...)
			words[i] = strings.TrimSuffix(words[i], ",")
			i++
		}
	}

	got_from := false
	got_where := false
	insideQuotes := false
	var quotedString string

	for _, word := range words {
		// Handle quoted strings (even if they contain spaces)
		if strings.HasPrefix(word, "\"") && !insideQuotes {
			insideQuotes = true
			quotedString = word[1:]
			if strings.HasSuffix(word, "\"") && len(word) > 1 {
				insideQuotes = false
				quotedString = word[1 : len(word)-1]
				tokens = append(tokens, Token{Type: TOKEN_STRING, Value: quotedString})
			}
		} else if insideQuotes {
			if strings.HasSuffix(word, "\"") {
				insideQuotes = false
				quotedString += " " + word[:len(word)-1]
				tokens = append(tokens, Token{Type: TOKEN_STRING, Value: quotedString})
			} else {
				quotedString += " " + word
			}
		} else {
			switch strings.ToUpper(word) {
			case "TABLE":
				tokens = append(tokens, Token{Type: TOKEN_TABLE, Value: "TABLE"})
			case "TABLE_NO_ID":
				tokens = append(tokens, Token{Type: TOKEN_TABLE_NO_ID, Value: "TABLE_NO_ID"})
			case "AS":
				tokens = append(tokens, Token{Type: TOKEN_AS, Value: "AS"})
			case "LIST", "TASK", "PARAGRAPH", "ORDEREDLIST", "UNORDEREDLIST", "FENCEDCODE", "LIMIT", "CHECKED":
				tokens = append(tokens, Token{Type: TOKEN_KEYWORD, Value: strings.ToUpper(word)})
			case "FROM":
				tokens = append(tokens, Token{Type: TOKEN_KEYWORD, Value: "FROM"})
				got_from = true
			case "WHERE":
				tokens = append(tokens, Token{Type: TOKEN_KEYWORD, Value: "WHERE"})
				got_where = true
			case ",":
				tokens = append(tokens, Token{Type: TOKEN_COMMA, Value: word})
			case "CONTAINS":
				tokens = append(tokens, Token{Type: TOKEN_FUNCTION, Value: "CONTAINS"})
			case "NOT":
				tokens = append(tokens, Token{Type: TOKEN_NOT, Value: "NOT"})
			case "AND", "OR":
				tokens = append(tokens, Token{Type: TOKEN_LOGICAL_OP, Value: strings.ToUpper(word)})
			default:
				if _, err := strconv.Atoi(word); err == nil {
					tokens = append(tokens, Token{Type: TOKEN_NUMBER, Value: word})
				} else if got_from && !got_where {
					tokens = append(tokens, Token{Type: TOKEN_STRING, Value: word})
				} else {
					tokens = append(tokens, Token{Type: TOKEN_IDENTIFIER, Value: word})
				}
			}
		}
	}

	tokens = append(tokens, Token{Type: TOKEN_EOF, Value: ""})
	return tokens
}

func Parse(tokens []Token) (*QueryNode, error) {
	query := &QueryNode{Limit: -1}

	i := 0

	if tokens[i].Type == TOKEN_TABLE {
		query.Type = TABLE
	} else if tokens[i].Type == TOKEN_TABLE_NO_ID {
		query.Type = TABLE_NO_ID
	} else {
		if tokens[i].Type != TOKEN_KEYWORD {
			return nil, fmt.Errorf("expected valid query type, got %s", tokens[i].Value)
		}
		query.Type = parseQueryType(tokens[i].Value)
	}

	i++

	// Parse columns for TABLE queries
	if query.Type == TABLE || query.Type == TABLE_NO_ID {
		for i < len(tokens) && tokens[i].Type != TOKEN_KEYWORD {
			if tokens[i].Type == TOKEN_IDENTIFIER {
				columnName := tokens[i].Value
				i++
				if i < len(tokens) && tokens[i].Type == TOKEN_AS {
					i++
					if i >= len(tokens) || tokens[i].Type != TOKEN_STRING {
						return nil, fmt.Errorf("expected column alias, got %s", tokens[i].Value)
					}
					query.Columns = append(query.Columns, ColumnDefinition{
						Name:  columnName,
						Alias: tokens[i].Value,
					})
					i++
				} else {
					query.Columns = append(query.Columns, ColumnDefinition{
						Name:  columnName,
						Alias: columnName,
					})
				}
			} else if tokens[i].Type == TOKEN_COMMA {
				i++
			} else {
				return nil, fmt.Errorf("expected column name or comma, got %s", tokens[i].Value)
			}
		}
	}

	// Parse FROM clause
	if tokens[i].Value != "FROM" {
		return nil, fmt.Errorf("expected FROM, got %s", tokens[i].Value)
	}
	i++

	for i < len(tokens) && tokens[i].Type != TOKEN_KEYWORD {
		if tokens[i].Type == TOKEN_STRING {
			query.From = append(query.From, tokens[i].Value)
		}
		i++
	}

	// Parse WHERE clause
	if i < len(tokens) && tokens[i].Value == "WHERE" {
		whereNode, newIndex, err := parseWhereClause(tokens[i+1:])
		if err != nil {
			return nil, fmt.Errorf("error parsing WHERE clause: %w", err)
		}
		query.Where = whereNode
		i += newIndex + 1
	}

	// Parse LIMIT clause
	if i < len(tokens) && tokens[i].Value == "LIMIT" {
		if i+1 >= len(tokens) || tokens[i+1].Type != TOKEN_NUMBER {
			return nil, fmt.Errorf("invalid LIMIT clause")
		}
		limit, err := strconv.Atoi(tokens[i+1].Value)
		if err != nil {
			return nil, fmt.Errorf("invalid LIMIT value: %w", err)
		}
		query.Limit = limit
	}

	return query, nil
}

func parseQueryType(value string) QueryType {
	switch value {
	case "LIST":
		return LIST
	case "TASK":
		return TASK
	case "PARAGRAPH":
		return PARAGRAPH
	case "ORDEREDLIST":
		return ORDEREDLIST
	case "UNORDEREDLIST":
		return UNORDEREDLIST
	case "FENCEDCODE":
		return FENCEDCODE
	case "TABLE":
		return TABLE
	default:
		return ""
	}
}

func parseWhereClause(tokens []Token) (*WhereNode, int, error) {
	whereNode := &WhereNode{}
	i := 0
	var currentCondition ConditionNode
	var logicalOp string

	for i < len(tokens) && tokens[i].Value != "LIMIT" {
		switch tokens[i].Type {
		case TOKEN_NOT:
			currentCondition.IsNegated = true
		case TOKEN_FUNCTION:
			currentCondition.Function = tokens[i].Value
		case TOKEN_STRING:
			currentCondition.Value = tokens[i].Value
			currentCondition.LogicalOp = logicalOp
			whereNode.Conditions = append(whereNode.Conditions, currentCondition)
			currentCondition = ConditionNode{}
			logicalOp = ""
		case TOKEN_LOGICAL_OP:
			logicalOp = tokens[i].Value
		case TOKEN_KEYWORD:
			if tokens[i].Value == "CHECKED" {
				currentCondition.Function = "CHECKED"
				currentCondition.LogicalOp = logicalOp
				whereNode.Conditions = append(whereNode.Conditions, currentCondition)
				currentCondition = ConditionNode{}
				logicalOp = ""
			}
		}
		i++
	}

	return whereNode, i, nil
}

func InterpretTableQuery(ast *QueryNode) (string, error) {
	var result strings.Builder
	var headers []string

	if ast.Type == TABLE {
		headers = append(headers, "File")
	}
	// headers = append(headers, ast.Columns...)
	for _, col := range ast.Columns {
		headers = append(headers, col.Alias)
	}

	// Initialize maxWidths with the length of headers
	maxWidths := make([]int, len(headers))
	for i, header := range headers {
		maxWidths[i] = utf8.RuneCountInString(header)
	}

	// Collect all rows and calculate max width for each column
	var rows [][]string
	var paths []string

	for _, path := range ast.From {
		info, err := os.Stat(path)
		if err != nil {
			return "", err
		}

		if info.IsDir() {
			err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
					paths = append(paths, p)
				}
				return nil
			})
			if err != nil {
				return "", err
			}
		} else {
			paths = append(paths, path)
		}
	}

	for _, path := range paths {
		_, metadata, err := parseMarkdownContent(path, ast.Type)
		if err != nil {
			return "", err
		}

		var row []string
		if ast.Type == TABLE {
			row = append(row, filepath.Base(path))
		}

		for _, colDef := range ast.Columns {
			colName := colDef.Name
			if value, ok := metadata[colName]; ok {
				row = append(row, fmt.Sprintf("%v", value))
			} else {
				row = append(row, "")
			}
		}

		// Update maxWidths based on the current row
		for i, cell := range row {
			if utf8.RuneCountInString(cell) > maxWidths[i] {
				maxWidths[i] = utf8.RuneCountInString(cell)
			}
		}

		rows = append(rows, row)
	}

	// Write table headers
	for i, header := range headers {
		result.WriteString("| " + tablePadString(header, maxWidths[i]) + " ")
	}
	result.WriteString("|\n")

	// Write table header separator
	for _, width := range maxWidths {
		result.WriteString("|" + strings.Repeat("-", width+2))
	}
	result.WriteString("|\n")

	// Write table rows
	for _, row := range rows {
		for i, cell := range row {
			result.WriteString("| " + tablePadString(cell, maxWidths[i]) + " ")
		}
		result.WriteString("|\n")
	}

	return result.String(), nil
}

func tablePadString(str string, length int) string {
	return str + strings.Repeat(" ", length-utf8.RuneCountInString(str))
}

func Interpret(ast *QueryNode) (string, error) {
	if ast.Type == TABLE || ast.Type == TABLE_NO_ID {
		return InterpretTableQuery(ast)
	}

	content, err := parseMarkdownFiles(ast.From, ast.Type)
	if err != nil {
		return "", err
	}

	if ast.Where != nil {
		content = filterContent(content, ast.Type, ast.Where.Conditions)
	}

	if ast.Limit >= 0 && ast.Limit < len(content) {
		content = content[:ast.Limit]
	}

	return strings.Join(content, "\n"), nil
}

func parseMarkdownContent(path string, queryType QueryType) ([]string, Metadata, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	metadata := make(Metadata)
	inFrontMatter := false
	frontMatterLines := []string{}

	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)

		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "---" {
			inFrontMatter = !inFrontMatter
			if !inFrontMatter {
				// Process YAML front matter
				for _, fmLine := range frontMatterLines {
					if strings.Contains(fmLine, ":") {
						parts := strings.SplitN(fmLine, ":", 2)
						key := strings.ToLower(strings.TrimSpace(parts[0]))
						value := strings.TrimSpace(parts[1])
						value = strings.Trim(value, `"`)
						if b, err := strconv.ParseBool(value); err == nil {
							metadata[key] = b
						} else if i, err := strconv.Atoi(value); err == nil {
							metadata[key] = i
						} else {
							metadata[key] = value
						}
					}
				}
			}
			continue
		}

		if inFrontMatter {
			frontMatterLines = append(frontMatterLines, trimmedLine)
		} else {
			parseMetadataLine(trimmedLine, metadata)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	// Add file-related metadata
	addFileMetadata(path, metadata)

	// For TABLE and TABLE_NO_ID, no need to parse the content
	// Just return an empty slice for the content and the metadata
	if queryType == TABLE || queryType == TABLE_NO_ID {
		return []string{}, metadata, nil
	}

	// Strip YAML frontmatter from lines
	lines = stripYAMLFrontmatter(lines)

	var parsedContent []string
	switch queryType {
	case TASK:
		parsedContent = parseTasks(lines)
	case PARAGRAPH:
		parsedContent = parseParagraphs(lines)
	case ORDEREDLIST:
		parsedContent = parseOrderedLists(lines)
	case UNORDEREDLIST:
		parsedContent = parseUnorderedLists(lines)
	case FENCEDCODE:
		parsedContent = parseFencedCode(lines)
	default:
		return nil, nil, fmt.Errorf("unsupported query type: %s", queryType)
	}

	return parsedContent, metadata, nil
}

func parseMetadataLine(line string, metadata Metadata) {
	// Check for metadata in the form of key:: value
	// TODO: Parse the string yourself without regex
	if matched, _ := regexp.MatchString(`^\w+ *::`, line); matched {
		line = strings.Trim(line, " ")
		parseMetadataPair(line, metadata)
	} else if strings.HasPrefix(line, "**") && strings.Contains(line, "::") {
		line = strings.Trim(line, "* ")
		parseMetadataPair(line, metadata)
	} else if strings.HasPrefix(line, "[") && strings.Contains(line, "::") {
		line = strings.Trim(line, "[] ")
		parts := strings.Split(line, "] | [")
		for _, part := range parts {
			parseMetadataPair(part, metadata)
		}
	} else if strings.Contains(line, "[") && strings.Contains(line, "::") {
		for strings.Contains(line, "[") && strings.Contains(line, "::") {
			start := strings.Index(line, "[")
			end := strings.Index(line, "]")
			if start != -1 && end != -1 && start < end {
				inlineMetadata := line[start+1 : end]
				parseMetadataPair(inlineMetadata, metadata)
				line = line[end+1:]
			} else {
				break
			}
		}
	}
}

func parseMetadataPair(pair string, metadata Metadata) {
	parts := strings.SplitN(pair, "::", 2)
	if len(parts) == 2 {
		key := strings.ToLower(strings.TrimSpace(strings.Trim(parts[0], "*")))
		value := strings.TrimSpace(parts[1])
		if b, err := strconv.ParseBool(value); err == nil {
			metadata[key] = b
		} else if i, err := strconv.Atoi(value); err == nil {
			metadata[key] = i
		} else {
			metadata[key] = value
		}
	}
}

func addFileMetadata(path string, metadata Metadata) {
	fileInfo, err := os.Stat(path)
	if err == nil {
		metadata["file.folder"] = filepath.Dir(path)
		metadata["file.path"] = path
		metadata["file.link"] = fmt.Sprintf("[%s](%s)", filepath.Base(path), filepath.Base(filepath.Dir(path)))
		metadata["file.size"] = fileInfo.Size()
		metadata["file.ctime"] = fileInfo.ModTime().Format(time.RFC3339)
		metadata["file.cday"] = fileInfo.ModTime().Format("2006-01-02")
		metadata["file.mtime"] = fileInfo.ModTime().Format(time.RFC3339)
		metadata["file.mday"] = fileInfo.ModTime().Format("2006-01-02")
	}
}

func stripYAMLFrontmatter(lines []string) []string {
	if len(lines) > 0 && lines[0] == "---" {
		endIndex := -1
		for i := 1; i < len(lines); i++ {
			if lines[i] == "---" {
				endIndex = i
				break
			}
		}
		if endIndex != -1 {
			return lines[endIndex+1:]
		}
	}
	return lines
}

func parseTasks(lines []string) []string {
	var tasks []string
	for _, line := range lines {
		if strings.HasPrefix(line, "- [ ]") || strings.HasPrefix(line, "- [x]") || strings.HasPrefix(line, "- [X]") {
			tasks = append(tasks, line)
		}
	}
	return tasks
}

func parseParagraphs(lines []string) []string {
	var paragraphs []string
	var inCodeBlock bool
	var inList bool

	for _, line := range lines {
		// Skip fenced blocks and their content
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}

		if inCodeBlock {
			continue
		}

		// Skip headings
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Skip unordered list items and tasks
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			inList = true
			continue
		}

		// Skip ordered list items
		if matched, _ := regexp.MatchString(`^\d+\.\s`, line); matched {
			inList = true
			continue
		}

		// If we're in a list and the line is empty, we're done with the list
		if inList && strings.TrimSpace(line) == "" {
			inList = false
		}

		// Skip indented lines if we're in a list
		if inList && len(line)-len(strings.TrimLeft(line, " ")) > 0 {
			continue
		}

		paragraphs = append(paragraphs, line)
	}

	return paragraphs
}

func parseUnorderedLists(lines []string) []string {
	var items []string
	var currentItem []string
	inList := false
	indentLevel := 0
	trailingEmptyLines := 0

	for i, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "-") && !strings.HasPrefix(trimmedLine, "- [") && trimmedLine != "---" {
			if len(currentItem) > 0 {
				items = append(items, strings.Join(currentItem[:len(currentItem)-trailingEmptyLines], "\n"))
				currentItem = nil
				trailingEmptyLines = 0
			}
			currentItem = append(currentItem, line)
			inList = true
			indentLevel = len(line) - len(trimmedLine)
		} else if inList && (strings.HasPrefix(trimmedLine, "-") || len(line)-len(strings.TrimLeft(line, " ")) > indentLevel) {
			currentItem = append(currentItem[:len(currentItem)-trailingEmptyLines], line)
			trailingEmptyLines = 0
		} else if inList && trimmedLine == "" {
			currentItem = append(currentItem, line)
			trailingEmptyLines++
		} else {
			if len(currentItem) > 0 {
				items = append(items, strings.Join(currentItem[:len(currentItem)-trailingEmptyLines], "\n"))
				currentItem = nil
				trailingEmptyLines = 0
			}
			inList = false
			indentLevel = 0
		}

		// Handle the case when we reach the end of the file
		if i == len(lines)-1 && len(currentItem) > 0 {
			items = append(items, strings.Join(currentItem[:len(currentItem)-trailingEmptyLines], "\n"))
		}
	}

	return items
}

func parseOrderedLists(lines []string) []string {
	var items []string
	var currentItem []string
	inList := false
	indentLevel := 0

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if matched, _ := regexp.MatchString(`^\d+\.\s`, trimmedLine); matched && (!inList || len(line)-len(strings.TrimLeft(line, " ")) <= indentLevel) {
			if len(currentItem) > 0 {
				items = append(items, strings.Join(currentItem, "\n"))
				currentItem = nil
			}
			currentItem = append(currentItem, line)
			inList = true
			indentLevel = len(line) - len(trimmedLine)
		} else if trimmedLine == "" && inList {
			if len(currentItem) > 0 {
				items = append(items, strings.Join(currentItem, "\n"))
				currentItem = nil
			}
			inList = false
			indentLevel = 0
		} else if inList && (strings.HasPrefix(trimmedLine, "") || len(line)-len(strings.TrimLeft(line, " ")) > indentLevel) {
			currentItem = append(currentItem, line)
		} else {
			if len(currentItem) > 0 {
				items = append(items, strings.Join(currentItem, "\n"))
				currentItem = nil
			}
			inList = false
			indentLevel = 0
		}
	}

	if len(currentItem) > 0 {
		items = append(items, strings.Join(currentItem, "\n"))
	}

	return items
}

func parseFencedCode(lines []string) []string {
	var fencedCode []string
	var currentCode []string
	inCodeBlock := false

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				fencedCode = append(fencedCode, strings.Join(currentCode, "\n"))
				currentCode = nil
				inCodeBlock = false
			} else {
				inCodeBlock = true
			}
		} else if inCodeBlock {
			currentCode = append(currentCode, line)
		}
	}

	return fencedCode
}

func parseMarkdownFiles(paths []string, queryType QueryType) ([]string, error) {
	var results []string

	for _, path := range paths {
		if strings.HasPrefix(path, "~") {
			path = filepath.Join(os.Getenv("HOME"), path[1:])
		}

		path = os.ExpandEnv(path)
		fileInfo, err := os.Stat(path)
		if err != nil {
			return nil, err
		}

		if fileInfo.IsDir() {
			files, err := filepath.Glob(filepath.Join(path, "*.md"))
			if err != nil {
				return nil, err
			}
			if queryType == LIST {
				for _, file := range files {
					results = append(results, "- "+filepath.Base(file))
				}
			} else {
				for _, file := range files {
					// INFO: This also returns metadata
					content, _, err := parseMarkdownContent(file, queryType)
					if err != nil {
						return nil, err
					}
					results = append(results, content...)
				}
			}
		} else {
			if queryType == LIST {
				results = append(results, filepath.Base(path))
			} else {
				// INFO: This also returns metadata
				content, _, err := parseMarkdownContent(path, queryType)
				if err != nil {
					return nil, err
				}
				results = append(results, content...)
			}
		}
	}

	return results, nil
}

func applyConditions(item string, queryType QueryType, conditions []ConditionNode) bool {
	// TODO: QueryType arg is not used yet but will need to be in the future
	if len(conditions) == 0 {
		return true
	}

	result := true
	for i, condition := range conditions {
		conditionMet := false
		switch condition.Function {
		case "CONTAINS":
			conditionMet = strings.Contains(strings.ToLower(item), strings.ToLower(condition.Value))
		case "CHECKED":
			isChecked := strings.Contains(item, "[x]") || strings.Contains(item, "[X]")
			conditionMet = isChecked
		}

		if condition.IsNegated {
			conditionMet = !conditionMet
		}

		if i == 0 {
			result = conditionMet
		} else if condition.LogicalOp == "OR" {
			result = result || conditionMet
		} else {
			result = result && conditionMet
		}
	}

	return result
}

func filterContent(content []string, queryType QueryType, conditions []ConditionNode) []string {
	var filteredContent []string

	for _, item := range content {
		if applyConditions(item, queryType, conditions) {
			filteredContent = append(filteredContent, item)
		}
	}

	return filteredContent
}

func readFromPipe() (string, error) {
	bytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func executeQuery(query string) (string, error) {
	tokens := Lex(query)
	ast, err := Parse(tokens)
	if err != nil {
		return "", fmt.Errorf("failed to parse query: %w", err)
	}

	result, err := Interpret(ast)
	if err != nil {
		return "", fmt.Errorf("failed to execute query: %w", err)
	}

	return result, nil
}

func main() {
	var query string
	var err error
	versionFlag := flag.Bool("v", false, "print the version number")
	longVersionFlag := flag.Bool("version", false, "print the version number")

	flag.StringVar(&query, "query", "", "The query string to be processe")
	flag.StringVar(&query, "q", "", "The query string to be processed (shorthand)")

	flag.Parse()

	if *versionFlag || *longVersionFlag {
		fmt.Println("Version:", version)
		os.Exit(0)
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		if query == "" {
			// Data is being piped to stdin
			query, err = readFromPipe()
		} else {
			fmt.Println("ERROR: Can't read from pipe when query is given as a parameter already.")
		}
	} else if query == "" {
		fmt.Println("No query provided. Use -q or --query to specify the query string.")
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	result, err := executeQuery(query)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	w := bufio.NewWriter(os.Stdout)
	_, err = w.WriteString(result + "\n")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing result: %v\n", err)
		os.Exit(1)
	}
	err = w.Flush()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error flushing output: %v\n", err)
		os.Exit(1)
	}
}
