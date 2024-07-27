package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type QueryType string

const (
	LIST          QueryType = "LIST"
	TASK          QueryType = "TASK"
	PARAGRAPH     QueryType = "PARAGRAPH"
	ORDEREDLIST   QueryType = "ORDEREDLIST"
	UNORDEREDLIST QueryType = "UNORDEREDLIST"
	FENCEDCODE    QueryType = "FENCEDCODE"
)

type Condition struct {
	Field    string
	Operator string
	Value    string
}

type Query struct {
	Type       QueryType
	From       []string
	Fields     []string
	Conditions []Condition
	Limit      int
}

func parseMarkdownContent(path string, queryType QueryType) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	switch queryType {
	case TASK:
		return parseTasks(lines), nil
	case PARAGRAPH:
		return parseParagraphs(lines), nil
	case ORDEREDLIST:
		return parseOrderedLists(lines), nil
	case UNORDEREDLIST:
		return parseUnorderedLists(lines), nil
	case FENCEDCODE:
		return parseFencedCode(lines), nil
	default:
		return nil, fmt.Errorf("unsupported query type: %s", queryType)
	}
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
	var currentParagraph []string

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			if len(currentParagraph) > 0 {
				paragraphs = append(paragraphs, strings.Join(currentParagraph, " "))
				currentParagraph = nil
			}
		} else if !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "1.") {
			currentParagraph = append(currentParagraph, strings.TrimSpace(line))
		}
	}

	if len(currentParagraph) > 0 {
		paragraphs = append(paragraphs, strings.Join(currentParagraph, " "))
	}

	return paragraphs
}

func parseOrderedLists(lines []string) []string {
	var orderedLists []string
	var currentList []string

	for _, line := range lines {
		if matched, _ := regexp.MatchString(`^\d+\.`, strings.TrimSpace(line)); matched {
			currentList = append(currentList, strings.TrimSpace(line))
		} else if strings.TrimSpace(line) == "" {
			if len(currentList) > 0 {
				orderedLists = append(orderedLists, strings.Join(currentList, "\n"))
				currentList = nil
			}
		}
	}

	if len(currentList) > 0 {
		orderedLists = append(orderedLists, strings.Join(currentList, "\n"))
	}

	return orderedLists
}

func parseUnorderedLists(lines []string) []string {
	var unorderedLists []string
	var currentList []string

	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "-") && !strings.HasPrefix(strings.TrimSpace(line), "- [") {
			currentList = append(currentList, strings.TrimSpace(line))
		} else if strings.TrimSpace(line) == "" {
			if len(currentList) > 0 {
				unorderedLists = append(unorderedLists, strings.Join(currentList, "\n"))
				currentList = nil
			}
		}
	}

	if len(currentList) > 0 {
		unorderedLists = append(unorderedLists, strings.Join(currentList, "\n"))
	}

	return unorderedLists
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
		fileInfo, err := os.Stat(path)
		if err != nil {
			return nil, err
		}

		if fileInfo.IsDir() {
			files, err := filepath.Glob(filepath.Join(path, "*.md"))
			if err != nil {
				return nil, err
			}
			for _, file := range files {
				results = append(results, filepath.Base(file))
			}
		} else if queryType == LIST {
			results = append(results, filepath.Base(path))
		} else {
			content, err := parseMarkdownContent(path, queryType)
			if err != nil {
				return nil, err
			}
			results = append(results, content...)
		}
	}

	return results, nil
}

func parseMarkdownFile(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var results []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "- [ ]") || strings.HasPrefix(line, "- [x]") || strings.HasPrefix(line, "- [X]") {
			results = append(results, line)
		}
	}

	return results, scanner.Err()
}

func applyConditions(item string, queryType QueryType, conditions []Condition) bool {
	for _, condition := range conditions {
		switch condition.Operator {
		case "CONTAINS":
			if !strings.Contains(item, condition.Value) {
				return false
			}
		case "NOT CONTAINS":
			if strings.Contains(item, condition.Value) {
				return false
			}
		case "=":
			if condition.Field == "status" && queryType == TASK {
				isChecked := strings.Contains(item, "[x]") || strings.Contains(item, "[X]")
				if condition.Value == "checked" && !isChecked {
					return false
				}
			}
		case "!=":
			if condition.Field == "status" && queryType == TASK {
				isChecked := strings.Contains(item, "[x]") || strings.Contains(item, "[X]")
				if condition.Value == "checked" && isChecked {
					return false
				}
			}
		}
	}
	return true
}

func filterContent(content []string, queryType QueryType, conditions []Condition) []string {
	var filteredContent []string

	for _, item := range content {
		if applyConditions(item, queryType, conditions) {
			filteredContent = append(filteredContent, item)
		}
	}

	return filteredContent
}

