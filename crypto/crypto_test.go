package crypto

import (
	"bytes"
	"fmt"
	"testing"
)

func TestCopyEncryptDecrypt(t *testing.T) {
	payload := "secret message"
	src := bytes.NewReader([]byte(payload))
	dst := new(bytes.Buffer)
	key := NewEncryptionKey()

	_, err := CopyEncrypt(key, src, dst)
	if err != nil {
		t.Errorf("Failed to copyEncrypt: %v", err)
	}
	fmt.Println(len(payload))
	fmt.Println(len(dst.String()))
	out := new(bytes.Buffer)

	nw, err := CopyDecrypt(key, dst, out)
	if err != nil {
		t.Errorf("Failed to copyDecrypt: %v", err)
	}

	if nw != 16+len(payload) {
		t.Errorf("Decryption Failed: Expected: %d Actual: %d", len(payload), nw)
	}

	if !bytes.Equal(out.Bytes(), []byte(payload)) {
		t.Errorf("Decryption Failed: Expected: %s Actual: %s", payload, out)
	}

}
