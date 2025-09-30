#define DOCTEST_CONFIG_IMPLEMENT_WITH_MAIN
#include "doctest.h"
#include <iostream>

// Función Fibonacci iterativa
int fibonacci(int position) {
    if (position == 0) return 0;
    if (position == 1) return 1;
    int a = 0, b = 1;
    for (int i = 2; i <= position; ++i) {
        int temp = a + b;
        a = b;
        b = temp;
    }
    return b;
}

// -------------------- TESTS --------------------
TEST_CASE("Fibonacci básicos") {
    CHECK(fibonacci(0) == 0); // input 0, output 0 (Esto viene en el request : input=0, expectedOutput=0)
    CHECK(fibonacci(1) == 1);
    CHECK(fibonacci(2) == 1);
    CHECK(fibonacci(3) == 2);
    CHECK(fibonacci(4) == 3);
    CHECK(fibonacci(5) == 5);
}

TEST_CASE("Fibonacci más avanzados") {
    CHECK(fibonacci(6) == 8);
    CHECK(fibonacci(7) == 13);
    CHECK(fibonacci(10) == 55);
    CHECK(fibonacci(15) == 610);
    CHECK(fibonacci(20) == 6765);
}
