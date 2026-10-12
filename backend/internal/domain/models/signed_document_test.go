package models

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testRSAPEMs(t *testing.T) (privPEM, pubPEM string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	privDER, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)
	privPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER}))
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	require.NoError(t, err)
	pubPEM = string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))
	return privPEM, pubPEM
}

func sampleDocument() *Document {
	doc := NewDocument("d1")
	title := NewField("title")
	title.Value = append(title.Value, NewValue(ValueTypeText).WithString("Rechnung"))
	doc.Fields.Add(title)
	doc.Properties["source"] = *NewValue(ValueTypeText).WithString("inbox")
	return doc
}

func TestSignDocumentAndCheck(t *testing.T) {
	priv, pub := testRSAPEMs(t)
	doc := sampleDocument()

	signed, err := SignDocument(doc, "store-1", priv)
	require.NoError(t, err)
	require.NotNil(t, signed)
	assert.Equal(t, "d1", signed.ID)
	assert.Equal(t, FormatVersion, signed.FormatVersion)
	assert.Equal(t, "store-1", signed.GetKeyID())
	assert.Equal(t, SigAlgRS256, signed.Signature.Algorithm)
	assert.NotEmpty(t, signed.Signature.Signature)
	assert.NoError(t, signed.Check(pub))
	assert.NoError(t, signed.Check(priv))
}

func TestDocumentSignMethod(t *testing.T) {
	priv, pub := testRSAPEMs(t)
	signed, err := sampleDocument().Sign("k1", priv)
	require.NoError(t, err)
	assert.Equal(t, "k1", signed.GetKeyID())
	assert.NoError(t, signed.Check(pub))
}

func TestCheckWithLooksUpKey(t *testing.T) {
	priv, pub := testRSAPEMs(t)
	otherPriv, _ := testRSAPEMs(t)
	signed, err := SignDocument(sampleDocument(), "k-main", priv)
	require.NoError(t, err)

	keys := map[string]string{"k-main": pub, "other": otherPriv}
	assert.NoError(t, signed.CheckWith(func(keyID string) (string, error) {
		keyPEM, ok := keys[keyID]
		if !ok {
			return "", ErrInvalidKey
		}
		return keyPEM, nil
	}))
	assert.ErrorIs(t, signed.CheckWith(nil), ErrInvalidKey)
	assert.ErrorIs(t, signed.CheckWith(func(string) (string, error) { return "", ErrInvalidKey }), ErrInvalidKey)
}

func TestCheckRejectsTamperAndWrongKey(t *testing.T) {
	priv, pub := testRSAPEMs(t)
	_, otherPub := testRSAPEMs(t)
	signed, err := SignDocument(sampleDocument(), "k1", priv)
	require.NoError(t, err)

	signed.Properties["source"] = *NewValue(ValueTypeText).WithString("changed")
	assert.ErrorIs(t, signed.Check(pub), ErrInvalidSignature)

	fresh, err := SignDocument(sampleDocument(), "k1", priv)
	require.NoError(t, err)
	assert.ErrorIs(t, fresh.Check(otherPub), ErrInvalidSignature)
}

func TestSignedDocumentJSONCompatibleWithDocument(t *testing.T) {
	priv, pub := testRSAPEMs(t)
	signed, err := SignDocument(sampleDocument(), "k1", priv)
	require.NoError(t, err)

	raw, err := json.Marshal(signed)
	require.NoError(t, err)
	assert.Contains(t, string(raw), `"signature"`)
	assert.Contains(t, string(raw), `"keyId":"k1"`)

	var asDoc Document
	require.NoError(t, json.Unmarshal(raw, &asDoc))
	assert.Equal(t, "d1", asDoc.ID)
	require.Len(t, asDoc.Fields, 1)
	assert.Equal(t, "title", asDoc.Fields[0].Name)
	assert.Equal(t, "inbox", asDoc.Properties["source"].Value)

	var roundtrip SignedDocument
	require.NoError(t, json.Unmarshal(raw, &roundtrip))
	assert.Equal(t, "k1", roundtrip.GetKeyID())
	assert.NoError(t, roundtrip.Check(pub))
}

func TestCanonicalDocumentJSONIsRFC8785(t *testing.T) {
	doc := NewDocument("d1")
	doc.Properties["b"] = *NewValue(ValueTypeInt).WithInt(2)
	doc.Properties["a"] = *NewValue(ValueTypeInt).WithInt(1)

	canonical, err := CanonicalDocumentJSON(doc)
	require.NoError(t, err)
	text := string(canonical)
	assert.NotContains(t, text, " ")
	assert.True(t, strings.HasPrefix(text, "{"))
	assert.Less(t, strings.Index(text, `"fields"`), strings.Index(text, `"formatVersion"`))
	assert.Less(t, strings.Index(text, `"formatVersion"`), strings.Index(text, `"id"`))
	assert.Less(t, strings.Index(text, `"id"`), strings.Index(text, `"properties"`))
	assert.Less(t, strings.Index(text, `"a"`), strings.Index(text, `"b"`))

	again, err := CanonicalDocumentJSON(doc)
	require.NoError(t, err)
	assert.Equal(t, canonical, again)
}

func TestSignDocumentRejectsBadInput(t *testing.T) {
	priv, _ := testRSAPEMs(t)
	_, err := SignDocument(nil, "k1", priv)
	assert.ErrorIs(t, err, ErrNilDocument)

	_, err = SignDocument(sampleDocument(), "", priv)
	assert.ErrorIs(t, err, ErrMissingKeyID)

	_, err = SignDocument(sampleDocument(), "k1", "not-a-key")
	assert.ErrorIs(t, err, ErrInvalidKey)

	assert.ErrorIs(t, (*SignedDocument)(nil).Check(priv), ErrMissingSignature)
	unsigned := &SignedDocument{Document: *sampleDocument()}
	assert.ErrorIs(t, unsigned.Check(priv), ErrMissingSignature)
	assert.Empty(t, (*SignedDocument)(nil).GetKeyID())
}
