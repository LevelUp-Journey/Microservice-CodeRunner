// Exercise 06: reverseInt
// Description: return integer with digits reversed (preserve sign for negative)
// Tests:
// 1) input=123 => 321
// 2) input=-120 => -21
// 3) input=0 => 0

int reverseInt(int n) {
    int sign = n < 0 ? -1 : 1;
    if (n < 0) n = -n;
    int rev = 0;
    while (n > 0) {
        rev = rev * 10 + (n % 10);
        n /= 10;
    }
    return sign * rev;
}
