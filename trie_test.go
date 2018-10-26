package main

import (
	"testing"
)

func TestFind(t *testing.T) {
	trie := NewTrie()

	trie.Add("com.baidu")

	if !trie.Find("com.baidu") || !trie.Find("com.baidu.www") {
		t.Fatal("Could not find node")
	}
}

func TestReverse(t *testing.T) {
	if reverse("abcde") != "edcba" {
		t.Error("invalid reverse")
	}
}
