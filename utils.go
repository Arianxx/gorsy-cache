package gorsy_cache

func min(nums ...int) int {
	ans := nums[0]
	for _, v := range nums {
		if v < ans {
			ans = v
		}
	}
	return ans
}

func max(nums ...int) int {
	ans := nums[0]
	for _, v := range nums {
		if v > ans {
			ans = v
		}
	}
	return ans
}
