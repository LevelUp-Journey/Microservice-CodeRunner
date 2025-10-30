// Exercise 07: countSetBits
// Description: return number of set bits in binary representation of n (n >= 0)
// Tests:
// 1) input=0 => 0
// 2) input=7 => 3
// 3) input=1023 => 10

int countSetBits(int n) {
    if (n < 0) n = -n;
    int cnt = 0;
    while (n) {
        cnt += n & 1;
        n >>= 1;
    }
    return cnt;
}
