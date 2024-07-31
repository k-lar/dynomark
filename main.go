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
)

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
)

type Condition struct {
	Field    string
	Operator string
	Value    string
	IsOr     bool
}

type Query struct {
	Type       QueryType
	From       []string
	Fields     []string
	Conditions []Condition
	Limit      int
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

		// Try to convert the value to a boolean or an integer if possible
		// Try to convert the value to a boolean or an integer if possible
		// Try to convert the value to a boolean or an integer if possible
		// Try to convert the value to a boolean or an integer if possible
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
	if strings.HasPrefix(line, "**") && strings.Contains(line, "::") {
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

func parseUnorderedLists(lines []string) []string {
	var items []string
	var currentItem []string
	inList := false
	indentLevel := 0

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "-") && !strings.HasPrefix(trimmedLine, "- [") && trimmedLine != "---" {
			if len(currentItem) > 0 {
				items = append(items, strings.Join(currentItem, "\n"))
				currentItem = nil
			}
			currentItem = append(currentItem, line)
			inList = true
			indentLevel = len(line) - len(trimmedLine)
		} else if inList && (strings.HasPrefix(trimmedLine, "-") || len(line)-len(strings.TrimLeft(line, " ")) > indentLevel || trimmedLine == "") {
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
	if len(conditions) == 0 {
		return true
	}

	var result bool
	for i, condition := range conditions {
		conditionMet := false
		switch condition.Operator {
		case "CONTAINS":
			conditionMet = strings.Contains(strings.ToLower(item), strings.ToLower(condition.Value))
		case "NOT CONTAINS":
			conditionMet = !strings.Contains(strings.ToLower(item), strings.ToLower(condition.Value))
		case "=":
			if condition.Field == "status" && queryType == TASK {
				isChecked := strings.Contains(item, "[x]") || strings.Contains(item, "[X]")
				conditionMet = condition.Value == "checked" && isChecked
			}
		case "!=":
			if condition.Field == "status" && queryType == TASK {
				isChecked := strings.Contains(item, "[x]") || strings.Contains(item, "[X]")
				conditionMet = condition.Value == "checked" && !isChecked
			}
		}

		if i == 0 {
			result = conditionMet
		} else if condition.IsOr {
			result = result || conditionMet
		} else {
			result = result && conditionMet
		}
	}

	return result
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

func main() {
	var query string
	var err error

	flag.StringVar(&query, "query", "", "The query string to be processe")
	flag.StringVar(&query, "q", "", "The query string to be processed (shorthand)")

	flag.Parse()

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

	fmt.Println(result)
}

func parseConditions(words []string) ([]Condition, error) {
	var conditions []Condition
	var current Condition
	expectingValue := false
	notFlag := false
	orFlag := false

	for _, word := range words {
		upperWord := strings.ToUpper(word)
		if upperWord == "AND" {
			continue
		}
		if upperWord == "OR" {
			orFlag = true
			continue
		}
		if expectingValue {
			current.Value = strings.Trim(word, "\"")
			current.IsOr = orFlag
			conditions = append(conditions, current)
			current = Condition{}
			expectingValue = false
			notFlag = false
			orFlag = false
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
			current.IsOr = orFlag
			conditions = append(conditions, current)
			current = Condition{}
			orFlag = false
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
	case "TABLE":
		q.Type = TABLE
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
	for _, source := range sources {
		if strings.ToUpper(source) == "AND" {
			continue
		}

		if strings.HasSuffix(source, ",") {
			source = source[:len(source)-1]
			q.From = append(q.From, strings.Trim(source, "\""))
			continue
		}

		q.From = append(q.From, strings.Trim(source, "\""))
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
