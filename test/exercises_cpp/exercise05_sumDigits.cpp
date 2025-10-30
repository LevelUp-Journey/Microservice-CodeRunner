// Exercise 05: sumDigits
// Description: return sum of digits of n (n >= 0)
// Tests:
// 1) input=0 => 0
// 2) input=123 => 6
// 3) input=1009 => 10

int sumDigits(int n) {
    if (n < 0) n = -n;
    int s = 0;
    while (n > 0) {
        s += n % 10;
        n /= 10;
    }
    return s;
}
