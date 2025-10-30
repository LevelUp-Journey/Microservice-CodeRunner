#include <vector>
#include <algorithm>

std::vector<int> mergeSortedArrays(const std::vector<int>& nums1, const std::vector<int>& nums2) {
    std::vector<int> result;
    result.reserve(nums1.size() + nums2.size());

    size_t i = 0, j = 0;

    while (i < nums1.size() && j < nums2.size()) {
        if (nums1[i] <= nums2[j]) {
            result.push_back(nums1[i++]);
        } else {
            result.push_back(nums2[j++]);
        }
    }

    while (i < nums1.size()) {
        result.push_back(nums1[i++]);
    }

    while (j < nums2.size()) {
        result.push_back(nums2[j++]);
    }

    return result;
}