//go:build nativegui

package nativegui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func TestMakeEntriesPageScrollableDisablesInnerScroller(t *testing.T) {
	entry := widget.NewEntry()

	makeEntriesPageScrollable(entry)

	if entry.Wrapping != fyne.TextWrapOff {
		t.Fatalf("Wrapping = %v, want TextWrapOff", entry.Wrapping)
	}
	if entry.Scroll != container.ScrollNone {
		t.Fatalf("Scroll = %v, want ScrollNone", entry.Scroll)
	}
}

func TestSectionCardHasUsefulMinimumSize(t *testing.T) {
	card := sectionCard("起動確認", widget.NewLabel("content"))
	if card.MinSize().Width <= 0 || card.MinSize().Height <= 0 {
		t.Fatalf("section card has invalid minimum size: %v", card.MinSize())
	}
}
