#include <string>
#include <stack>

bool isValidParentheses(const std::string& s) {
    std::stack<char> stk;

    for (char c : s) {
        if (c == '(' || c == '{' || c == '[') {
            stk.push(c);
        } else {
            if (stk.empty()) return false;

            char top = stk.top();
            stk.pop();

            if ((c == ')' && top != '(') ||
                (c == '}' && top != '{') ||
                (c == ']' && top != '[')) {
                return false;
            }
        }
    }

    return stk.empty();
}