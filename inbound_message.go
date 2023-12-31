package rebugpager

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	switch strings.ToLower(s) {
	case "p":
		return Paragraph, true
	case "h":
		return Heading, true
	case "ol":
		return List, true
	}
	return Paragraph, false
}

type ElementMessage struct {
	DocID   string
	Element ElementType
	Content string
}

func (m *ElementMessage) ContentToData() any {
	if m.Element == List {
		return strings.Split(m.Content, "\n")
	}
	return m.Content
}

type DocElement struct {
	ElementType    string `firestore:"Type" json:"Type"`
	ElementContent any    `firestore:"Content" json:"Content"`
}

type UserDoc struct {
	CreatedAt  time.Time    `firestore:"createdAt"`
	UpdatedAt  time.Time    `firestore:"updatedAt"`
	SyncedAt   time.Time    `firestore:"syncedAt"`
	FromNumber string       `firestore:"fromNumber"`
	Elements   []DocElement `firestore:"elements"`
}

func (u *UserDoc) FromMap(data map[string]any) error {
	if val, ok := data["createdAt"]; ok {
		if cval, ok := val.(time.Time); ok {
			u.CreatedAt = cval
		}
	}
	if val, ok := data["updatedAt"]; ok {
		if cval, ok := val.(time.Time); ok {
			u.UpdatedAt = cval
		}
	}
	if val, ok := data["syncedAt"]; ok {
		if cval, ok := val.(time.Time); ok {
			u.SyncedAt = cval
		}
	}
	if val, ok := data["fromNumber"]; ok {
		if cval, ok := val.(string); ok {
			u.FromNumber = cval
		}
	}
	if val, ok := data["elements"]; ok {
		if b, err := json.Marshal(val); err != nil {
			return err
		} else {
			var cval []DocElement
			json.Unmarshal(b, &cval)
			u.Elements = cval
		}
	}
	return nil
}

func ParseElementMessage(content string) (ElementMessage, error) {
	message := ElementMessage{}
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
	element, found := Paragraph, false
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

func MergeElementDoc(doc UserDoc, message ElementMessage, data map[string]string) UserDoc {
	doc.UpdatedAt = time.Now()
	doc.Elements = append(doc.Elements, DocElement{
		ElementType:    message.Element.ToString(),
		ElementContent: message.ContentToData(),
	})
	doc.FromNumber = data["From"]
	return doc
}

func DoHash(b []byte) (string, error) {
	hasher := sha256.New()
	if _, err := hasher.Write(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil)), nil
}

func MergeAutoDoc(doc UserDoc, message string, data map[string]string, initialSync bool) (UserDoc, error) {
	var err error
	doc.UpdatedAt = time.Now()
	if !initialSync {
		doc.Elements = append(doc.Elements, DocElement{
			ElementType:    "p",
			ElementContent: message,
		})
	} else {
		doc.SyncedAt = time.Now()
		doc.Elements = []DocElement{}
	}
	if doc.FromNumber, err = DoHash([]byte(data["From"])); err != nil {
		return doc, err
	}
	return doc, nil
}

// HandleUserDoc attempts to link the inbound message with a document and perform cleanup.
func HandleUserDoc(db *Database, messageBody string, fromNumber string) (string, *UserDoc, bool, error) {
	userDoc := new(UserDoc)
	fromHash, err := DoHash([]byte(fromNumber))
	if err != nil {
		return "", userDoc, false, err
	}
	queries := []Query{
		{
			path:  "fromNumber",
			op:    "==",
			value: fromHash,
		},
	}
	existingDocs, err := db.QueryDocs("inbound/pager/sessions", queries, OrderBy{"createdAt", firestore.Desc})
	if err != nil {
		return "", userDoc, false, err
	}
	words := strings.Split(messageBody, " ")
	explicitDocPath := ""
	var explicitDocData map[string]any
	if len(words) > 0 && len(words[0]) == 6 {
		if err := db.GetDoc(fmt.Sprintf("inbound/pager/sessions/%s", words[0]), &explicitDocData); err != nil && status.Code(err) != codes.NotFound {
			return "", userDoc, false, err
		} else if status.Code(err) != codes.NotFound {
			explicitDocPath = fmt.Sprintf("inbound/pager/sessions/%s", words[0])
		}
	}

	var docFromString string
	if docFrom, ok := explicitDocData["fromNumber"]; !ok {
		docFromString = ""
	} else if docFromString, ok = docFrom.(string); !ok {
		return "", userDoc, false, errors.New("Malformed document.")
	}
	// We have an explicit match and it's unclaimed or claimed by the caller.
	if explicitDocPath != "" && (docFromString == "" || docFromString == fromHash) {
		for _, d := range existingDocs {
			if val, ok := d["docPath"]; ok && val.(string) != explicitDocPath {
				if err := db.DeleteDoc(val.(string)); err != nil {
					return "", userDoc, false, err
				}
			}
		}
		err := userDoc.FromMap(explicitDocData)
		// If the doc has not been claimed all we do is claim it for the caller... if it has, we append the full body.
		return explicitDocPath, userDoc, docFromString == "", err
	}
	if len(existingDocs) > 1 {
		for _, doc := range existingDocs[1:] {
			if val, ok := doc["docPath"]; !ok {
				continue
			} else if err := db.DeleteDoc(val.(string)); err != nil {
				return "", userDoc, false, err
			}
		}
	}
	// There is no valid explicit doc and we do not recognize the caller.
	if len(existingDocs) == 0 {
		return "", userDoc, false, errors.New("No doc found.")
	}
	// There is no valid explicit doc but we have latest claimed doc for the caller.
	err = userDoc.FromMap(existingDocs[0])
	return existingDocs[0]["docPath"].(string), userDoc, false, err
}
