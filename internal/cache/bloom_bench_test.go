package cache

import (
	"math/rand"
	"testing"
)

// 基线 benchmark：用于 minimax 自动优化建议的实测数据。
// 目标路径是线上热点：Bloom 挡不存在券 id，每次请求都要 Add/MayContain。

func benchIDs(n int) []int64 {
	r := rand.New(rand.NewSource(42))
	ids := make([]int64, n)
	for i := range ids {
		ids[i] = r.Int63n(1 << 40)
	}
	return ids
}

func BenchmarkBloomMayContain(b *testing.B) {
	ids := benchIDs(1024)
	bl := NewBloom(1<<20, 4)
	for _, id := range ids {
		bl.Add(id)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bl.MayContain(ids[i&1023])
	}
}

func BenchmarkBloomAdd(b *testing.B) {
	ids := benchIDs(1024)
	bl := NewBloom(1<<20, 4)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bl.Add(ids[i&1023])
	}
}

func BenchmarkBloomMixed(b *testing.B) {
	ids := benchIDs(1024)
	bl := NewBloom(1<<20, 4)
	for _, id := range ids {
		bl.Add(id)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		id := ids[i&1023]
		bl.MayContain(id)
		if i%8 == 0 {
			bl.Add(id)
		}
	}
}

func BenchmarkHashPair(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		h1, h2 := hashPair(int64(i))
		_, _ = h1, h2
	}
}

// 并发读：线上多 goroutine 一起查 bloom，看 RWMutex 在高并发下的成本。
func BenchmarkBloomMayContainParallel(b *testing.B) {
	ids := benchIDs(1024)
	bl := NewBloom(1<<20, 4)
	for _, id := range ids {
		bl.Add(id)
	}
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			bl.MayContain(ids[i&1023])
			i++
		}
	})
}

// 闸门 1 补测：生产券 id 是自增连续整数，不是随机数。
// 上一版 benchmark 用随机 id，对新哈希有利；这里用连续 id 复测。
func BenchmarkBloomMayContainSeq(b *testing.B) {
	const n = 100000
	bl := NewBloom(1<<20, 4)
	for i := 0; i < n; i++ {
		bl.Add(int64(i + 1))
	}
	// 探针取远端随机 id，模拟 meta.go:116 挡的「不存在的券 id」
	probes := make([]int64, 1024)
	r := rand.New(rand.NewSource(7))
	for i := range probes {
		probes[i] = r.Int63n(1<<40) + 50_000_000
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bl.MayContain(probes[i&1023])
	}
}

// 闸门 1 补测：串行复现并发数字（纪律陷阱 B）。
func BenchmarkBloomMayContainSerialLong(b *testing.B) {
	const n = 100000
	bl := NewBloom(1<<20, 4)
	for i := 0; i < n; i++ {
		bl.Add(int64(i + 1))
	}
	probes := make([]int64, 1024)
	r := rand.New(rand.NewSource(7))
	for i := range probes {
		probes[i] = r.Int63n(1<<40) + 50_000_000
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		bl.MayContain(probes[i&1023])
	}
}
