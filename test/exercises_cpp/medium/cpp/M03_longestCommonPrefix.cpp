#include <string>
#include <vector>
#include <algorithm>

std::string longestCommonPrefix(const std::vector<std::string>& strs) {
    if (strs.empty()) return "";

    std::string prefix = strs[0];

    for (size_t i = 1; i < strs.size(); ++i) {
        size_t j = 0;
        while (j < prefix.size() && j < strs[i].size() && prefix[j] == strs[i][j]) {
            ++j;
        }
        prefix = prefix.substr(0, j);
        if (prefix.empty()) return "";
    }

    return prefix;
}