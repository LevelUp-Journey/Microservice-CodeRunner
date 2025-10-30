#include <cmath>

double integrate(double a, double b, int n) {
    double h = (b - a) / n;
    double sum = 0.5 * (a*a + b*b);
    for (int i = 1; i < n; i++) {
        double x = a + i * h;
        sum += x * x;
    }
    return sum * h;
}