package idgen

import (
	"testing"
)

// 雪花 id 生成是下单路径的每请求调用。
//
// 注意：包注释已写明「代价是时钟大幅回拨时会阻塞等到时间追上」，
// 序列耗尽后的等待属于**设计取舍**，不是缺陷。此处只建立测量基线，
// 不建议以降低 CPU 为由把它改成 sleep（那会增加墙钟延迟，取舍属于 owner 决策）。

func BenchmarkNext(b *testing.B) {
	g := New(7)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = g.Next()
	}
}

func BenchmarkNextParallel(b *testing.B) {
	g := New(7)
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = g.Next()
		}
	})
}
