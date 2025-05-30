package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var version string = "0.2.1"

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
	TOKEN_TABLE_NO_ID // DEPRECATED: Use 'TABLE NO ID' syntax instead.
	TOKEN_AS
	TOKEN_METADATA
	TOKEN_GROUP
	TOKEN_BY
	TOKEN_SORT
)

var TokenTypeNames = map[TokenType]string{
	TOKEN_KEYWORD:     "TOKEN_KEYWORD",
	TOKEN_IDENTIFIER:  "TOKEN_IDENTIFIER",
	TOKEN_FUNCTION:    "TOKEN_FUNCTION",
	TOKEN_NOT:         "TOKEN_NOT",
	TOKEN_LOGICAL_OP:  "TOKEN_LOGICAL_OP",
	TOKEN_STRING:      "TOKEN_STRING",
	TOKEN_NUMBER:      "TOKEN_NUMBER",
	TOKEN_COMMA:       "TOKEN_COMMA",
	TOKEN_EOF:         "TOKEN_EOF",
	TOKEN_TABLE:       "TOKEN_TABLE",
	TOKEN_TABLE_NO_ID: "TOKEN_TABLE_NO_ID",
	TOKEN_AS:          "TOKEN_AS",
	TOKEN_METADATA:    "TOKEN_METADATA",
	TOKEN_GROUP:       "TOKEN_GROUP",
	TOKEN_BY:          "TOKEN_BY",
	TOKEN_SORT:        "TOKEN_SORT",
}

func (t TokenType) String() string {
	return TokenTypeNames[t]
}

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

type ColumnDefinition struct {
	Name  string
	Alias string
}

type QueryNode struct {
	Type       QueryType
	From       []string
	Where      *WhereNode
	GroupBy    string
	GroupLimit int
	Limit      int
	Columns    []ColumnDefinition
	Sorts      []SortNode
}

type SortNode struct {
	Metadata      string
	SortDirection string
}

type WhereNode struct {
	Conditions []ConditionNode
}

