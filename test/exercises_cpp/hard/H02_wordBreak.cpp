#include <string>
#include <unordered_set>
#include <vector>

bool wordBreak(const std::string& s, const std::vector<std::string>& wordDict) {
    std::unordered_set<std::string> wordSet(wordDict.begin(), wordDict.end());
    std::vector<bool> dp(s.size() + 1, false);
    dp[0] = true;

    for (size_t i = 1; i <= s.size(); ++i) {
        for (size_t j = 0; j < i; ++j) {
            if (dp[j] && wordSet.count(s.substr(j, i - j))) {
                dp[i] = true;
                break;
            }
        }
    }

    return dp[s.size()];
}