func executeQueryType(query Query) (string, error) {
	content, err := parseMarkdownFiles(query.From, query.Type)
	if err != nil {
		return "", err
	}

	filteredContent := filterContent(content, query.Type, query.Conditions)

	// Apply LIMIT
	if query.Limit >= 0 && query.Limit < len(filteredContent) {
		filteredContent = filteredContent[:query.Limit]
	}

	return strings.Join(filteredContent, "\n"), nil
}

func readFromPipe() (string, error) {
	bytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func readFromInteractive() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Enter your DQL query (press Ctrl+D to execute):")

	var queryLines []string
	for scanner.Scan() {
		line := scanner.Text()
		queryLines = append(queryLines, line)
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.Join(queryLines, "\n"), nil
}

func main() {
	var query string
	var err error

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		// Data is being piped to stdin
		query, err = readFromPipe()
	} else {
		// Interactive mode
		query, err = readFromInteractive()
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

	fmt.Println(result)
}

func parseConditions(words []string) ([]Condition, error) {
	var conditions []Condition
	var current Condition
	expectingValue := false
	notFlag := false

	for _, word := range words {
		upperWord := strings.ToUpper(word)
		if upperWord == "AND" {
			continue
		}
		if expectingValue {
			current.Value = strings.Trim(word, "\"")
			conditions = append(conditions, current)
			current = Condition{}
			expectingValue = false
			notFlag = false
		} else if upperWord == "NOT" {
			notFlag = true
		} else if upperWord == "CONTAINS" {
			if notFlag {
				current.Operator = "NOT CONTAINS"
				notFlag = false
			} else {
				current.Operator = "CONTAINS"
			}
			expectingValue = true
		} else if upperWord == "CHECKED" {
			current.Field = "status"
			if notFlag {
				current.Operator = "!="
				notFlag = false
			} else {
				current.Operator = "="
			}
			current.Value = "checked"
			conditions = append(conditions, current)
			current = Condition{}
		} else {
			current.Field = word
		}
	}

	return conditions, nil
}

func parseQuery(query string) (Query, error) {
	words := strings.Fields(query)
	if len(words) < 3 {
		return Query{}, fmt.Errorf("invalid query: must have at least query type and FROM clause")
	}

	q := Query{Limit: -1} // -1 means no limit

	// Parse query type
	switch strings.ToUpper(words[0]) {
	case "LIST":
		q.Type = LIST
	case "TASK":
		q.Type = TASK
	case "PARAGRAPH":
		q.Type = PARAGRAPH
	case "ORDEREDLIST":
		q.Type = ORDEREDLIST
	case "UNORDEREDLIST":
		q.Type = UNORDEREDLIST
	case "FENCEDCODE":
		q.Type = FENCEDCODE
	default:
		return Query{}, fmt.Errorf("invalid query type: %s", words[0])
	}

	// Find the FROM, WHERE, and LIMIT clauses
	fromIndex := -1
	whereIndex := -1
	limitIndex := -1
	for i, word := range words {
		switch strings.ToUpper(word) {
		case "FROM":
			fromIndex = i
		case "WHERE":
			whereIndex = i
		case "LIMIT":
			limitIndex = i
		}
	}

	if fromIndex == -1 {
		return Query{}, fmt.Errorf("invalid query: missing FROM clause")
	}

	// Parse sources
	var sourceEnd int
	if whereIndex != -1 {
		sourceEnd = whereIndex
	} else if limitIndex != -1 {
		sourceEnd = limitIndex
	} else {
		sourceEnd = len(words)
	}

	sources := words[fromIndex+1 : sourceEnd]
	for i, source := range sources {
		if strings.ToUpper(source) == "AND" {
			continue
		}
		q.From = append(q.From, strings.Trim(source, "\""))
		if i+1 < len(sources) && strings.ToUpper(sources[i+1]) != "AND" {
			break
		}
	}

	// Parse fields (if any)
	if fromIndex > 1 {
		q.Fields = words[1:fromIndex]
	}

	// Parse WHERE conditions
	if whereIndex != -1 {
		endConditions := limitIndex
		if endConditions == -1 {
			endConditions = len(words)
		}
		conditions, err := parseConditions(words[whereIndex+1 : endConditions])
		if err != nil {
			return Query{}, err
		}
		q.Conditions = conditions
	}

	// Parse LIMIT
	if limitIndex != -1 {
		if limitIndex == len(words)-1 {
			return Query{}, fmt.Errorf("invalid query: LIMIT clause requires a value")
		}
		limit, err := strconv.Atoi(words[limitIndex+1])
		if err != nil {
			return Query{}, fmt.Errorf("invalid LIMIT value: %s", words[limitIndex+1])
		}
		q.Limit = limit
	}

	return q, nil
}

func executeQuery(query string) (string, error) {
	parsedQuery, err := parseQuery(query)
	if err != nil {
		return "", fmt.Errorf("failed to parse query: %w", err)
	}

	result, err := executeQueryType(parsedQuery)
	if err != nil {
		return "", fmt.Errorf("failed to execute query: %w", err)
	}

	return result, nil
}