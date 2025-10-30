#include <vector>
#include <algorithm>

int longestIncreasingSubsequence(const std::vector<int>& nums) {
    if (nums.empty()) return 0;

    std::vector<int> dp(nums.size(), 1);
    int max_length = 1;

    for (size_t i = 1; i < nums.size(); ++i) {
        for (size_t j = 0; j < i; ++j) {
            if (nums[i] > nums[j]) {
                dp[i] = std::max(dp[i], dp[j] + 1);
            }
        }
        max_length = std::max(max_length, dp[i]);
    }

    return max_length;
}