package checksum

import "testing"

func TestSHA256(t *testing.T) {
	got := SHA256([]byte("abc"))
	want := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got != want {
		t.Fatalf("SHA256 mismatch: got %s want %s", got, want)
	}
}
