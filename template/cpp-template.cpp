// Start Test 
#define DOCTEST_CONFIG_IMPLEMENT_WITH_MAIN
// Solution - Start
#include "doctest.h"
#include <iostream>
namespace std;

int fibonacci(int n) {
    if (n < 0) throw invalid_argument("n debe ser >= 0");
    if (n == 0) return 0;
    if (n == 1) return 1;
    return fibonacci(n - 1) + fibonacci(n - 2);
}
// Solution - End

// Tests - Start
TEST_CASE("Test_id") {
    CHECK(fibonacci(0) == 0);
    CHECK(fibonacci(1) == 1);
    CHECK(fibonacci(2) == 1);
    CHECK(fibonacci(3) == 2);
    CHECK(fibonacci(5) == 5);
    CHECK(fibonacci(10) == 55);
    // CHECK(function_name(input) == expected_output);
}
// Tests - End
