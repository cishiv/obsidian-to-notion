// Parse and convert a trivialised markdown spec to an opinionated JSON format
package markdown

import (
	"fmt"
	"obsidian-to-notion/utils"
	"regexp"
	"strings"
)

type Pair[T, U any] struct {
	T any
	U any
}

type Match struct {
	name    string
	line    string
	indices [][]int
}

type LinkedLine struct {
	LineType      string   `json:"lineType"`
	ResultStrings []string `json:"resultStrings"`
	Safe          bool     `json:"safe"`
}

// paragraph is fallback
var blockpatterns = map[string]string{
	// These matchers _should_ work because we're matching blocks line-by-line
	// recurrence capture doesn't bubble up the way I expect, this needs to be handled programatically
	// if something is an ###, it can't be ## or #
	"heading1":   "^#{1}",
	"heading2":   "^#{2}",
	"heading3":   "^#{3}",
	"linebreak":  "\n",
	"blockquote": "^>",          // support only single level blocks (blocks can contain other elements).
	"codeblock":  "^```",        // this one needs depth.
	"ul":         `^\-|^\*|^\+`, // this one is debatable.
	"ol":         `^[0-9]*\.`,
	"hr":         "^---",
}

// excludes images.
var spanpatterns = map[string]string{
	// for em and strong, we need to count, but regex works.
	"em":         "_.*_",
	"link":       `\[.*\]\(.*\)`,
	"strong":     `\*.*\*`,
	"inlinecode": "`.*`",
	"img":        "TODO", // TODO
}

// do not use - just for context / remembering that these might be useful things.
var miscpatterns = map[string]string{
	"escape":   "TODO", // TODO
	"autolink": "TODO", // TODO
}

// func Parse(markdownString string) Block {
// 	return Block{
// 		name:     "body",
// 		content:  "",
// 		children: [
// 			Block{},
// 			Block{}
// 		],
// 	}
// }

func fileToSlice(file string) []string {
	lines := strings.Split(file, "\n")
	return lines
}

func parse(fileSlice []string) string {
	// Parse line by line
	maxPtr := len(fileSlice) - 1
	for ptr := 0; ptr <= maxPtr; ptr++ {
		if fileSlice[ptr] == "" {
			continue
		}
		// fmt.Println(fileSlice[ptr])
	}
	return ""
}

func apply(patterns map[string]regexp.Regexp, line string) []Match {
	// fmt.Println(line)
	var matches []Match
	for name, pattern := range patterns {
		matched := pattern.MatchString(line)
		indices := pattern.FindAllIndex([]byte(line), -1)
		if matched {
			matches = append(matches, Match{
				name:    name,
				line:    line,
				indices: indices,
			})
		}
	}
	return matches
}

func compilePatterns() map[string]regexp.Regexp {
	fmt.Println("compile block patterns")
	patterns := make(map[string]regexp.Regexp)
	for name, pattern := range blockpatterns {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			panic(err)
		}
		patterns[name] = *compiled.Copy()
		fmt.Println("Complied Name:", name, "=>", "Pattern:", pattern)
	}

	fmt.Println("compile span patterns")
	for name, pattern := range spanpatterns {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			panic(err)
		}
		patterns[name] = *compiled.Copy()
		fmt.Println("Compiled Name:", name, "=>", "Pattern:", pattern)
	}
	return patterns
}

// TODO: Refactor once working
func precompute(matchMap map[int][]Match) map[int]LinkedLine {
	// TODO: Key arrays should be constant
	spankeys := make([]string, len(spanpatterns))
	i := 0
	for k := range spanpatterns {
		spankeys[i] = k
		i++
	}

	blockkeys := make([]string, len(blockpatterns))
	j := 0
	for k := range blockpatterns {
		blockkeys[j] = k
		j++
	}

	// create computedLines map and preallocate memory for each line
	computedLines := make(map[int]LinkedLine, len(matchMap))
	// precompute the things needed to determine hierarchies
	for idx, results := range matchMap {
		// fmt.Println(line, results)
		resultStrings := utils.Map(results, func(t Match) string { return t.name })
		// if the previous thing is not a newline and the current thing doesn't contain lone block patterns, this thing is likely in a paragraph
		safe := true
		for _, m := range resultStrings {
			// does the current thing contain a block pattern without a span pattern?
			if !utils.Contains(spankeys, m) &&
				utils.Contains(blockkeys, m) {
				safe = !safe
				break
			}
		}
		computedLines[idx] = LinkedLine{
			ResultStrings: resultStrings,
			Safe:          safe,
		}
	}

	// for the lines computed, we can look back and forth based on the indices
	prev := "start"
	for idx, line := range computedLines {
		// we don't need to _store_ previous and next for a line, we only need it to determine if the line is
		// paragrah_start
		// paragraph_end
		// paragraph_internal
		// block
		next := ""
		if idx != len(computedLines)-1 {
			nextResults := computedLines[idx+1].ResultStrings
			if len(nextResults) == 1 {
				if nextResults[0] == "newline" {
					next = nextResults[0]
				}
			} else {
				if idx == 0 {
					// we might have spans IN blocks, just how we can have "block-esque" things in spans (this will likely require an index check)
					// if idx ALL spans > idx ALL blocks then BLOCK else SPAN
					if !utils.ContainsAny(spankeys, line.ResultStrings) &&
						utils.ContainsAny(blockkeys, line.ResultStrings) &&
						utils.ContainsAny(blockkeys, nextResults) &&
						!utils.ContainsAny(spankeys, nextResults) {
						// this line is a block, it's likely the next line is going to be a paragraph_start if it not a new line or block, so check if its a block
						next = "block"
					} else {
						next = "span"
					}
				}
			}
		}

		if (prev != "block" && prev != "newline") && computedLines[idx].Safe {
			// we're in a paragraph
			copied := LinkedLine{
				ResultStrings: computedLines[idx].ResultStrings,
				Safe:          computedLines[idx].Safe,
				LineType:      "paragraph_internal",
			}
			computedLines[idx] = copied
		} else {
			if prev == "newline" && computedLines[idx].Safe {
				// we're starting a paragraph
				copied := LinkedLine{
					ResultStrings: computedLines[idx].ResultStrings,
					Safe:          computedLines[idx].Safe,
					LineType:      "paragraph_start",
				}
				computedLines[idx] = copied
			} else {
				// this is something else, no paragraph
				if next == "newline" || next == "block" {
					copied := LinkedLine{
						ResultStrings: computedLines[idx].ResultStrings,
						Safe:          computedLines[idx].Safe,
						LineType:      "paragraph_end",
					}
					computedLines[idx] = copied
				} else {
					copied := LinkedLine{
						ResultStrings: computedLines[idx].ResultStrings,
						Safe:          computedLines[idx].Safe,
						LineType:      "block_start_end",
					}
					computedLines[idx] = copied
				}
			}
		}

		// change prev
		if len(line.ResultStrings) == 1 {
			if line.ResultStrings[0] == "newline" {
				prev = line.ResultStrings[0]
			}
			// if the current line is not a span in any way, but is a block, then it is a block
			// otherwise, it must be a span (?)
		} else if !utils.ContainsAny(spankeys, line.ResultStrings) &&
			utils.ContainsAny(blockkeys, line.ResultStrings) {
			prev = "block"
		} else {
			prev = "span"
		}

	}
	return computedLines
}