type ConditionNode struct {
	IsNegated  bool
	IsMetadata bool
	Field      string // Metadata field
	Function   string
	Value      string
	LogicalOp  string // "AND" or "OR"
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
	got_sort := false
	insideQuotes := false
	var quotedString string

	for _, word := range words {
		// Handle metadata (e.g. [author])
		if strings.HasPrefix(word, "[") && strings.HasSuffix(word, "]") {
			tokens = append(tokens, Token{Type: TOKEN_METADATA, Value: strings.Trim(word, "[]")})
			// Handle quoted strings (even if they contain spaces)
		} else if strings.HasPrefix(word, "\"") && !insideQuotes {
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
				// DEPRECATED: Use 'TABLE NO ID' syntax instead
				fmt.Fprintf(os.Stderr, "Warning: 'TABLE_NO_ID' token is deprecated. Use 'TABLE NO ID' syntax instead.\n")
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
			case "GROUP":
				tokens = append(tokens, Token{Type: TOKEN_GROUP, Value: "GROUP"})
			case "SORT":
				tokens = append(tokens, Token{Type: TOKEN_SORT, Value: "SORT"})
				got_sort = true
			case "BY":
				tokens = append(tokens, Token{Type: TOKEN_BY, Value: "BY"})
			case ",":
				tokens = append(tokens, Token{Type: TOKEN_COMMA, Value: word})
			case "CONTAINS":
				tokens = append(tokens, Token{Type: TOKEN_FUNCTION, Value: "CONTAINS"})
			case "IS":
				tokens = append(tokens, Token{Type: TOKEN_FUNCTION, Value: "IS"})
			case "NOT":
				tokens = append(tokens, Token{Type: TOKEN_NOT, Value: "NOT"})
			case "AND", "OR":
				tokens = append(tokens, Token{Type: TOKEN_LOGICAL_OP, Value: strings.ToUpper(word)})
			default:
				if _, err := strconv.Atoi(word); err == nil {
					tokens = append(tokens, Token{Type: TOKEN_NUMBER, Value: word})
					// If previous token was 'TABLE' and current word is 'NO', uppercase it
				} else if len(tokens) > 0 && tokens[len(tokens)-1].Type == TOKEN_TABLE && strings.ToUpper(word) == "NO" {
					tokens = append(tokens, Token{Type: TOKEN_IDENTIFIER, Value: "NO"})
					// If previous tokens were 'TABLE' and 'NO', and current word is 'ID', uppercase it
				} else if len(tokens) > 1 && tokens[len(tokens)-2].Type == TOKEN_TABLE && tokens[len(tokens)-1].Type == TOKEN_IDENTIFIER && strings.ToUpper(word) == "ID" {
					tokens = append(tokens, Token{Type: TOKEN_IDENTIFIER, Value: "ID"})
				} else if got_from && !got_where && !got_sort {
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
		// Check for 'NO ID' after 'TABLE'
		if i+2 < len(tokens) &&
			tokens[i+1].Type == TOKEN_IDENTIFIER && tokens[i+1].Value == "NO" &&
			tokens[i+2].Type == TOKEN_IDENTIFIER && tokens[i+2].Value == "ID" {
			query.Type = TABLE_NO_ID
			i += 3
		} else {
			i++
		}
	} else if tokens[i].Type == TOKEN_TABLE_NO_ID {
		// DEPRECATED: Handle the old TOKEN_TABLE_NO_ID for backward compatibility
		query.Type = TABLE_NO_ID
		i++
	} else {
		if tokens[i].Type != TOKEN_KEYWORD {
			return nil, fmt.Errorf("expected valid query type, got %s", tokens[i].Value)
		}
		query.Type = parseQueryType(tokens[i].Value)
		i++
	}

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
	} else {
		i++
	}

	for i < len(tokens) && tokens[i].Type != TOKEN_KEYWORD {
		if tokens[i].Type == TOKEN_GROUP || tokens[i].Type == TOKEN_SORT {
			break
		} else if tokens[i].Type == TOKEN_STRING {
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

	// Parse SORT clause
	if i < len(tokens) && tokens[i].Value == "SORT" {
		sortNodes, newIndex, err := parseSortClause(tokens[i+1:], query)
		if err != nil {
			return nil, fmt.Errorf("error parsing SORT clause: %w", err)
		}
		query.Sorts = sortNodes
		i += newIndex + 1
	}

	// Parse GROUP BY clause
	if i < len(tokens) && tokens[i].Type == TOKEN_GROUP {
		i++
		if i < len(tokens) && tokens[i].Type == TOKEN_BY {
			i++
			if i < len(tokens) && tokens[i].Type == TOKEN_NUMBER {
				query.GroupLimit, _ = strconv.Atoi(tokens[i].Value)
				i++
				if i < len(tokens) && tokens[i].Type != TOKEN_METADATA {
					return nil, fmt.Errorf("expected metadata field after GROUP BY %s, got %s", tokens[i-1].Value, tokens[i].Value)
				}
			}
			if i < len(tokens) && tokens[i].Type == TOKEN_METADATA {
				query.GroupBy = tokens[i].Value
				i++
			} else {
				return nil, fmt.Errorf("expected metadata field after GROUP BY, got %s", tokens[i].Value)
			}
		} else {
			return nil, fmt.Errorf("expected BY after GROUP, got %s", tokens[i].Value)
		}
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

func parseSortClause(tokens []Token, queryNode *QueryNode) ([]SortNode, int, error) {
	i := 0
	var gotGroup bool
	var gotLimit bool
	var sortNodes []SortNode
	var sortTokens []Token

	// Isolate the sort tokens here
	for i < len(tokens) && tokens[i].Value != "LIMIT" && tokens[i].Value != "GROUP" && tokens[i].Type != TOKEN_EOF {
		switch tokens[i].Type {
		case TOKEN_METADATA:
			sortTokens = append(sortTokens, tokens[i])
		case TOKEN_COMMA:
			sortTokens = append(sortTokens, tokens[i])
		case TOKEN_STRING, TOKEN_IDENTIFIER:
			if strings.ToUpper(tokens[i].Value) == "DESC" || strings.ToUpper(tokens[i].Value) == "ASC" {
				sortTokens = append(sortTokens, tokens[i])
			}
		}

		i++
	}

	// Split sortTokens into separate sortTokens based on commas
	i = 0
	newSortTokens := make([][]Token, 0)
	for i < len(sortTokens) {
		if sortTokens[i].Type == TOKEN_COMMA {
			i++
			continue
		}
		var sortToken []Token
		for i < len(sortTokens) && sortTokens[i].Type != TOKEN_COMMA {
			sortToken = append(sortToken, sortTokens[i])
			i++
		}
		newSortTokens = append(newSortTokens, sortToken)
	}

	// Parse each sortToken and create a SortNode
	for _, sortToken := range newSortTokens {
		sortNode := SortNode{SortDirection: "ASC"}
		for _, token := range sortToken {
			if queryNode.Type == TABLE || queryNode.Type == TABLE_NO_ID {
				if token.Type == TOKEN_METADATA {
					sortNode.Metadata = token.Value
				} else if strings.ToUpper(token.Value) == "DESC" {
					if sortNode.Metadata != "" {
						sortNode.SortDirection = "DESC"
					} else {
						return sortNodes, 0, fmt.Errorf("expected metadata field before DESC, got DESC")
					}
				} else if strings.ToUpper(token.Value) == "ASC" {
					if sortNode.Metadata != "" {
						sortNode.SortDirection = "ASC"
					} else {
						return sortNodes, 0, fmt.Errorf("expected metadata field before ASC, got ASC")
					}
				} else if token.Value == "GROUP" {
					gotGroup = true
					break
				} else if token.Value == "LIMIT" {
					gotLimit = true
					break
				} else {
					return sortNodes, 0, fmt.Errorf("expected metadata field or DESC/ASC, got %s", token.Value)
				}
			} else {
				if token.Type == TOKEN_METADATA {
					return sortNodes, 0, fmt.Errorf("metadata field not allowed in non-TABLE queries")
				} else if strings.ToUpper(token.Value) == "DESC" {
					sortNode.SortDirection = "DESC"
				} else if strings.ToUpper(token.Value) == "ASC" {
					sortNode.SortDirection = "ASC"
				} else if token.Value == "GROUP" {
					gotGroup = true
					break
				} else if token.Value == "LIMIT" {
					gotLimit = true
					break
				} else {
					return sortNodes, 0, fmt.Errorf("expected DESC/ASC, got %s", token.Value)
				}
			}
		}

		if gotGroup || gotLimit {
			break
		}

		sortNodes = append(sortNodes, sortNode)
	}

	// If querytype not table or table_no_id, take the last sort node and return just that
	if queryNode.Type != TABLE && queryNode.Type != TABLE_NO_ID {
		if len(sortNodes) == 0 {
			sortNode := SortNode{SortDirection: "ASC"}
			sortNodes = append(sortNodes, sortNode)
		} else {
			sortNodes = sortNodes[len(sortNodes)-1:]
		}

	}

	return sortNodes, i, nil
}

func parseWhereClause(tokens []Token) (*WhereNode, int, error) {
	whereNode := &WhereNode{}
	i := 0
	var currentCondition ConditionNode
	var logicalOp string
	var gotGroup bool
	var gotSort bool

	for i < len(tokens) && tokens[i].Value != "LIMIT" {
		switch tokens[i].Type {
		case TOKEN_GROUP:
			gotGroup = true
			break
		case TOKEN_SORT:
			gotSort = true
			break
		case TOKEN_METADATA:
			currentCondition.IsMetadata = true
			currentCondition.Field = tokens[i].Value
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
		if gotGroup || gotSort {
			break
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
	var rowsMetadata []map[string]interface{} // Store metadata for sorting
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

		// Apply WHERE conditions to filter rows
		if ast.Where != nil {
			if !applyConditions("", metadata, ast.Where.Conditions) {
				continue
			}
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
		rowsMetadata = append(rowsMetadata, metadata)

		// HACK: This is here to ensure metadata is printed when query is TABLE.
		// Fix this by returning metadata from parseMarkdownContent and this function.
		// When refactored, this should be inside executeQuery function.
		if printMetadataFlag {
			printMetadata([]Metadata{metadata})
		}
	}

	// Sort the rows based on multiple fields
	if len(ast.Sorts) > 0 {
		sort.Slice(rows, func(i, j int) bool {
			// Compare rows based on each sort criterion
			for _, sortNode := range ast.Sorts {
				// Find the column index for the metadata field
				colIndex := -1
				if sortNode.Metadata == "File" && ast.Type == TABLE {
					colIndex = 0
				} else {
					for idx, col := range ast.Columns {
						if col.Name == sortNode.Metadata {
							colIndex = idx
							if ast.Type == TABLE {
								colIndex++ // Adjust for File column
							}
							break
						}
					}
				}

				if colIndex == -1 {
					continue
				}

				// Get values to compare
				val1 := rows[i][colIndex]
				val2 := rows[j][colIndex]

				// Try to compare as numbers first
				num1, err1 := strconv.ParseFloat(val1, 64)
				num2, err2 := strconv.ParseFloat(val2, 64)

				var compareResult int
				if err1 == nil && err2 == nil {
					// Numeric comparison
					if num1 < num2 {
						compareResult = -1
					} else if num1 > num2 {
						compareResult = 1
					}
				} else {
					// String comparison
					compareResult = strings.Compare(val1, val2)
				}

				// If values are different, return the comparison result
				if compareResult != 0 {
					if sortNode.SortDirection == "DESC" {
						return compareResult > 0
					}
					return compareResult < 0
				}
			}
			return false // If all values are equal
		})
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

	content, metadataList, err := parseMarkdownFiles(ast.From, ast.Type)
	if err != nil {
		return "", err
	}

	// Sort content ASC or DESC
	// If it's not a table, ast.Sorts will only have one element
	// Here you can't sort by metadata, so just sort alphabetically
	// Use NaturalSort for sorting because it's nicer :)
	if len(ast.Sorts) > 0 {
		if ast.Sorts[0].SortDirection == "DESC" {
			sort.Slice(content, func(i, j int) bool {
				return NaturalSort(content[i], content[j])
			})
		} else {
			sort.Slice(content, func(i, j int) bool {
				sorted := NaturalSort(content[i], content[j])
				return !sorted
			})
		}
	}

	if ast.Where != nil {
		content, metadataList = filterContent(content, metadataList, ast.Where.Conditions)
	}

	if ast.GroupBy != "" {
		// This handles LIMIT too, that's why I can just return it
		return groupContent(content, metadataList, ast)
	}

	if ast.Limit >= 0 && ast.Limit < len(content) {
		content = content[:ast.Limit]
	}

	// HACK: This is here because metadata isn't returned far enough in the code.
	// Fix this by returning metadata from parseMarkdownContent.
	// When refactored, this should be inside executeQuery function.
	if printMetadataFlag {
		printMetadata(metadataList)
	}

	return strings.Join(content, "\n"), nil
}

func groupContent(content []string, metadataList []Metadata, ast *QueryNode) (string, error) {
	groups := make(map[string][]string)

	for i, item := range content {
		groupValue, ok := metadataList[i][ast.GroupBy]
		if !ok {
			groupValue = "Unknown"
		}
		groupKey := fmt.Sprintf("%v", groupValue)
		if ast.Limit > 0 && len(groups[groupKey]) >= ast.Limit {
			continue
		}
		groups[groupKey] = append(groups[groupKey], item)
	}

	var result strings.Builder
	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		return NaturalSort(keys[i], keys[j])
	})

	if ast.GroupLimit > 0 && len(keys) > ast.GroupLimit {
		keys = keys[:ast.GroupLimit]
	}

	for _, key := range keys {
		result.WriteString(fmt.Sprintf("- %s\n", key))
		for _, item := range groups[key] {
			switch ast.Type {
			case TASK, UNORDEREDLIST, ORDEREDLIST, PARAGRAPH, FENCEDCODE:
				result.WriteString(fmt.Sprintf("    %s\n", item))
			}
		}
		result.WriteString("\n")
	}

	return result.String(), nil
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
						if i, err := strconv.Atoi(value); err == nil {
							metadata[key] = i
						} else if b, err := strconv.ParseBool(value); err == nil {
							metadata[key] = b
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
	addFileMetadata(path, &metadata)

	// For TABLE and TABLE_NO_ID, no need to parse the content
	// Just return an empty slice for the content and the metadata
	if queryType == TABLE || queryType == TABLE_NO_ID {
		return []string{}, metadata, nil
	}

	// Strip YAML frontmatter from lines
	lines = stripYAMLFrontmatter(lines)

	var parsedContent []string
	switch queryType {
	case LIST:
		parsedContent = nil
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
	if !strings.Contains(line, "::") {
		return
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
	} else {
		parseMetadataPair(line, metadata)
	}
}

func parseMetadataPair(pair string, metadata Metadata) {
	parts := strings.SplitN(pair, "::", 2)
	if len(parts) == 2 {
		// INFO:
		// Key has to adhere to the following rules:
		// - No leading or trailing spaces
		// - Has to be lowercase
		// - Only contain alphanumeric characters and hyphens
		key := strings.TrimSpace(parts[0])
		key = strings.ToLower(key)
		key = strings.ReplaceAll(key, " ", "-")
		key = strings.ReplaceAll(key, "*", "")

		value := strings.TrimSpace(parts[1])

		if i, err := strconv.Atoi(value); err == nil {
			metadata[key] = i
		} else if b, err := strconv.ParseBool(value); err == nil {
			metadata[key] = b
		} else {
			metadata[key] = value
		}
	}
}

func addFileMetadata(path string, metadata *Metadata) {
	fileInfo, err := os.Stat(path)
	if err == nil {
		(*metadata)["file.folder"] = filepath.Base(filepath.Dir(path))
		(*metadata)["file.path"] = path
		(*metadata)["file.name"] = filepath.Base(path)
		(*metadata)["file.shortname"] = filepath.Base(path)[:len(filepath.Base(path))-3]
		(*metadata)["file.link"] = fmt.Sprintf("[%s](%s)", filepath.Base(path), path)
		(*metadata)["file.size"] = fileInfo.Size()
		(*metadata)["file.ctime"] = fileInfo.ModTime().Format(time.RFC3339)
		(*metadata)["file.cday"] = fileInfo.ModTime().Format("2006-01-02")
		(*metadata)["file.mtime"] = fileInfo.ModTime().Format(time.RFC3339)
		(*metadata)["file.mday"] = fileInfo.ModTime().Format("2006-01-02")
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
		trimmedLine := strings.TrimLeft(line, " \t")
		if isTaskListItem(trimmedLine) {
			tasks = append(tasks, line)
		}
	}
	return tasks
}

func parseParagraphs(lines []string) []string {
	var paragraphs []string
	var inCodeBlock bool
	var inList bool
	var emptyLineCount int

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
		if isOrderedListItem(line) {
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

		// Handle multiple empty lines
		if strings.TrimSpace(line) == "" {
			emptyLineCount++
			// Allow only the first empty line, skip the rest
			if emptyLineCount > 1 {
				continue
			}
		} else {
			emptyLineCount = 0 // Reset when a non-empty line is found
		}

		paragraphs = append(paragraphs, line)
	}

	// Remove the first element if it's an empty line
	if len(paragraphs) > 0 && strings.TrimSpace(paragraphs[0]) == "" {
		paragraphs = paragraphs[1:]
	}

	// Remove the last element if it's an empty line
	if len(paragraphs) > 0 && strings.TrimSpace(paragraphs[len(paragraphs)-1]) == "" {
		paragraphs = paragraphs[:len(paragraphs)-1]
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
		if isUnorderedListItem(trimmedLine) {
			if len(currentItem) > 0 {
				items = append(items, strings.Join(currentItem[:len(currentItem)-trailingEmptyLines], "\n"))
				currentItem = nil
				trailingEmptyLines = 0
			}
			currentItem = append(currentItem, line)
			inList = true
			indentLevel = len(line) - len(trimmedLine)
		} else if inList && (isUnorderedListItem(line) || len(line)-len(strings.TrimLeft(line, " ")) > indentLevel) {
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

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if isOrderedListItem(trimmedLine) {
			if inList && len(currentItem) > 0 {
				items = append(items, strings.Join(currentItem, "\n"))
				currentItem = nil
			}
			currentItem = append(currentItem, line)
			inList = true
		} else if inList && trimmedLine == "" {
			if len(currentItem) > 0 {
				items = append(items, strings.Join(currentItem, "\n"))
				currentItem = nil
			}
			inList = false
		} else if inList {
			currentItem = append(currentItem, line)
		} else {
			inList = false
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

func parseMarkdownFiles(paths []string, queryType QueryType) ([]string, []Metadata, error) {
	var results []string
	var metadataList []Metadata

	for _, path := range paths {
		if strings.HasPrefix(path, "~") {
			path = filepath.Join(os.Getenv("HOME"), path[1:])
		}

		path = os.ExpandEnv(path)
		fileInfo, err := os.Stat(path)
		if err != nil {
			return nil, nil, err
		}

		if fileInfo.IsDir() {
			err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && filepath.Ext(filePath) == ".md" {
					if queryType == LIST {
						results = append(results, "- "+filepath.Base(filePath))
						_, metadata, err := parseMarkdownContent(filePath, queryType)
						if err != nil {
							return err
						}
						metadataList = append(metadataList, metadata)
					} else {
						content, metadata, err := parseMarkdownContent(filePath, queryType)
						if err != nil {
							return err
						}
						results = append(results, content...)
						for range content {
							metadataList = append(metadataList, metadata)
						}
					}
				}
				return nil
			})
			if err != nil {
				return nil, nil, err
			}
		} else {
			if queryType == LIST {
				results = append(results, "- "+filepath.Base(path))
				_, metadata, err := parseMarkdownContent(path, queryType)
				if err != nil {
					return nil, nil, err
				}
				metadataList = append(metadataList, metadata)
			} else {
				content, metadata, err := parseMarkdownContent(path, queryType)
				if err != nil {
					return nil, nil, err
				}
				results = append(results, content...)
				for range content {
					metadataList = append(metadataList, metadata)
				}
			}
		}
	}

	return results, metadataList, nil
}

func isUnorderedListItem(line string) bool {
	trimmedLine := strings.TrimSpace(line)
	return (strings.HasPrefix(trimmedLine, "- ") || strings.HasPrefix(trimmedLine, "* ")) &&
		!strings.HasPrefix(trimmedLine, "- [ ]") &&
		!strings.HasPrefix(trimmedLine, "- [.]") &&
		!strings.HasPrefix(trimmedLine, "- [o]") &&
		!strings.HasPrefix(trimmedLine, "- [O]") &&
		!strings.HasPrefix(trimmedLine, "- [0]") &&
		!strings.HasPrefix(trimmedLine, "- [x]") &&
		!strings.HasPrefix(trimmedLine, "- [X]")
}

func isOrderedListItem(line string) bool {
	trimmedLine := strings.TrimSpace(line)
	if len(trimmedLine) == 0 || !unicode.IsNumber(rune(trimmedLine[0])) {
		return false
	}

	for i, char := range trimmedLine {
		if char == ' ' && i > 0 && trimmedLine[i-1] == '.' {
			return true
		}
		if !unicode.IsNumber(char) && char != '.' {
			return false
		}
	}
	return false
}

func isTaskListItem(line string) bool {
	trimmedLine := strings.TrimSpace(line)
	return strings.HasPrefix(trimmedLine, "- [ ]") ||
		strings.HasPrefix(trimmedLine, "- [x]") ||
		strings.HasPrefix(trimmedLine, "- [X]") ||
		strings.HasPrefix(trimmedLine, "- [.]") ||
		strings.HasPrefix(trimmedLine, "- [o]") ||
		strings.HasPrefix(trimmedLine, "- [O]") ||
		strings.HasPrefix(trimmedLine, "- [0]")
}

func applyConditions(item string, metadata Metadata, conditions []ConditionNode) bool {
	if len(conditions) == 0 {
		return true
	}

	result := true
	for i, condition := range conditions {
		conditionMet := false
		var fieldValue string

		if condition.IsMetadata {
			if value, ok := metadata[condition.Field]; ok {
				fieldValue = fmt.Sprintf("%v", value)
			}
		} else {
			fieldValue = item
		}

		switch condition.Function {
		case "CONTAINS":
			conditionMet = strings.Contains(strings.ToLower(fieldValue), strings.ToLower(condition.Value))
		case "IS":
			conditionMet = fieldValue == condition.Value
		case "CHECKED":
			isChecked := strings.Contains(fieldValue, "[x]") || strings.Contains(fieldValue, "[X]")
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

func filterContent(content []string, metadata []Metadata, conditions []ConditionNode) ([]string, []Metadata) {
	var filteredContent []string
	var filteredMetadata []Metadata

	for i, item := range content {
		if applyConditions(item, metadata[i], conditions) {
			filteredContent = append(filteredContent, item)
			filteredMetadata = append(filteredMetadata, metadata[i])
		}
	}

	return filteredContent, filteredMetadata
}

func readFromPipe() (string, error) {
	bytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func executeQuery(query string, showAST bool) (string, error) {
	tokens := Lex(query)
	ast, err := Parse(tokens)
	if err != nil {
		return "", fmt.Errorf("failed to parse query: %w", err)
	}

	if showAST {
		printTokens(tokens)
	}

	result, err := Interpret(ast)
	if err != nil {
		return "", fmt.Errorf("failed to execute query: %w", err)
	}

	return result, nil
}

func printMetadata(metadataList []Metadata) {
	for _, metadata := range metadataList {
		jsonData, err := json.MarshalIndent(metadata, "", "  ")
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(jsonData))
	}
}

func printTokens(tokens []Token) {
	type jsonToken struct {
		Type  string `json:"Type"`
		Value string `json:"Value"`
	}

	var jsonTokens []jsonToken

	for _, token := range tokens {
		jsonTokens = append(jsonTokens, jsonToken{
			Type:  TokenTypeNames[token.Type],
			Value: token.Value,
		})
	}

	jsonData, err := json.MarshalIndent(jsonTokens, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(jsonData))
}

var printMetadataFlag bool

func main() {
	var query string
	var err error
	versionFlag := flag.Bool("v", false, "print the version number")
	longVersionFlag := flag.Bool("version", false, "print the version number")

	ShowASTFlag := flag.Bool("ast", false, "print the whole AST before showing the results")
	flag.BoolVar(&printMetadataFlag, "metadata", false, "print metadata as JSON")

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
			if err != nil {
				fmt.Println("ERROR: Unable to read from pipe:", err)
				os.Exit(1)
			}
		}
	} else if query == "" {
		fmt.Println("No query provided. Use -q or --query to specify the query string.")
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
		os.Exit(1)
	}

	result, err := executeQuery(query, *ShowASTFlag)
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
