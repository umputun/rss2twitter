package xpp_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/mmcdole/goxpp"
	"github.com/stretchr/testify/assert"
)

func TestEventName(t *testing.T) {
	var eventNameTests = []struct {
		event    xpp.XMLEventType
		expected string
	}{
		{xpp.StartTag, "StartTag"},
		{xpp.EndTag, "EndTag"},
		{xpp.StartDocument, "StartDocument"},
		{xpp.EndDocument, "EndDocument"},
		{xpp.ProcessingInstruction, "ProcessingInstruction"},
		{xpp.Directive, "Directive"},
		{xpp.Comment, "Comment"},
		{xpp.Text, "Text"},
		{xpp.IgnorableWhitespace, "IgnorableWhitespace"},
	}

	p := xpp.XMLPullParser{}
	for _, test := range eventNameTests {
		actual := p.EventName(test.event)
		assert.Equal(t, actual, test.expected, "Expect event name %s did not match actual event name %s.\n", test.expected, actual)
	}
}

func TestSpaceStackSelfClosingTag(t *testing.T) {
	crReader := func(charset string, input io.Reader) (io.Reader, error) {
		return input, nil
	}
	r := bytes.NewBufferString(`<a:y xmlns:a="z"/><x>foo</x>`)
	p := xpp.NewXMLPullParser(r, false, crReader)
	toNextStart(t, p)
	assert.EqualValues(t, map[string]string{"z": "a"}, p.Spaces)
	toNextStart(t, p)
	assert.EqualValues(t, map[string]string{}, p.Spaces)
}

func TestSpaceStackNestedTag(t *testing.T) {
	crReader := func(charset string, input io.Reader) (io.Reader, error) {
		return input, nil
	}
	r := bytes.NewBufferString(`<y xmlns:a="z"><a:x>foo</a:x></y><w></w>`)
	p := xpp.NewXMLPullParser(r, false, crReader)
	toNextStart(t, p)
	assert.EqualValues(t, map[string]string{"z": "a"}, p.Spaces)
	toNextStart(t, p)
	assert.EqualValues(t, map[string]string{"z": "a"}, p.Spaces)
	toNextStart(t, p)
	assert.EqualValues(t, map[string]string{}, p.Spaces)
}

func TestDecodeElementDepth(t *testing.T) {
	crReader := func(charset string, input io.Reader) (io.Reader, error) {
		return input, nil
	}
	r := bytes.NewBufferString(`<root><d2>foo</d2><d2>bar</d2></root>`)
	p := xpp.NewXMLPullParser(r, false, crReader)

	type v struct{}

	// move to root
	p.NextTag()
	assert.Equal(t, "root", p.Name)
	assert.Equal(t, 1, p.Depth)

	// decode first <d2>
	p.NextTag()
	assert.Equal(t, "d2", p.Name)
	assert.Equal(t, 2, p.Depth)
	p.DecodeElement(&v{})

	// decode second <d2>
	p.NextTag()
	assert.Equal(t, "d2", p.Name)
	assert.Equal(t, 2, p.Depth) // should still be 2, not 3
	p.DecodeElement(&v{})
}

func toNextStart(t *testing.T, p *xpp.XMLPullParser) {
	for {
		tok, err := p.NextToken()
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		if tok == xpp.StartTag {
			break
		}
	}
	return
}
