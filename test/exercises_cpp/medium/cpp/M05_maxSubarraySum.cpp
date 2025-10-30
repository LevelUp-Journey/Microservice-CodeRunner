#include <vector>
#include <algorithm>
#include <limits>

int maxSubarraySum(const std::vector<int>& nums) {
    if (nums.empty()) return 0;

    int max_current = nums[0];
    int max_global = nums[0];

    for (size_t i = 1; i < nums.size(); ++i) {
        max_current = std::max(nums[i], max_current + nums[i]);
        max_global = std::max(max_global, max_current);
    }

    return max_global;
}