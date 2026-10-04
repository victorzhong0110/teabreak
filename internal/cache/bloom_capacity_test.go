package cache

import (
	"math"
	"testing"
)

// 容量层实测：生产用 meta.go:75 的 NewBloom(1<<20, 4)。
// 这个 benchmark 直接输出假阳性率，覆盖从饱和到完全失效的区间。
//
// 参考：Bloom 最优 k = (m/n) * ln2。m=1<<20 时：
//   n=10万  -> m/n=10.5 -> k≈7.3（当前 k=4 明显偏小）
//   n=50万  -> m/n=2.1  -> k≈1.5（k=4 已过饱和）
//   n=100万 -> m/n=1.05 -> k≈0.7（任何 k≥1 都不够）

// 连续 id 更接近生产：券 id 自增，不是随机数。
func benchSeqIDs(n int) []int64 {
	ids := make([]int64, n)
	for i := range ids {
		ids[i] = int64(i + 1)
	}
	return ids
}

// TestBloomFalsePositiveRate 报告当前实现（m=1<<20, k=4）在各负载下的假阳性率。
// 这是诊断工具，不是通过/失败断言——它把「容量够不够」变成可读数字。
func TestBloomFalsePositiveRate(t *testing.T) {
	const probe = 200000
	for _, n := range []int{1000, 10000, 100000, 500000, 1000000} {
		bl := NewBloom(1<<20, 4)
		for _, id := range benchSeqIDs(n) {
			bl.Add(id)
		}
		var fp int
		for i := 0; i < probe; i++ {
			// 探针 id 取在已插入区间之外
			if bl.MayContain(int64(10_000_000 + i)) {
				fp++
			}
		}
		// 理论值：p ≈ (1 - e^(-kn/m))^k
		kn := float64(4 * n)
		m := float64(1 << 20)
		theory := math.Pow(1-math.Exp(-kn/m), 4)
		t.Logf("n=%-8d 实测FPR=%.5f  理论FPR=%.5f  最优k≈%.1f",
			n, float64(fp)/probe, theory, kn/m*math.Ln2)
	}
}

// TestBloomOptimalK 对比不同 k 在生产容量下的假阳性率。
func TestBloomOptimalK(t *testing.T) {
	const probe = 200000
	for _, n := range []int{10000, 100000, 500000} {
		var line string
		for k := 1; k <= 8; k++ {
			bl := NewBloom(1<<20, k)
			for _, id := range benchSeqIDs(n) {
				bl.Add(id)
			}
			var fp int
			for i := 0; i < probe; i++ {
				if bl.MayContain(int64(10_000_000 + i)) {
					fp++
				}
			}
			line += " k=" + itoa(k) + ":" + ftoa(float64(fp)/probe*100) + "%"
		}
		t.Logf("n=%-7d%s", n, line)
	}
}

// TestBloomScaledM 按券量上探 m，验证「加大容量」这条路的代价。
func TestBloomScaledM(t *testing.T) {
	const probe = 200000
	cases := []struct {
		bits, k int
	}{
		{1 << 20, 4}, // 当前生产配置
		{1 << 20, 7},
		{1 << 22, 4},
		{1 << 22, 7},
		{1 << 24, 7},
		{1 << 24, 8},
	}
	for _, c := range cases {
		var out string
		for _, n := range []int{10000, 100000, 500000, 1000000} {
			bl := NewBloom(c.bits, c.k)
			for _, id := range benchSeqIDs(n) {
				bl.Add(id)
			}
			var fp int
			for i := 0; i < probe; i++ {
				if bl.MayContain(int64(10_000_000 + i)) {
					fp++
				}
			}
			out += "  n=" + itoa(n) + ":" + ftoa(float64(fp)/probe*100) + "%"
		}
		t.Logf("m=1<<%d k=%d  内存=%dKB%s", log2(c.bits), c.k, c.bits/8/1024, out)
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b [20]byte
	p := len(b)
	for i > 0 {
		p--
		b[p] = byte('0' + i%10)
		i /= 10
	}
	return string(b[p:])
}

func ftoa(f float64) string {
	// 保留两位小数，够用且不引 fmt 依赖
	whole := int(f)
	frac := int((f-float64(whole))*100 + 0.5)
	return itoa(whole) + "." + itoa2(frac)
}

func itoa2(i int) string {
	return string([]byte{byte('0' + i/10), byte('0' + i%10)})
}

func log2(n int) int {
	r := 0
	for n > 1 {
		n >>= 1
		r++
	}
	return r
}
