// Exercise 09: pow2
// Description: return 2^n (assume n >= 0 and small enough to fit in int)
// Tests:
// 1) input=0 => 1
// 2) input=5 => 32
// 3) input=10 => 1024

int pow2(int n) {
    int res = 1;
    while (n-- > 0) res *= 2;
    return res;
}
