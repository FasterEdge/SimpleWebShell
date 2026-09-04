package route

import (
	"bytes"
	"strings"
	"testing"
)

// TestSecureCompare 校验时序安全比较的正确性: 相等返回 true, 一切不等
// (含仅长度不同) 返回 false, 且不 panic。
func TestSecureCompare(t *testing.T) {
	cases := []struct {
		name string
		a, b string
		want bool
	}{
		{"empty-equal", "", "", true},
		{"equal", "secret-key-123", "secret-key-123", true},
		{"differ", "secret-key-123", "secret-key-124", false},
		{"prefix", "secret-key", "secret-key-123", false},
		{"empty-vs-nonempty", "", "x", false},
		{"unicode-equal", "密钥中文测试", "密钥中文测试", true},
		{"unicode-differ", "密钥中文测试", "密钥中文测试2", false},
	}
	for _, c := range cases {
		if got := secureCompare(c.a, c.b); got != c.want {
			t.Errorf("%s: secureCompare(%q, %q) = %v, want %v", c.name, c.a, c.b, got, c.want)
		}
	}
}

// TestLimitedBufferWithinLimit 校验在限制内写入时完整保留且不标记超限。
func TestLimitedBufferWithinLimit(t *testing.T) {
	b := &limitedBuffer{limit: 1024}
	payload := bytes.Repeat([]byte("a"), 100)
	n, err := b.Write(payload)
	if err != nil {
		t.Fatal(err)
	}
	if n != 100 {
		t.Fatalf("write count = %d, want 100", n)
	}
	if b.exceeded {
		t.Fatal("exceeded set on within-limit write")
	}
	if b.String() != string(payload) {
		t.Fatal("buffer content mismatch")
	}
}

// TestLimitedBufferExceedsLimit 校验超限时: 仅保留前 limit 字节,
// 后续写入丢弃并置 exceeded, 但 Write 返回全量计数 (io.Writer 语义)。
func TestLimitedBufferExceedsLimit(t *testing.T) {
	b := &limitedBuffer{limit: 8}
	first := bytes.Repeat([]byte("x"), 8)
	if _, err := b.Write(first); err != nil {
		t.Fatal(err)
	}
	if b.exceeded {
		t.Fatal("exceeded set at exact limit")
	}
	second := bytes.Repeat([]byte("y"), 100)
	n, err := b.Write(second)
	if err != nil {
		t.Fatal(err)
	}
	if n != 100 {
		t.Fatalf("write count = %d, want 100 (full count semantics)", n)
	}
	if !b.exceeded {
		t.Fatal("exceeded not set after overflow")
	}
	if got := b.String(); got != strings.Repeat("x", 8) {
		t.Fatalf("buffer = %q, want 8 x's only", got)
	}
}

// TestLimitedBufferMultipleWrites 校验多次小写入叠加不丢数据、恰好到限不触发超限。
func TestLimitedBufferMultipleWrites(t *testing.T) {
	b := &limitedBuffer{limit: 12}
	total := 0
	for _, part := range []string{"ab", "cd", "efgh", "ijkl"} {
		n, _ := b.Write([]byte(part))
		total += n
	}
	if total != 12 {
		t.Fatalf("total written = %d, want 12", total)
	}
	if b.exceeded {
		t.Fatal("exceeded set at exact cumulative limit")
	}
	if b.String() != "abcdefghijkl" {
		t.Fatalf("buffer = %q, want abcdefghijkl", b.String())
	}
	// 再来一笔即超限
	if _, err := b.Write([]byte("z")); err != nil {
		t.Fatal(err)
	}
	if !b.exceeded || b.String() != "abcdefghijkl" {
		t.Fatalf("after overflow: exceeded=%v content=%q", b.exceeded, b.String())
	}
}

// TestLimitedBufferEmptyWrite 校验空写入不改变状态。
func TestLimitedBufferEmptyWrite(t *testing.T) {
	b := &limitedBuffer{limit: 4}
	if _, err := b.Write(nil); err != nil {
		t.Fatal(err)
	}
	if b.exceeded || b.String() != "" {
		t.Fatalf("empty write changed state: exceeded=%v content=%q", b.exceeded, b.String())
	}
}
