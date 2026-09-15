package template

import (
	"fmt"
	"sort"
	"strings"
)

type LanguageInfo struct {
	Label string `json:"label"`
	Ext   string `json:"ext"`
}

var DefaultLanguages = map[string]LanguageInfo{
	"cpp":        {Label: "C++", Ext: "cpp"},
	"c":          {Label: "C", Ext: "c"},
	"python":     {Label: "Python", Ext: "py"},
	"go":         {Label: "Go", Ext: "go"},
	"java":       {Label: "Java", Ext: "java"},
	"rust":       {Label: "Rust", Ext: "rs"},
	"javascript": {Label: "JavaScript", Ext: "js"},
	"typescript": {Label: "TypeScript", Ext: "ts"},
	"csharp":     {Label: "C#", Ext: "cs"},
}

var LeetCodeLangMap = map[string]string{
	"cpp":        "cpp",
	"c":          "c",
	"python":     "python3",
	"go":         "golang",
	"java":       "java",
	"rust":       "rust",
	"javascript": "javascript",
	"typescript": "typescript",
	"csharp":     "csharp",
}

var (
	LevelToName = map[int]string{1: "Easy", 2: "Medium", 3: "Hard"}
	DiffToLevel = map[string]int{"Easy": 1, "Medium": 2, "Hard": 3}
)

func NormalizeLanguages(cfgLanguages map[string]interface{}) map[string]LanguageInfo {
	if len(cfgLanguages) == 0 {
		result := make(map[string]LanguageInfo)
		for k, v := range DefaultLanguages {
			result[k] = v
		}
		return result
	}

	normalized := make(map[string]LanguageInfo)
	for key, value := range cfgLanguages {
		label := ""
		ext := ""
		if v, ok := DefaultLanguages[key]; ok {
			label = v.Label
		} else {
			label = strings.Title(key)
		}

		switch v := value.(type) {
		case map[string]interface{}:
			if e, ok := v["ext"]; ok {
				ext = fmt.Sprint(e)
			}
			if l, ok := v["label"]; ok {
				label = fmt.Sprint(l)
			}
		default:
			ext = fmt.Sprint(v)
		}

		ext = strings.TrimLeft(ext, ".")
		if ext == "" {
			continue
		}
		normalized[key] = LanguageInfo{Label: label, Ext: ext}
	}

	if len(normalized) == 0 {
		result := make(map[string]LanguageInfo)
		for k, v := range DefaultLanguages {
			result[k] = v
		}
		return result
	}

	return normalized
}

func GetDefaultLanguageKey(languages map[string]LanguageInfo, defaultKey string) string {
	if _, ok := languages[defaultKey]; ok {
		return defaultKey
	}
	for k := range languages {
		return k
	}
	return "cpp"
}

func GetLanguageChoices(languages map[string]LanguageInfo, defaultKey string) ([]string, map[string]string) {
	var keys []string
	for k := range languages {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	choices := make([]string, 0)
	mapping := make(map[string]string)
	for _, k := range keys {
		info := languages[k]
		label := fmt.Sprintf("%s (%s)", info.Label, info.Ext)
		if k == defaultKey {
			label += " (default)"
		}
		choices = append(choices, label)
		mapping[label] = k
	}
	return choices, mapping
}

func GetLanguageByExtension(languages map[string]LanguageInfo, ext string) string {
	ext = strings.TrimLeft(ext, ".")
	ext = strings.ToLower(ext)
	for k, info := range languages {
		if strings.ToLower(info.Ext) == ext {
			return k
		}
	}
	return ""
}

func BuildProblemHeader(languageKey, problemNum, title, link, difficulty, tagsStr, structure string) string {
	lines := []string{
		fmt.Sprintf("LeetCode Problem %s: %s", problemNum, title),
		fmt.Sprintf("Link: %s", link),
		fmt.Sprintf("Difficulty: %s", difficulty),
		fmt.Sprintf("Tags: %s", tagsStr),
		fmt.Sprintf("Data Structure: %s", structure),
	}
	header := strings.Join(lines, "\n")

	if languageKey == "python" {
		return fmt.Sprintf("\"\"\"\n%s\n\"\"\"\n\n", header)
	}
	return fmt.Sprintf("/*\n%s\n*/\n\n", header)
}

func BuildProblemTemplateWithSnippet(languageKey, problemNum, title, link, difficulty, tagsStr, structure, codeSnippet string) string {
	header := BuildProblemHeader(languageKey, problemNum, title, link, difficulty, tagsStr, structure)
	if strings.TrimSpace(codeSnippet) != "" {
		return header + strings.TrimSpace(codeSnippet) + "\n"
	}
	return BuildProblemTemplate(languageKey, problemNum, title, link, difficulty, tagsStr, structure)
}

func BuildProblemTemplate(languageKey, problemNum, title, link, difficulty, tagsStr, structure string) string {
	header := BuildProblemHeader(languageKey, problemNum, title, link, difficulty, tagsStr, structure)

	switch languageKey {
	case "python":
		return fmt.Sprintf(`%sfrom typing import List

class Solution:
    pass

if __name__ == "__main__":
    print("Test cases go here!")
`, header)
	case "go":
		return fmt.Sprintf(`%spackage main

import "fmt"

func main() {
    fmt.Println("Test cases go here!")
}
`, header)
	case "c":
		return fmt.Sprintf(`%s#include <stdio.h>

int main() {
    printf("Test cases go here!\n");
    return 0;
}
`, header)
	case "cpp":
		return fmt.Sprintf(`%s#include <iostream>
#include <vector>
#include <string>

using namespace std;

class Solution {
public:
    // Write your solution method here, e.g.:
    // int twoSum(vector<int>& nums, int target) { ... }
};

int main() {
    return 0;
}
`, header)
	case "java":
		return fmt.Sprintf(`%spublic class Solution {
    public static void main(String[] args) {
        System.out.println("Test cases go here!");
    }
}
`, header)
	case "rust":
		return fmt.Sprintf(`%sfn main() {
    println!("Test cases go here!");
}
`, header)
	case "javascript", "typescript":
		return fmt.Sprintf(`%sconsole.log("Test cases go here!");
`, header)
	case "csharp":
		return fmt.Sprintf(`%susing System;

class Program {
    static void Main(string[] args) {
        Console.WriteLine("Test cases go here!");
    }
}
`, header)
	default:
		return fmt.Sprintf(`%s#include <iostream>
#include <vector>
#include <string>

using namespace std;

// class Solution {
// public:
//     
// };

int main() {
    // Solution sol;
    cout << "Test cases go here!" << endl;
    return 0;
}
`, header)
	}
}

func BuildSolutionBlock(languageKey string, solNum int, method, time, space, code string) string {
	if languageKey == "python" {
		return fmt.Sprintf(`

# ================== Solution %d ==================
"""
Method: %s
Time Complexity: %s
Space Complexity: %s
"""

%s
`, solNum, method, time, space, code)
	}

	return fmt.Sprintf(`

// ================== Solution %d ==================
/*
Method: %s
Time Complexity: %s
Space Complexity: %s
*/

%s
`, solNum, method, time, space, code)
}
