// Exercise 10: smallestDivisor
// Description: return smallest divisor > 1 of n (n >= 2). If n is prime, return n.
// Tests:
// 1) input=2 => 2
// 2) input=15 => 3
// 3) input=17 => 17

int smallestDivisor(int n) {
    if (n % 2 == 0) return 2;
    for (int i = 3; (long long)i * i <= n; i += 2) {
        if (n % i == 0) return i;
    }
    return n;
}
