#include <vector>
#include <algorithm>
#include <limits>

double findMedianSortedArrays(const std::vector<int>& nums1, const std::vector<int>& nums2) {
    if (nums1.size() > nums2.size()) {
        return findMedianSortedArrays(nums2, nums1);
    }

    int m = nums1.size();
    int n = nums2.size();
    int total = m + n;
    int half = (total + 1) / 2;

    int left = 0;
    int right = m;

    while (left <= right) {
        int i = left + (right - left) / 2;
        int j = half - i;

        int nums1_left = (i > 0) ? nums1[i - 1] : std::numeric_limits<int>::min();
        int nums1_right = (i < m) ? nums1[i] : std::numeric_limits<int>::max();
        int nums2_left = (j > 0) ? nums2[j - 1] : std::numeric_limits<int>::min();
        int nums2_right = (j < n) ? nums2[j] : std::numeric_limits<int>::max();

        if (nums1_left <= nums2_right && nums2_left <= nums1_right) {
            if (total % 2 == 1) {
                return std::max(nums1_left, nums2_left);
            } else {
                return (std::max(nums1_left, nums2_left) + std::min(nums1_right, nums2_right)) / 2.0;
            }
        } else if (nums1_left > nums2_right) {
            right = i - 1;
        } else {
            left = i + 1;
        }
    }

    return 0.0;
}