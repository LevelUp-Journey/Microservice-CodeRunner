// Exercise 02: factorial
// Description: return n! for n >= 0 (0! = 1)
// Tests:
// 1) input=0 => 1
// 2) input=5 => 120
// 3) input=7 => 5040

int factorial(int n) {
    if (n <= 1) return 1;
    int res = 1;
    for (int i = 2; i <= n; i++) res *= i;
    return res;
}
