double sqrt_newton(double x, int iterations) {
    if (x < 0) return -1;
    double guess = x / 2.0;
    for (int i = 0; i < iterations; i++) {
        guess = (guess + x / guess) / 2.0;
    }
    return guess;
}