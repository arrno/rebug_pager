package rebugpager

import (
	"errors"
	"strings"
	"time"
)

type ElementType int

const (
	Paragraph ElementType = iota
	Heading
	List
)

func (e ElementType) ToString() string {
	switch e {
	case Heading:
		return "h"
	case List:
		return "ol"
	}
	return "p"
}
func ElementFromString(s string) (ElementType, bool) {
	switch s {
	case "p":
		return Paragraph, true
	case "h":
		return Heading, true
	case "ol":
		return List, true
	}
	return Paragraph, false
}

type UserMessage struct {
	DocID   string
	Element ElementType
	Content string
}

func (m *UserMessage) ContentToData() any {
	if m.Element == List {
		return strings.Split(m.Content, "\n")
	}
	return m.Content
}

type DocElement struct {
	ElementType    string `firestore:"Type"`
	ElementContent any    `firestore:"Content"`
}

type UserDoc struct {
	CreatedAt time.Time    `firestore:"createdAt"`
	UpdatedAt time.Time    `firestore:"updatedAt"`
	Elements  []DocElement `firestore:"elements"`
}

func ParseMessage(content string) (UserMessage, error) {
	message := UserMessage{}
	content = strings.TrimLeft(content, " ")
	words := strings.Split(content, " ")
	if len(words) < 2 {
		return message, errors.New("Invalid message format.")
	}
	docID := words[0]
	if len(docID) != 6 {
		return message, errors.New("Invalid docID format.")
	}
	message.DocID = docID
	element, found := ElementFromString(words[1])
	message.Element = element
	if found && len(words) >= 3 {
		// element type specified. Content starts on third word.
		message.Content = strings.Join(words[2:], " ")
	} else if !found {
		// element type not specified. Content starts on second word.
		message.Content = strings.Join(words[1:], " ")
	}
	return message, nil
}

func MergeDoc(doc UserDoc, message UserMessage) UserDoc {
	doc.UpdatedAt = time.Now()
	doc.Elements = append(doc.Elements, DocElement{
		ElementType:    message.Element.ToString(),
		ElementContent: message.ContentToData(),
	})
	return doc
}
