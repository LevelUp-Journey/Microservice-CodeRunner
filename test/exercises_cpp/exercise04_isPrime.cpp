// Exercise 04: isPrime
// Description: return 1 if n is prime, 0 otherwise (n >= 2)
// Tests:
// 1) input=2 => 1
// 2) input=15 => 0
// 3) input=17 => 1

int isPrime(int n) {
    if (n <= 1) return 0;
    if (n <= 3) return 1;
    if (n % 2 == 0) return 0;
    for (int i = 3; (long long)i * i <= n; i += 2) {
        if (n % i == 0) return 0;
    }
    return 1;
}
