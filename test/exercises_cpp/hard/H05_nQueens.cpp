#include <vector>
#include <string>

class Solution {
private:
    std::vector<std::vector<std::string>> result;
    std::vector<std::string> board;
    int n;

    bool isSafe(int row, int col) {
        // Check column
        for (int i = 0; i < row; ++i) {
            if (board[i][col] == 'Q') return false;
        }

        // Check upper-left diagonal
        for (int i = row - 1, j = col - 1; i >= 0 && j >= 0; --i, --j) {
            if (board[i][j] == 'Q') return false;
        }

        // Check upper-right diagonal
        for (int i = row - 1, j = col + 1; i >= 0 && j < n; --i, ++j) {
            if (board[i][j] == 'Q') return false;
        }

        return true;
    }

    void solveNQueens(int row) {
        if (row == n) {
            result.push_back(board);
            return;
        }

        for (int col = 0; col < n; ++col) {
            if (isSafe(row, col)) {
                board[row][col] = 'Q';
                solveNQueens(row + 1);
                board[row][col] = '.';
            }
        }
    }

public:
    std::vector<std::vector<std::string>> solveNQueens(int n) {
        this->n = n;
        board = std::vector<std::string>(n, std::string(n, '.'));
        result.clear();
        solveNQueens(0);
        return result;
    }
};