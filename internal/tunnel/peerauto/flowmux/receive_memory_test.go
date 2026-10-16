package smux

import (
	"bytes"
	"testing"
)

func TestTinyFramesKeepReceiveMetadataBoundedAndOrdered(t *testing.T) {
	a, _ := flowPair(t, 2)
	if err := a.EnableAuthenticatedFlowControl(); err != nil {
		t.Fatal(err)
	}
	s := newStream(3, 32768, a)
	const count = 128 << 10
	want := make([]byte, count)
	for i := range want {
		want[i] = byte(i % 251)
		h := defaultAllocator.Get(1)
		(*h)[0] = want[i]
		if !s.pushBytes(h) {
			t.Fatal("valid tiny frames exceeded receive credit")
		}
	}
	if s.bufferRing.len() > 34 {
		t.Fatalf("128KiB of tiny frames retained %d allocator/ring entries", s.bufferRing.len())
	}
	var out []byte
	for s.bufferRing.len() > 0 {
		b, h, _ := s.bufferRing.pop()
		out = append(out, b...)
		a.returnTokens(len(b))
		defaultAllocator.Put(h)
	}
	if !bytes.Equal(out, want) || a.BufferedBytes() != 0 {
		t.Fatal("coalescing changed stream bytes or payload accounting")
	}
}

func TestReceiveCoalescingPreservesPartialHeadAndAllocatorCapacity(t *testing.T) {
	a, _ := flowPair(t, 2)
	if err := a.EnableAuthenticatedFlowControl(); err != nil {
		t.Fatal(err)
	}
	s := newStream(3, 32768, a)
	h := defaultAllocator.Get(3)
	copy(*h, []byte{1, 2, 3})
	if !s.pushBytes(h) {
		t.Fatal("initial frame rejected")
	}
	read := make([]byte, 2)
	n, recycled := s.bufferRing.consumeFront(read)
	if n != 2 || recycled != nil || !bytes.Equal(read, []byte{1, 2}) {
		t.Fatal("partial head fixture failed")
	}
	s.numRead += 2
	a.returnTokens(2)
	h = defaultAllocator.Get(4094)
	payload := bytes.Repeat([]byte{7}, 4094)
	copy(*h, payload)
	if !s.pushBytes(h) {
		t.Fatal("valid tail rejected")
	}
	var out []byte
	for s.bufferRing.len() > 0 {
		b, head, _ := s.bufferRing.pop()
		if cap(*head) != 4096 {
			t.Fatalf("allocator head capacity changed to %d", cap(*head))
		}
		out = append(out, b...)
		a.returnTokens(len(b))
		defaultAllocator.Put(head)
	}
	if !bytes.Equal(out, append([]byte{3}, payload...)) || a.BufferedBytes() != 0 {
		t.Fatal("partial-head append corrupted data or retained payload")
	}
}
