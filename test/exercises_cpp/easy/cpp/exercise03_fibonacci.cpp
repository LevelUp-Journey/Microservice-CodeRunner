// Exercise 03: fibonacci
// Description: return fibonacci(n) where fib(0)=0, fib(1)=1
// Tests:
// 1) input=0 => 0
// 2) input=5 => 5
// 3) input=10 => 55

int fibonacci(int n) {
    if (n <= 0) return 0;
    if (n == 1) return 1;
    int a = 0, b = 1;
    for (int i = 2; i <= n; ++i) {
        int c = a + b;
        a = b;
        b = c;
    }
    return b;
}
