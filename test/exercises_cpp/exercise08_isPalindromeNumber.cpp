// Exercise 08: isPalindromeNumber
// Description: return 1 if integer n is palindrome (ignoring sign), else 0
// Tests:
// 1) input=121 => 1
// 2) input=-121 => 1
// 3) input=123 => 0

int isPalindromeNumber(int n) {
    int sign = n < 0 ? -1 : 1;
    if (n < 0) n = -n;
    int original = n;
    int rev = 0;
    while (n > 0) {
        rev = rev * 10 + (n % 10);
        n /= 10;
    }
    return rev == original ? 1 : 0;
}
