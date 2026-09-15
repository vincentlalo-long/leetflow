package template

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchesProblemNumber(t *testing.T) {
	tests := []struct {
		filename string
		num      string
		expected bool
	}{
		{"0001_two_sum.cpp", "1", true},
		{"0001_two_sum.cpp", "0001", true},
		{"1_two_sum.cpp", "1", true},
		{"0002_add_two_numbers.py", "2", true},
		{"0002_add_two_numbers.py", "1", false},
		{"invalid_file.go", "1", false},
	}

	for _, tt := range tests {
		got := MatchesProblemNumber(tt.filename, tt.num)
		if got != tt.expected {
			t.Errorf("MatchesProblemNumber(%q, %q) = %v, want %v", tt.filename, tt.num, got, tt.expected)
		}
	}
}

func TestGetLanguageByExtension(t *testing.T) {
	langs := NormalizeLanguages(nil)

	if got := GetLanguageByExtension(langs, ".cpp"); got != "cpp" {
		t.Errorf("GetLanguageByExtension(.cpp) = %q, want cpp", got)
	}
	if got := GetLanguageByExtension(langs, "py"); got != "python" {
		t.Errorf("GetLanguageByExtension(py) = %q, want python", got)
	}
	if got := GetLanguageByExtension(langs, ".go"); got != "go" {
		t.Errorf("GetLanguageByExtension(.go) = %q, want go", got)
	}
	if got := GetLanguageByExtension(langs, ".java"); got != "java" {
		t.Errorf("GetLanguageByExtension(.java) = %q, want java", got)
	}
	if got := GetLanguageByExtension(langs, ".rs"); got != "rust" {
		t.Errorf("GetLanguageByExtension(.rs) = %q, want rust", got)
	}
	if got := GetLanguageByExtension(langs, ".unknown"); got != "" {
		t.Errorf("GetLanguageByExtension(.unknown) = %q, want empty", got)
	}
}

func TestStripHTMLTags(t *testing.T) {
	html := "<p>Given an array of integers <code>nums</code>.</p>"
	expected := "Given an array of integers nums."
	got := StripHTMLTags(html)
	if got != expected {
		t.Errorf("StripHTMLTags() = %q, want %q", got, expected)
	}
}

func TestDecodeHTMLEntities(t *testing.T) {
	got := DecodeHTMLEntities("2 &lt;= nums.length &lt;= 104 &amp; x &quot;q&quot; &#39;y&#39; a&nbsp;b")
	want := `2 <= nums.length <= 104 & x "q" 'y' a b`
	if got != want {
		t.Errorf("DecodeHTMLEntities() = %q, want %q", got, want)
	}
}

func TestFormatDescriptionMarkdown(t *testing.T) {
	html := `<p>Given an array of integers <code>nums</code>&nbsp;and an integer <code>target</code>.</p>

<p>&nbsp;</p>
<p><strong class="example">Example 1:</strong></p>
<pre><strong>Input:</strong> nums = [2,7,11,15], target = 9 <strong>Output:</strong> [0,1] <strong>Explanation:</strong> Because nums[0] + nums[1] == 9, we return [0, 1].</pre>

<p><strong>Constraints:</strong></p>
<ul>
<li><code>2 &lt;= nums.length &lt;= 10<sup>4</sup></code></li><li><code>-10<sup>9</sup> &lt;= nums[i] &lt;= 10<sup>9</sup></code></li><li><code>-10<sup>9</sup> &lt;= target &lt;= 10<sup>9</sup></code></li><li><code>Only one valid answer exists.</code></li>
</ul>

<p>&nbsp;</p>
Follow-up:&nbsp;Can you come up with an algorithm that is less than O(n<sup>2</sup>)&nbsp;time complexity?`

	got := FormatDescriptionMarkdown(html)
	want := `Given an array of integers nums and an integer target.

Example 1:

Input: nums = [2,7,11,15], target = 9
Output: [0,1]
Explanation: Because nums[0] + nums[1] == 9, we return [0, 1].

Constraints:

2 <= nums.length <= 10^4
-10^9 <= nums[i] <= 10^9
-10^9 <= target <= 10^9
Only one valid answer exists.

Follow-up: Can you come up with an algorithm that is less than O(n^2) time complexity?`
	if got != want {
		t.Errorf("FormatDescriptionMarkdown() = %q\nwant %q", got, want)
	}
}

func TestGenerateRootReadme(t *testing.T) {
	tmpDir := t.TempDir()

	os.MkdirAll(filepath.Join(tmpDir, "array", "1-two-sum"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "array", "1-two-sum", "README.md"), []byte("# [1. Two Sum](https://leetcode.com/problems/two-sum/)\n\n- **Difficulty:** Easy\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "array", "1-two-sum", "1_two_sum.cpp"), []byte("int main(){}"), 0644)

	readmePath, count, err := GenerateRootReadme(tmpDir, nil)
	if err != nil {
		t.Fatalf("GenerateRootReadme failed: %v", err)
	}
	if count != 1 {
		t.Errorf("GenerateRootReadme count = %d, want 1", count)
	}
	if _, err := os.Stat(readmePath); err != nil {
		t.Errorf("generated README missing: %v", err)
	}
}
