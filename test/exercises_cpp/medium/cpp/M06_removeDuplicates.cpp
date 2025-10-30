#include <vector>
#include <algorithm>

int removeDuplicates(std::vector<int>& nums) {
    if (nums.empty()) return 0;

    size_t write_index = 1;

    for (size_t read_index = 1; read_index < nums.size(); ++read_index) {
        if (nums[read_index] != nums[read_index - 1]) {
            nums[write_index++] = nums[read_index];
        }
    }

    return write_index;
}